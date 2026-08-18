package jetkvm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const maxRPCFrame = 16 << 10

type rpcResponse struct {
	ID     uint64
	Result json.RawMessage
	Event  rpcEvent
}

type rpcEvent uint8

const (
	rpcEventNone rpcEvent = iota
	rpcEventOtherSessionConnected
)

type rpcWireError struct {
	Code    *int    `json:"code"`
	Message *string `json:"message"`
}

type rpcProtocolError struct {
	code int
}

func (err *rpcProtocolError) Error() string { return "JetKVM RPC request failed" }

func marshalRPCRequest(id uint64, method string, params any) (string, error) {
	if params == nil {
		params = map[string]any{}
	}
	request := struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
		ID      uint64 `json:"id"`
	}{JSONRPC: "2.0", Method: method, Params: params, ID: id}
	payload, err := json.Marshal(request)
	if err == nil && len(payload) > maxRPCFrame {
		err = errors.New("RPC request too large")
	}
	return string(payload), err
}

func decodeRPCResponse(data []byte) (rpcResponse, error) {
	if len(data) == 0 || len(data) > maxRPCFrame {
		return rpcResponse{}, ErrInvalidResponse
	}
	allowed := map[string]struct{}{"jsonrpc": {}, "method": {}, "params": {}, "result": {}, "error": {}, "id": {}}
	members, malformed, duplicateID := decodeObjectMembers(data, allowed)
	if malformed || duplicateID {
		return rpcResponse{}, ErrInvalidResponse
	}
	var version string
	if err := json.Unmarshal(members["jsonrpc"], &version); err != nil || version != "2.0" {
		return rpcResponse{}, ErrInvalidResponse
	}
	if len(members["id"]) == 0 {
		if len(members["method"]) != 0 {
			var method string
			if json.Unmarshal(members["method"], &method) == nil && method == "otherSessionConnected" &&
				len(members["params"]) == 0 && len(members["result"]) == 0 && len(members["error"]) == 0 {
				return rpcResponse{Event: rpcEventOtherSessionConnected}, nil
			}
			return rpcResponse{}, ErrUnsolicitedRPC
		}
		return rpcResponse{}, ErrInvalidResponse
	}
	var id uint64
	if err := json.Unmarshal(members["id"], &id); err != nil {
		return rpcResponse{}, ErrInvalidResponse
	}
	if len(members["method"]) != 0 || len(members["params"]) != 0 {
		return rpcResponse{}, ErrInvalidResponse
	}
	hasResult, hasError := len(members["result"]) != 0, len(members["error"]) != 0
	if hasResult && hasError {
		return rpcResponse{}, ErrInvalidResponse
	}
	if hasError {
		return rpcResponse{ID: id}, decodeRPCError(members["error"])
	}
	return rpcResponse{ID: id, Result: members["result"]}, nil
}

func decodeRPCError(raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '{' {
		return ErrInvalidResponse
	}
	members, malformed, _ := decodeObjectMembers(trimmed, map[string]struct{}{"code": {}, "message": {}})
	if malformed {
		return ErrInvalidResponse
	}
	var wire rpcWireError
	if err := json.Unmarshal(members["code"], &wire.Code); err != nil || json.Unmarshal(members["message"], &wire.Message) != nil || wire.Code == nil || wire.Message == nil {
		return ErrInvalidResponse
	}
	if *wire.Code == -32601 {
		return ErrRPCMethodUnavailable
	}
	return &rpcProtocolError{code: *wire.Code}
}

func decodeObjectMembers(data []byte, allowed map[string]struct{}) (map[string]json.RawMessage, bool, bool) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return nil, true, false
	}
	members := make(map[string]json.RawMessage)
	malformed, duplicateID := false, false
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return members, true, duplicateID
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return members, true, duplicateID
		}
		if _, exists := members[key]; exists {
			malformed = true
			duplicateID = duplicateID || key == "id"
		} else {
			members[key] = raw
		}
		if _, ok := allowed[key]; !ok || !uniqueJSONValue(raw) {
			malformed = true
		}
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
		malformed = true
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		malformed = true
	}
	return members, malformed, duplicateID
}

func uniqueJSONValue(data []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if !consumeUniqueJSONValue(decoder) {
		return false
	}
	return errors.Is(decoder.Decode(new(any)), io.EOF)
}

func consumeUniqueJSONValue(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			key, ok := keyToken.(string)
			if err != nil || !ok {
				return false
			}
			if _, duplicate := keys[key]; duplicate {
				return false
			}
			keys[key] = struct{}{}
			if !consumeUniqueJSONValue(decoder) {
				return false
			}
		}
		closing, err := decoder.Token()
		return err == nil && closing == json.Delim('}')
	case '[':
		for decoder.More() {
			if !consumeUniqueJSONValue(decoder) {
				return false
			}
		}
		closing, err := decoder.Token()
		return err == nil && closing == json.Delim(']')
	default:
		return false
	}
}
