package jetkvm

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDecodeRPCResponseRejectsSimultaneousResultAndError(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":10,"result":null,"error":{"code":1,"message":"x"}}`)
	if _, err := decodeRPCResponse(input); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("decodeRPCResponse(%s) error = %v", input, err)
	}
}

func FuzzDecodeRPCResponse(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"jsonrpc":"2.0","id":7,"result":{"ready":true}}`),
		[]byte(`{"jsonrpc":"2.0","id":8}`),
		[]byte(`{"jsonrpc":"2.0","id":9,"error":{"code":-32601,"message":"redacted"}}`),
		[]byte(`{"jsonrpc":"2.0","id":10,"result":null,"error":{"code":1,"message":"x"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"event","params":{}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"id":2,"result":null}`),
		[]byte(`{"jsonrpc":"2.0","id":0,"result":{"nested":[{},[],true,null,1]}}`),
		{},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxRPCFrame+1 {
			data = data[:maxRPCFrame+1]
		}
		response, err := decodeRPCResponse(data)
		if len(data) == 0 || len(data) > maxRPCFrame {
			if !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("out-of-bounds input error = %v", err)
			}
			return
		}
		if err != nil {
			return
		}
		var wire struct {
			ID uint64 `json:"id"`
		}
		if err := json.Unmarshal(data, &wire); err != nil || response.ID != wire.ID {
			t.Fatalf("accepted response changed id: got %d want %d", response.ID, wire.ID)
		}
		if len(response.Result) > 0 && (!json.Valid(response.Result) || !uniqueJSONValue(response.Result)) {
			t.Fatalf("accepted non-canonical result for id %d", response.ID)
		}
	})
}
