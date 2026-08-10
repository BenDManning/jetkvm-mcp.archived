package jetkvm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var debugRPCMethodPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,127}$`)

func (manager *Manager) DebugRPC(ctx context.Context, name, method string, params json.RawMessage) (json.RawMessage, error) {
	device, err := manager.device(name)
	if err != nil {
		return nil, err
	}
	method = strings.TrimSpace(method)
	if !debugRPCMethodPattern.MatchString(method) {
		return nil, fmt.Errorf("%w: RPC method", ErrUnsupportedInput)
	}
	trimmed := bytes.TrimSpace(params)
	if len(trimmed) == 0 {
		trimmed = []byte("{}")
	}
	if len(trimmed) > maxRPCFrame || trimmed[0] != '{' || !uniqueJSONValue(trimmed) {
		return nil, fmt.Errorf("%w: RPC params must be one JSON object", ErrUnsupportedInput)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, fmt.Errorf("%w: RPC params", ErrUnsupportedInput)
	}
	var result json.RawMessage
	err = manager.provider.WithSession(ctx, device, SessionProfileData, func(session Session) error {
		return session.Call(ctx, method, object, &result)
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return json.RawMessage("null"), nil
	}
	return append(json.RawMessage(nil), result...), nil
}
