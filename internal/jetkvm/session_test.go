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
			session := newRPCSession(context.Background(), sender, time.Second)
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

func TestRPCSessionReturnsStableErrorsAndTimeout(t *testing.T) {
	t.Run("protocol error", func(t *testing.T) {
		sender := &fakeTextSender{sent: make(chan string, 1)}
		session := newRPCSession(context.Background(), sender, time.Second)
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
		session := newRPCSession(context.Background(), sender, 5*time.Millisecond)
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
		session := newRPCSession(context.Background(), sender, time.Second)
		done := make(chan error, 1)
		go func() { done <- session.Call(context.Background(), "ping", nil, new(string)) }()
		<-sender.sent
		session.Close()
		if err := <-done; !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("error = %v", err)
		}
	})
}

func jsonNumber(value uint64) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
