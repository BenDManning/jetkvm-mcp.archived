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

var debugRPCReadOnlyMethods = map[string]struct{}{
	methodPing:            {},
	methodLocalVersion:    {},
	methodActiveExtension: {},
}

func (manager *Manager) DebugRPC(ctx context.Context, name, method string, params json.RawMessage, unsafeAcknowledged bool) (json.RawMessage, error) {
	method = strings.TrimSpace(method)
	if !debugRPCMethodPattern.MatchString(method) {
		return nil, fmt.Errorf("%w: RPC method", ErrUnsupportedInput)
	}
	if _, reviewed := debugRPCReadOnlyMethods[method]; !reviewed && !unsafeAcknowledged {
		return nil, fmt.Errorf("%w: unreviewed RPC method requires unsafe acknowledgement", ErrUnsupportedInput)
	}
	device, err := manager.device(name)
	if err != nil {
		return nil, err
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
	_, reviewed := debugRPCReadOnlyMethods[method]
	err = manager.withOperation(ctx, device, !reviewed, false, func(operationCtx context.Context, session Session) error {
		return session.Call(operationCtx, method, object, &result)
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return json.RawMessage("null"), nil
	}
	return append(json.RawMessage(nil), result...), nil
}
