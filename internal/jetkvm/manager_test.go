package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type recordedCall struct {
	method string
	params any
}

type fakeSession struct {
	calls      []recordedCall
	results    map[string]any
	err        map[string]error
	callHook   func(context.Context, string, any) error
	uploads    []recordedUpload
	uploadHook func()
	uploadErr  error
	uploadFunc func(context.Context, string, io.Reader, int64) error
	h264       []byte
	capturedAt time.Time
}

type recordedUpload struct {
	id   string
	data []byte
	size int64
}

func (session *fakeSession) Call(ctx context.Context, method string, params any, result any) error {
	session.calls = append(session.calls, recordedCall{method: method, params: params})
	if session.callHook != nil {
		if err := session.callHook(ctx, method, params); err != nil {
			return err
		}
	}
	if err := session.err[method]; err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	encoded, err := json.Marshal(session.results[method])
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}

func (session *fakeSession) Upload(ctx context.Context, id string, reader io.Reader, size int64) error {
	if session.uploadFunc != nil {
		return session.uploadFunc(ctx, id, reader, size)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	session.uploads = append(session.uploads, recordedUpload{id: id, data: data, size: size})
	if session.uploadHook != nil {
		session.uploadHook()
	}
	return session.uploadErr
}

func (session *fakeSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return append([]byte(nil), session.h264...), session.capturedAt, nil
}

type fakeProvider struct {
	session *fakeSession
	device  DeviceConfig
	profile SessionProfile
}

type failingBeforeSessionProvider struct {
	err error
}

func (provider *failingBeforeSessionProvider) WithSession(context.Context, DeviceConfig, SessionProfile, func(Session) error) error {
	return provider.err
}

func (provider *fakeProvider) WithSession(ctx context.Context, device DeviceConfig, profile SessionProfile, operation func(Session) error) error {
	provider.device = device
	provider.profile = profile
	return operation(provider.session)
}

func TestManagerStatusProjectsOrdinaryDeviceState(t *testing.T) {
	session := &fakeSession{results: map[string]any{
		"ping":                 "pong",
		"getLocalVersion":      map[string]any{"appVersion": "0.6.0", "systemVersion": "1.2.3"},
		"getActiveExtension":   "atx-power",
		"getVideoState":        map[string]any{"ready": true, "streaming": 1, "width": 1920, "height": 1080, "fps": 30},
		"getUSBState":          "configured",
		"getVirtualMediaState": nil,
		"getATXState":          map[string]any{"power": true, "hdd": false},
	}}
	manager := testManager(t, session)

	status, err := manager.Status(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	if status.Device != "lab" || !status.Connected || status.Application != "0.6.0" || status.System != "1.2.3" || status.Extension != "atx-power" || status.VideoReady == nil || !*status.VideoReady || status.USBState != "configured" {
		t.Fatalf("status = %+v", status)
	}
	if status.VideoWidth != 1920 || status.VideoHeight != 1080 || status.VideoFPS != 30 || status.ATXPowerOn == nil || !*status.ATXPowerOn {
		t.Fatalf("detailed status = %+v", status)
	}
	wantMethods := []string{"ping", "getLocalVersion", "getActiveExtension", "getVirtualMediaState", "getVideoState", "getUSBState", "getATXState"}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want %v", got, wantMethods)
	}
}

func TestManagerPowerMapsSevenActionsExactly(t *testing.T) {
	for _, test := range []struct {
		action     mcpserver.PowerAction
		target     string
		wantMethod string
		wantParams any
	}{
		{mcpserver.PowerActionPressHostPowerButton, "", "setATXPowerAction", map[string]any{"action": "power-short"}},
		{mcpserver.PowerActionForceHostPowerOff, "", "setATXPowerAction", map[string]any{"action": "power-long"}},
		{mcpserver.PowerActionPressHostResetButton, "", "setATXPowerAction", map[string]any{"action": "reset"}},
		{mcpserver.PowerActionTurnHostDCPowerOn, "", "setDCPowerState", map[string]any{"enabled": true}},
		{mcpserver.PowerActionTurnHostDCPowerOff, "", "setDCPowerState", map[string]any{"enabled": false}},
		{mcpserver.PowerActionWakeHostUSB, "", "wakeHost", nil},
		{mcpserver.PowerActionWakeHostLAN, "server", "sendWOLMagicPacket", map[string]any{"macAddress": "02:00:00:00:00:01", "broadcastIP": "192.0.2.255"}},
	} {
		t.Run(string(test.action), func(t *testing.T) {
			session := &fakeSession{results: map[string]any{}}
			manager := testManager(t, session)
			result, err := manager.Power(context.Background(), "lab", test.action, test.target)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != "completed" || result.Action != test.action || len(session.calls) != 1 {
				t.Fatalf("result=%+v calls=%+v", result, session.calls)
			}
			call := session.calls[0]
			if call.method != test.wantMethod || !reflect.DeepEqual(call.params, test.wantParams) {
				t.Fatalf("call = %#v, want %s %#v", call, test.wantMethod, test.wantParams)
			}
		})
	}
}

func TestManagerPowerClassifiesConnectionFailureAsDefinitelyNotSent(t *testing.T) {
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &failingBeforeSessionProvider{err: ErrDeviceUnreachable})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "device_unavailable" || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
		t.Fatalf("error = %#v, want device_unavailable/not_sent", err)
	}
}

func TestManagerPowerPreservesPossiblySentFailure(t *testing.T) {
	possiblySent := classifyOperationError(context.DeadlineExceeded, ToolOutcomeUnknown)
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &failingBeforeSessionProvider{err: possiblySent})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
	var classified interface{ ToolErrorOutcome() string }
	if !errors.As(err, &classified) || classified.ToolErrorOutcome() != ToolOutcomeUnknown {
		t.Fatalf("error = %#v, want preserved unknown outcome", err)
	}
}

func TestManagerRejectsUnknownDeviceAndWakeTarget(t *testing.T) {
	manager := testManager(t, &fakeSession{results: map[string]any{}})
	if _, err := manager.Status(context.Background(), "missing"); !errors.Is(err, ErrUnknownDevice) {
		t.Fatalf("unknown device error = %v", err)
	}
	if _, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionWakeHostLAN, "missing"); !errors.Is(err, ErrUnknownWakeTarget) {
		t.Fatalf("unknown target error = %v", err)
	}
}

func TestNewManagerRejectsIPv6WakeOnLANBroadcast(t *testing.T) {
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewManager([]DeviceConfig{{
		Name: "lab", BaseURL: *base,
		WakeOnLAN: map[string]WakeOnLANTarget{
			"server": {MACAddress: "02:00:00:00:00:01", BroadcastIP: "2001:db8::ffff"},
		},
	}}, &fakeProvider{session: &fakeSession{}})
	if err == nil || !strings.Contains(err.Error(), "IPv4") {
		t.Fatalf("error = %v, want IPv4 broadcast rejection", err)
	}
}

func TestNewManagerRejectsDiscardedDeviceURLComponents(t *testing.T) {
	for _, rawURL := range []string{"https://jetkvm.invalid/base?token=private", "https://jetkvm.invalid/base#private"} {
		base, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		_, err = NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &fakeProvider{session: &fakeSession{}})
		if err == nil || !strings.Contains(err.Error(), "query or fragment") {
			t.Fatalf("url=%q error=%v", rawURL, err)
		}
	}
}

func testManager(t *testing.T, session *fakeSession) *Manager {
	t.Helper()
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{
		Name: "lab", BaseURL: *base,
		MediaURLAllowedOrigins: []string{
			"https://example.invalid",
			"https://media.invalid",
		},
		WakeOnLAN: map[string]WakeOnLANTarget{
			"server": {MACAddress: "02:00:00:00:00:01", BroadcastIP: "192.0.2.255"},
		},
	}}, &fakeProvider{session: session})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func calledMethods(calls []recordedCall) []string {
	methods := make([]string, len(calls))
	for index, call := range calls {
		methods[index] = call.method
	}
	return methods
}

var _ mcpserver.Device = (*Manager)(nil)
