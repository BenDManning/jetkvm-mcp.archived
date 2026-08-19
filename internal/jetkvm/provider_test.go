package jetkvm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/pion/webrtc/v4"
)

func TestDeviceHTTPClientDoesNotUseEnvironmentProxy(t *testing.T) {
	client, err := newDeviceHTTPClient(DeviceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("device transport inherited an environment proxy callback")
	}
}

func TestConnectedSessionTreatsDisconnectedPeerAsTerminal(t *testing.T) {
	connectedCtx, cancel := context.WithCancel(context.Background())
	session := &connectedSession{ctx: connectedCtx, cancel: cancel}
	session.handleConnectionState(webrtc.PeerConnectionStateConnected)
	select {
	case <-session.Done():
		t.Fatal("connected peer ended the session")
	default:
	}
	session.handleConnectionState(webrtc.PeerConnectionStateDisconnected)
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("disconnected peer did not end the session")
	}
}

func TestConnectedSessionRecognizedTakeoverIsLatchedBeforeTermination(t *testing.T) {
	connectedCtx, cancel := context.WithCancel(context.Background())
	session := &connectedSession{ctx: connectedCtx, cancel: cancel}
	session.recognizeTakeover()
	if !session.RecognizedTakeover() || !session.SuppressHIDCleanup() {
		t.Fatal("recognized takeover was not latched for owner and HID cleanup")
	}
	select {
	case <-session.Done():
	default:
		t.Fatal("recognized takeover did not terminate the generation")
	}
}

func TestWebRTCConnectorClosesIdleHTTPConnectionsAfterAuthenticationFailure(t *testing.T) {
	closed := make(chan struct{}, 1)
	httpServer := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Length", "0")
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	httpServer.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			select {
			case closed <- struct{}{}:
			default:
			}
		}
	}
	httpServer.Start()
	defer func() {
		httpServer.CloseClientConnections()
		httpServer.Close()
	}()
	base, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	connector := NewWebRTCConnector(WebRTCConnectorOptions{})
	_, err = connector.connect(context.Background(), DeviceConfig{BaseURL: *base}, SessionProfileData)
	if !errors.Is(err, ErrAuthentication) {
		t.Fatalf("error = %v, want authentication failure", err)
	}
	select {
	case <-closed:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("authentication failure left the device HTTP connection open")
	}
}

type webRTCConnectorFixture struct {
	connector             *WebRTCConnector
	device                DeviceConfig
	rpcChannel            chan *webrtc.DataChannel
	loginCalls            atomic.Int32
	pingCalls             atomic.Int32
	deviceCalls           atomic.Int32
	authorizedDeviceCalls atomic.Int32
	offerSDP              atomic.Value
}

func newWebRTCConnectorFixture(t *testing.T) *webRTCConnectorFixture {
	t.Helper()
	fixture := &webRTCConnectorFixture{rpcChannel: make(chan *webrtc.DataChannel, 1)}
	remotePeer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = remotePeer.Close() })

	remotePeer.OnDataChannel(func(channel *webrtc.DataChannel) {
		if channel.Label() != "rpc" {
			return
		}
		channel.OnOpen(func() { fixture.rpcChannel <- channel })
		channel.OnMessage(func(message webrtc.DataChannelMessage) {
			var request struct {
				JSONRPC string `json:"jsonrpc"`
				Method  string `json:"method"`
				ID      uint64 `json:"id"`
			}
			if !message.IsString || json.Unmarshal(message.Data, &request) != nil {
				return
			}
			if request.Method == "ping" {
				fixture.pingCalls.Add(1)
				_ = channel.SendText("{\"jsonrpc\":\"2.0\",\"id\":" + jsonNumber(request.ID) + ",\"result\":\"pong\"}")
			}
		})
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/device", func(writer http.ResponseWriter, request *http.Request) {
		fixture.deviceCalls.Add(1)
		cookie, err := request.Cookie("session")
		if err != nil || cookie.Value != "ok" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		fixture.authorizedDeviceCalls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte("{\"authMode\":\"password\",\"deviceId\":\"test-device\",\"appVersion\":\"test\",\"systemVersion\":\"test\"}"))
	})
	mux.HandleFunc("/auth/login-local", func(writer http.ResponseWriter, request *http.Request) {
		fixture.loginCalls.Add(1)
		http.SetCookie(writer, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte("{\"message\":\"ok\"}"))
	})
	mux.HandleFunc(signalingPath, func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie("session")
		if err != nil || cookie.Value != "ok" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		connection, err := websocket.Accept(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		signal := &testSignal{connection: connection}
		_ = signal.write(request.Context(), map[string]any{"type": "device-metadata", "data": map[string]any{"deviceVersion": "test"}})

		remoteDescriptionSet := false
		var queued []webrtc.ICECandidateInit
		remotePeer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				_ = signal.write(context.Background(), newCandidateEnvelope(candidate.ToJSON()))
			}
		})
		for {
			messageType, payload, err := connection.Read(request.Context())
			if err != nil {
				return
			}
			if messageType != websocket.MessageText {
				continue
			}
			var envelope signalingEnvelope
			if json.Unmarshal(payload, &envelope) != nil {
				continue
			}
			switch envelope.Type {
			case "offer":
				var data struct {
					SD string `json:"sd"`
				}
				if json.Unmarshal(envelope.Data, &data) != nil {
					return
				}
				offer, err := decodeSessionDescription(data.SD)
				if err != nil || remotePeer.SetRemoteDescription(offer) != nil {
					return
				}
				fixture.offerSDP.Store(offer.SDP)
				remoteDescriptionSet = true
				for _, candidate := range queued {
					_ = remotePeer.AddICECandidate(candidate)
				}
				queued = nil
				answer, err := remotePeer.CreateAnswer(nil)
				if err != nil || remotePeer.SetLocalDescription(answer) != nil {
					return
				}
				raw, _ := json.Marshal(answer)
				encoded, _ := json.Marshal(base64.StdEncoding.EncodeToString(raw))
				_ = signal.write(request.Context(), signalingEnvelope{Type: "answer", Data: encoded})
			case "new-ice-candidate":
				var candidate webrtc.ICECandidateInit
				if json.Unmarshal(envelope.Data, &candidate) != nil {
					continue
				}
				if !remoteDescriptionSet {
					queued = append(queued, candidate)
				} else {
					_ = remotePeer.AddICECandidate(candidate)
				}
			}
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	fixture.connector = NewWebRTCConnector(WebRTCConnectorOptions{ConnectTimeout: 10 * time.Second, RequestTimeout: time.Second})
	fixture.device = DeviceConfig{Name: "lab", BaseURL: *base, Password: "correct"}
	return fixture
}

func TestWebRTCConnectedSessionRecognizesFirmwareTakeoverEvent(t *testing.T) {
	fixture := newWebRTCConnectorFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	connected, err := fixture.connector.Connect(ctx, fixture.device)
	if err != nil {
		t.Fatal(err)
	}
	defer connected.Close(context.Background())

	observed, ok := connected.(interface {
		TakeoverDetected() <-chan struct{}
		RecognizedTakeover() bool
	})
	if !ok {
		t.Fatal("connected session does not expose takeover observation")
	}
	var remoteChannel *webrtc.DataChannel
	select {
	case remoteChannel = <-fixture.rpcChannel:
	case <-ctx.Done():
		t.Fatal("firmware-side RPC channel did not open")
	}
	if err := remoteChannel.SendText(`{"jsonrpc":"2.0","method":"otherSessionConnected"}`); err != nil {
		t.Fatal(err)
	}
	select {
	case <-observed.TakeoverDetected():
	case <-ctx.Done():
		t.Fatal("firmware takeover event was not delivered")
	}
	if !observed.RecognizedTakeover() {
		t.Fatal("firmware takeover event was not latched")
	}
	select {
	case <-connected.Done():
	case <-ctx.Done():
		t.Fatal("recognized takeover did not terminate the displaced session")
	}
}

func TestWebRTCConnectorAuthenticatesSignalsAndPreservesRPC(t *testing.T) {
	fixture := newWebRTCConnectorFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	connected, err := fixture.connector.Connect(ctx, fixture.device)
	if err != nil {
		t.Fatalf("%v (login=%d device=%d authorized=%d)", err, fixture.loginCalls.Load(), fixture.deviceCalls.Load(), fixture.authorizedDeviceCalls.Load())
	}
	var pong string
	if err := connected.Call(ctx, methodPing, nil, &pong); err != nil || pong != "pong" {
		t.Fatalf("ping = %q, error = %v", pong, err)
	}
	if err := connected.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if fixture.loginCalls.Load() != 1 {
		t.Fatalf("login calls = %d", fixture.loginCalls.Load())
	}
	if fixture.pingCalls.Load() != 1 {
		t.Fatalf("RPC ping calls = %d, want 1", fixture.pingCalls.Load())
	}
	offer, _ := fixture.offerSDP.Load().(string)
	if !strings.Contains(offer, "m=video") || !strings.Contains(strings.ToUpper(offer), "H264/90000") {
		t.Fatalf("connector did not negotiate the video-capable profile:\n%s", offer)
	}
}

func TestWebRTCConnectorTeardownDoesNotOutliveOperationDeadline(t *testing.T) {
	fixture := newWebRTCConnectorFixture(t)
	setupCtx, cancelSetup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelSetup()
	connected, err := fixture.connector.connect(setupCtx, fixture.device, SessionProfileData)
	if err != nil {
		t.Fatal(err)
	}
	releasePump := make(chan struct{})
	pumpExited := make(chan struct{})
	connected.pumpMu.Lock()
	connected.pumps.Add(1)
	connected.pumpMu.Unlock()
	go func() {
		defer connected.pumps.Done()
		defer close(pumpExited)
		<-releasePump
	}()
	const teardownTimeout = 50 * time.Millisecond
	teardownCtx, cancelTeardown := context.WithTimeout(context.Background(), teardownTimeout)
	defer cancelTeardown()
	resultCh := make(chan error, 1)
	go func() {
		<-teardownCtx.Done()
		operationErr := teardownCtx.Err()
		_ = connected.Close(teardownCtx)
		resultCh <- operationErr
	}()

	select {
	case err := <-resultCh:
		close(releasePump)
		<-pumpExited
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want operation deadline", err)
		}
		select {
		case <-connected.ctx.Done():
		default:
			t.Fatal("bounded teardown did not cancel the connected session")
		}
	case <-time.After(teardownTimeout + 200*time.Millisecond):
		close(releasePump)
		<-pumpExited
		err := <-resultCh
		t.Fatalf("provider remained blocked after operation deadline: %v", err)
	}
}

func TestFailedWebRTCSetupCleanupHonorsSetupContext(t *testing.T) {
	setupCtx, cancelSetup := context.WithCancel(context.Background())
	cancelSetup()
	connectedCtx, cancelConnected := context.WithCancel(context.Background())
	connected := &connectedSession{ctx: connectedCtx, cancel: cancelConnected}
	connected.pumps.Add(1)
	defer connected.pumps.Done()
	returnErr := errors.New("setup failed")
	done := make(chan struct{})
	go func() {
		closeConnectedOnError(setupCtx, connected, &returnErr)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("failed setup cleanup outlived the setup context")
	}
}

type testSignal struct {
	connection *websocket.Conn
	mu         sync.Mutex
}

func (signal *testSignal) write(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	signal.mu.Lock()
	defer signal.mu.Unlock()
	return signal.connection.Write(ctx, websocket.MessageText, payload)
}
