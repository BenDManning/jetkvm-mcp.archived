package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeTextSender struct {
	sent chan string
	err  error
}

func (sender *fakeTextSender) SendText(payload string) error {
	if sender.err != nil {
		return sender.err
	}
	sender.sent <- payload
	return nil
}

func TestRPCSessionCorrelatesResultsAndMutationOmission(t *testing.T) {
	for _, test := range []struct {
		name       string
		response   func(uint64) string
		wantResult bool
	}{
		{name: "result", response: func(id uint64) string { return `{"jsonrpc":"2.0","id":` + jsonNumber(id) + `,"result":{"ready":true}}` }, wantResult: true},
		{name: "mutation omission", response: func(id uint64) string { return `{"jsonrpc":"2.0","id":` + jsonNumber(id) + `}` }},
	} {
		t.Run(test.name, func(t *testing.T) {
			sender := &fakeTextSender{sent: make(chan string, 1)}
			session := newRPCSession(context.Background(), sender, time.Second, nil)
			defer session.Close()

			var result struct {
				Ready bool `json:"ready"`
			}
			resultTarget := any(nil)
			if test.wantResult {
				resultTarget = &result
			}
			done := make(chan error, 1)
			go func() { done <- session.Call(context.Background(), "getVideoState", nil, resultTarget) }()

			request := <-sender.sent
			var wire struct {
				ID uint64 `json:"id"`
			}
			if err := json.Unmarshal([]byte(request), &wire); err != nil {
				t.Fatal(err)
			}
			session.HandleMessage([]byte(test.response(wire.ID)))
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if test.wantResult && !result.Ready {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestRPCSessionRecognizedTakeoverLatchesBeforeNotifying(t *testing.T) {
	sender := &fakeTextSender{sent: make(chan string, 1)}
	notified := make(chan struct{})
	session := newRPCSession(context.Background(), sender, time.Second, func() { close(notified) })
	defer session.Close()

	session.HandleMessage([]byte(`{"jsonrpc":"2.0","method":"otherSessionConnected"}`))
	select {
	case <-notified:
	default:
		t.Fatal("recognized takeover was not delivered")
	}
}

func TestRPCSessionReturnsStableErrorsAndTimeout(t *testing.T) {
	t.Run("protocol error", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, time.Second, nil)
		defer session.Close()
		done := make(chan error, 1)
		go func() { done <- session.Call(context.Background(), "missing", nil, nil) }()
		request := <-sender.sent
		var wire struct {
			ID uint64 `json:"id"`
		}
		_ = json.Unmarshal([]byte(request), &wire)
		session.HandleMessage([]byte(`{"jsonrpc":"2.0","id":` + jsonNumber(wire.ID) + `,"error":{"code":-32601,"message":"secret firmware detail"}}`))
		if err := <-done; !errors.Is(err, ErrRPCMethodUnavailable) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("timeout removes pending request", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, 5*time.Millisecond, nil)
		defer session.Close()
		if err := session.Call(context.Background(), "ping", nil, new(string)); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v", err)
		}
		if count := session.pendingCount(); count != 0 {
			t.Fatalf("pending = %d", count)
		}
	})

	t.Run("close releases caller", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, time.Second, nil)
		done := make(chan error, 1)
		go func() { done <- session.Call(context.Background(), "ping", nil, new(string)) }()
		<-sender.sent
		session.Close()
		if err := <-done; !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestRPCSessionCancellationBeforeSendIsDefinitelyNotSent(t *testing.T) {
	sender := &fakeTextSender{sent: make(chan string, 1)}
	session := newRPCSession(context.Background(), sender, time.Second, nil)
	defer session.Close()
	<-session.sendGate

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- session.Call(ctx, "setATXPowerAction", nil, nil) }()
	deadline := time.Now().Add(time.Second)
	for session.pendingCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	err := <-done
	var classified interface {
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorOutcome() != "not_sent" {
		t.Fatalf("error = %#v, want classified not_sent outcome", err)
	}
	select {
	case payload := <-sender.sent:
		t.Fatalf("request was sent before cancellation: %q", payload)
	default:
	}
}

func TestRPCSessionDispatchPhasesClassifyMutationOutcomes(t *testing.T) {
	t.Run("during send is unknown", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1), err: errors.New("transport write failed")}
		session := newRPCSession(context.Background(), sender, time.Second, nil)
		defer session.Close()
		err := session.Call(context.Background(), "setATXPowerAction", nil, nil)
		assertToolOutcome(t, err, ToolOutcomeUnknown)
	})

	t.Run("after send before response is unknown", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, 5*time.Millisecond, nil)
		defer session.Close()
		err := session.Call(context.Background(), "setATXPowerAction", nil, nil)
		assertToolOutcome(t, err, ToolOutcomeUnknown)
	})

	t.Run("cancellation after send is unknown", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, time.Second, nil)
		defer session.Close()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- session.Call(ctx, "setATXPowerAction", nil, nil) }()
		<-sender.sent
		cancel()
		assertToolOutcome(t, <-done, ToolOutcomeUnknown)
	})

	t.Run("confirmed RPC failure is failed", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, time.Second, nil)
		defer session.Close()
		done := make(chan error, 1)
		go func() { done <- session.Call(context.Background(), "setATXPowerAction", nil, nil) }()
		var request struct {
			ID uint64 `json:"id"`
		}
		if err := json.Unmarshal([]byte(<-sender.sent), &request); err != nil {
			t.Fatal(err)
		}
		session.HandleMessage([]byte(`{"jsonrpc":"2.0","id":` + jsonNumber(request.ID) + `,"error":{"code":-32601,"message":"private"}}`))
		assertToolOutcome(t, <-done, ToolOutcomeFailed)
	})
}

func assertToolOutcome(t *testing.T, err error, want string) {
	t.Helper()
	var classified interface{ ToolErrorOutcome() string }
	if !errors.As(err, &classified) || classified.ToolErrorOutcome() != want {
		t.Fatalf("error = %#v, want outcome %q", err, want)
	}
}

func TestRPCSessionAdmissionLimitIsBusyAndNotSent(t *testing.T) {
	sender := &fakeTextSender{sent: make(chan string, 1)}
	session := newRPCSession(context.Background(), sender, time.Second, nil)
	defer session.Close()
	for id := uint64(1); id <= maxPendingRPCRequests; id++ {
		if err := session.addPending(id, make(chan rpcOutcome, 1)); err != nil {
			t.Fatal(err)
		}
	}
	err := session.Call(context.Background(), "setATXPowerAction", nil, nil)
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "busy" || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
		t.Fatalf("error = %#v, want busy/not_sent", err)
	}
	select {
	case payload := <-sender.sent:
		t.Fatalf("admission-limited call sent payload %q", payload)
	default:
	}
}

func jsonNumber(value uint64) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
