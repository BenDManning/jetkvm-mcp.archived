package jetkvm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type WebRTCProviderOptions struct {
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
}

type WebRTCProvider struct {
	options WebRTCProviderOptions
}

func NewWebRTCProvider(options WebRTCProviderOptions) *WebRTCProvider {
	if options.ConnectTimeout <= 0 {
		options.ConnectTimeout = 15 * time.Second
	}
	if options.RequestTimeout <= 0 {
		options.RequestTimeout = 10 * time.Second
	}
	return &WebRTCProvider{options: options}
}

func (provider *WebRTCProvider) WithSession(ctx context.Context, device DeviceConfig, profile SessionProfile, operation func(Session) error) error {
	if provider == nil || operation == nil {
		return errors.New("WebRTC provider and operation are required")
	}
	connectCtx, cancel := context.WithTimeout(ctx, provider.options.ConnectTimeout)
	defer cancel()
	connected, err := provider.connect(connectCtx, device, profile)
	if err != nil {
		return err
	}
	defer connected.Close()
	return operation(connected)
}

func (provider *WebRTCProvider) connect(ctx context.Context, device DeviceConfig, profile SessionProfile) (_ *connectedSession, returnErr error) {
	if profile != SessionProfileData && profile != SessionProfileVideo {
		return nil, errors.New("invalid session profile")
	}
	httpClient, err := newDeviceHTTPClient(device)
	if err != nil {
		return nil, err
	}
	defer func() {
		if returnErr != nil {
			httpClient.CloseIdleConnections()
		}
	}()
	if _, err := authenticate(ctx, httpClient, device.BaseURL, device.Password); err != nil {
		return nil, err
	}

	connectedCtx, cancel := context.WithCancel(context.Background())
	connected := &connectedSession{
		ctx: connectedCtx, cancel: cancel, httpClient: httpClient, baseURL: device.BaseURL,
	}
	if profile == SessionProfileVideo {
		connected.video = newVideoReceiver()
	}
	defer func() {
		if returnErr != nil {
			connected.Close()
		}
	}()

	var onTrack func(*webrtc.TrackRemote, *webrtc.RTPReceiver)
	if profile == SessionProfileVideo {
		onTrack = connected.handleTrack
	}
	peer, err := newPeerConnection(profile, onTrack)
	if err != nil {
		return nil, fmt.Errorf("%w: peer creation", ErrProtocol)
	}
	connected.peer = peer

	channel, err := peer.CreateDataChannel("rpc", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: RPC data channel", ErrProtocol)
	}
	connected.rpc = newRPCSession(connectedCtx, channel, provider.options.RequestTimeout)
	ready := make(chan struct{})
	var readyOnce sync.Once
	channel.OnOpen(func() { readyOnce.Do(func() { close(ready) }) })
	channel.OnMessage(func(message webrtc.DataChannelMessage) {
		if message.IsString {
			connected.rpc.HandleMessage(message.Data)
		}
	})
	channel.OnClose(cancel)
	channel.OnError(func(error) { cancel() })
	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			cancel()
		}
	})

	signalingURL := endpoint(device.BaseURL, signalingPath)
	if signalingURL.Scheme == "https" {
		signalingURL.Scheme = "wss"
	} else {
		signalingURL.Scheme = "ws"
	}
	connection, response, err := websocket.Dial(ctx, signalingURL.String(), &websocket.DialOptions{HTTPClient: httpClient})
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("%w: signaling", ErrDeviceUnreachable)
	}
	connection.SetReadLimit(maxSignalingFrame)
	connected.signal = &clientSignal{connection: connection}

	peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil || connectedCtx.Err() != nil {
			return
		}
		if err := connected.signal.write(connectedCtx, newCandidateEnvelope(candidate.ToJSON())); err != nil {
			cancel()
		}
	})
	offer, err := peer.CreateOffer(nil)
	if err != nil || peer.SetLocalDescription(offer) != nil {
		return nil, fmt.Errorf("%w: local offer", ErrProtocol)
	}
	envelope, err := newOfferEnvelope(offer)
	if err != nil || connected.signal.write(ctx, envelope) != nil {
		return nil, fmt.Errorf("%w: signaling offer", ErrDeviceUnreachable)
	}
	connected.pumps.Add(1)
	go func() {
		defer connected.pumps.Done()
		connected.readSignaling()
	}()

	select {
	case <-ready:
		return connected, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-connectedCtx.Done():
		return nil, ErrSessionClosed
	}
}

func newDeviceHTTPClient(device DeviceConfig) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport unavailable")
	}
	cloned := transport.Clone()
	cloned.Proxy = nil
	cloned.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: device.InsecureSkipVerify} //nolint:gosec -- explicit per-device opt-in for local appliances
	return &http.Client{
		Transport: cloned,
		Jar:       jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

type connectedSession struct {
	ctx        context.Context
	cancel     context.CancelFunc
	peer       *webrtc.PeerConnection
	signal     *clientSignal
	rpc        *rpcSession
	httpClient *http.Client
	baseURL    url.URL
	video      *videoReceiver

	closeOnce    sync.Once
	pumpMu       sync.Mutex
	pumps        sync.WaitGroup
	closed       bool
	trackStarted bool
}

func (session *connectedSession) Call(ctx context.Context, method string, params any, result any) error {
	if session == nil || session.rpc == nil {
		return ErrSessionClosed
	}
	return session.rpc.Call(ctx, method, params, result)
}

func (session *connectedSession) Upload(ctx context.Context, uploadID string, reader io.Reader, size int64) error {
	if session == nil || session.httpClient == nil || size < 0 || !uploadIDPattern.MatchString(uploadID) {
		return classifyOperationError(ErrInvalidResponse, ToolOutcomeNotSent)
	}
	uploadURL := endpoint(session.baseURL, "/storage/upload")
	query := uploadURL.Query()
	query.Set("uploadId", uploadID)
	uploadURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL.String(), io.LimitReader(reader, size))
	if err != nil {
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	request.Header.Set("Content-Type", "application/octet-stream")
	request.ContentLength = size
	response, err := session.httpClient.Do(request)
	if err != nil {
		return classifyOperationError(fmt.Errorf("%w: media upload", ErrDeviceUnreachable), ToolOutcomeUnknown)
	}
	_, readErr := readBounded(response.Body, 4<<10)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		return classifyOperationError(fmt.Errorf("%w: media upload response", ErrProtocol), ToolOutcomeUnknown)
	}
	return nil
}

func (session *connectedSession) CaptureH264(ctx context.Context) ([]byte, time.Time, error) {
	if session == nil || session.video == nil {
		return nil, time.Time{}, ErrDecoderUnavailable
	}
	return session.video.Capture(ctx)
}

func (session *connectedSession) Close() {
	if session == nil {
		return
	}
	session.closeOnce.Do(func() {
		session.pumpMu.Lock()
		session.closed = true
		session.cancel()
		if session.video != nil {
			session.video.Close()
		}
		session.pumpMu.Unlock()
		if session.rpc != nil {
			session.rpc.Close()
		}
		if session.signal != nil {
			session.signal.close()
		}
		if session.peer != nil {
			_ = session.peer.Close()
		}
		if session.httpClient != nil {
			session.httpClient.CloseIdleConnections()
		}
		session.pumps.Wait()
	})
}

func (session *connectedSession) handleTrack(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	if session == nil || track == nil {
		return
	}
	codec := track.Codec()
	if track.Kind() != webrtc.RTPCodecTypeVideo || !strings.EqualFold(codec.MimeType, webrtc.MimeTypeH264) || codec.ClockRate != 90000 || !strings.Contains(strings.ToLower(codec.SDPFmtpLine), "packetization-mode=1") {
		session.cancel()
		return
	}
	session.pumpMu.Lock()
	if session.closed || session.trackStarted || session.video == nil {
		session.pumpMu.Unlock()
		session.cancel()
		return
	}
	session.trackStarted = true
	session.video.SetPLI(func() error {
		return session.peer.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
	})
	session.pumps.Add(1)
	session.pumpMu.Unlock()
	go func() {
		defer session.pumps.Done()
		for {
			packet, _, err := track.ReadRTP()
			if err != nil {
				if session.ctx.Err() == nil {
					session.cancel()
				}
				return
			}
			session.video.Observe(packet, time.Now().UTC())
		}
	}()
}

func (session *connectedSession) readSignaling() {
	remoteDescriptionSet := false
	queued := make([]webrtc.ICECandidateInit, 0, 4)
	for {
		messageType, payload, err := session.signal.connection.Read(session.ctx)
		if err != nil {
			session.cancel()
			return
		}
		if messageType != websocket.MessageText || len(payload) > maxSignalingFrame {
			continue
		}
		var envelope signalingEnvelope
		if json.Unmarshal(payload, &envelope) != nil {
			continue
		}
		switch envelope.Type {
		case "device-metadata":
			continue
		case "answer":
			if remoteDescriptionSet {
				session.cancel()
				return
			}
			var encoded string
			if json.Unmarshal(envelope.Data, &encoded) != nil {
				session.cancel()
				return
			}
			answer, err := decodeSessionDescription(encoded)
			if err != nil || answer.Type != webrtc.SDPTypeAnswer || session.peer.SetRemoteDescription(answer) != nil {
				session.cancel()
				return
			}
			remoteDescriptionSet = true
			for _, candidate := range queued {
				if session.peer.AddICECandidate(candidate) != nil {
					session.cancel()
					return
				}
			}
			queued = nil
		case "new-ice-candidate":
			var candidate webrtc.ICECandidateInit
			if json.Unmarshal(envelope.Data, &candidate) != nil || candidate.Candidate == "" {
				continue
			}
			if !remoteDescriptionSet {
				if len(queued) < 32 {
					queued = append(queued, candidate)
				}
				continue
			}
			if session.peer.AddICECandidate(candidate) != nil {
				session.cancel()
				return
			}
		}
	}
}

type clientSignal struct {
	connection *websocket.Conn
	writeMu    sync.Mutex
	closeOnce  sync.Once
}

func (signal *clientSignal) write(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload) > maxSignalingFrame {
		return errors.New("signaling message too large")
	}
	signal.writeMu.Lock()
	defer signal.writeMu.Unlock()
	return signal.connection.Write(ctx, websocket.MessageText, payload)
}

func (signal *clientSignal) close() {
	if signal == nil || signal.connection == nil {
		return
	}
	signal.closeOnce.Do(func() { _ = signal.connection.CloseNow() })
}
