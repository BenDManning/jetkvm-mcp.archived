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

func TestWebRTCProviderClosesIdleHTTPConnectionsAfterAuthenticationFailure(t *testing.T) {
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
	provider := NewWebRTCProvider(WebRTCProviderOptions{})
	_, err = provider.connect(context.Background(), DeviceConfig{BaseURL: *base}, SessionProfileData)
	if !errors.Is(err, ErrAuthentication) {
		t.Fatalf("error = %v, want authentication failure", err)
	}
	select {
	case <-closed:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("authentication failure left the device HTTP connection open")
	}
}

func TestWebRTCProviderAuthenticatesSignalsAndCallsRPC(t *testing.T) {
	var loginCalls atomic.Int32
	var deviceCalls atomic.Int32
	var authorizedDeviceCalls atomic.Int32
	remotePeer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer remotePeer.Close()

	remotePeer.OnDataChannel(func(channel *webrtc.DataChannel) {
		if channel.Label() != "rpc" {
			return
		}
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
				_ = channel.SendText("{\"jsonrpc\":\"2.0\",\"id\":" + jsonNumber(request.ID) + ",\"result\":\"pong\"}")
			}
		})
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/device", func(writer http.ResponseWriter, request *http.Request) {
		deviceCalls.Add(1)
		cookie, err := request.Cookie("session")
		if err != nil || cookie.Value != "ok" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		authorizedDeviceCalls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte("{\"authMode\":\"password\",\"deviceId\":\"test-device\",\"appVersion\":\"test\",\"systemVersion\":\"test\"}"))
	})
	mux.HandleFunc("/auth/login-local", func(writer http.ResponseWriter, request *http.Request) {
		loginCalls.Add(1)
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
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	provider := NewWebRTCProvider(WebRTCProviderOptions{ConnectTimeout: 10 * time.Second, RequestTimeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err = provider.WithSession(ctx, DeviceConfig{Name: "lab", BaseURL: *base, Password: "correct"}, SessionProfileData, func(session Session) error {
		var pong string
		if err := session.Call(ctx, "ping", nil, &pong); err != nil {
			return err
		}
		if pong != "pong" {
			t.Fatalf("pong = %q", pong)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("%v (login=%d device=%d authorized=%d)", err, loginCalls.Load(), deviceCalls.Load(), authorizedDeviceCalls.Load())
	}
	if loginCalls.Load() != 1 {
		t.Fatalf("login calls = %d", loginCalls.Load())
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
