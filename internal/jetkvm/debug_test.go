package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestManagerDebugRPCUsesDataSessionAndReturnsRawResult(t *testing.T) {
	session := &fakeSession{results: map[string]any{"customMethod": map[string]any{"value": 42}}}
	provider := &fakeProvider{session: session}
	manager := testManager(t, session)
	manager.provider = provider
	result, err := manager.DebugRPC(context.Background(), "lab", "customMethod", json.RawMessage(`{"input":true}`))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil || decoded["value"] != float64(42) {
		t.Fatalf("result = %s err=%v", result, err)
	}
	if provider.profile != SessionProfileData || len(session.calls) != 1 || session.calls[0].method != "customMethod" {
		t.Fatalf("profile=%v calls=%#v", provider.profile, session.calls)
	}
}

func TestManagerDebugRPCRejectsInvalidMethodAndParamsBeforeSession(t *testing.T) {
	session := &fakeSession{results: map[string]any{}}
	manager := testManager(t, session)
	for _, test := range []struct {
		method string
		params json.RawMessage
	}{
		{method: "bad method", params: json.RawMessage(`{}`)},
		{method: "customMethod", params: json.RawMessage(`[]`)},
		{method: "customMethod", params: json.RawMessage(`{"duplicate":1,"duplicate":2}`)},
	} {
		if _, err := manager.DebugRPC(context.Background(), "lab", test.method, test.params); !errors.Is(err, ErrUnsupportedInput) {
			t.Fatalf("method=%q params=%s error=%v", test.method, test.params, err)
		}
	}
	if len(session.calls) != 0 {
		t.Fatalf("calls = %#v", session.calls)
	}
}
