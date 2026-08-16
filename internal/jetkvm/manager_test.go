package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestManagerStatusReturnsWrappedContextErrorWithoutLaterProbes(t *testing.T) {
	tests := []struct {
		name         string
		extension    string
		failedMethod string
		wantErr      error
		wantMethods  []string
	}{
		{name: "version cancellation", extension: "atx-power", failedMethod: methodLocalVersion, wantErr: context.Canceled, wantMethods: []string{methodPing, methodLocalVersion}},
		{name: "active extension deadline", extension: "atx-power", failedMethod: methodActiveExtension, wantErr: context.DeadlineExceeded, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension}},
		{name: "virtual media cancellation", extension: "atx-power", failedMethod: methodVirtualMediaState, wantErr: context.Canceled, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState}},
		{name: "video deadline", extension: "atx-power", failedMethod: methodVideoState, wantErr: context.DeadlineExceeded, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState}},
		{name: "USB cancellation", extension: "atx-power", failedMethod: methodUSBState, wantErr: context.Canceled, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState}},
		{name: "ATX deadline", extension: "atx-power", failedMethod: methodATXState, wantErr: context.DeadlineExceeded, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}},
		{name: "DC cancellation", extension: "dc-power", failedMethod: methodDCPowerState, wantErr: context.Canceled, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodDCPowerState}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &statusProbeContext{done: make(chan struct{})}
			session := &fakeSession{
				results: statusProbeResults(test.extension),
				callHook: func(_ context.Context, method string, _ any) error {
					if method != test.failedMethod {
						return nil
					}
					ctx.cancel(test.wantErr)
					return fmt.Errorf("wrapped caller failure: %w", test.wantErr)
				},
			}

			status, err := testManager(t, session).Status(ctx, "lab")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(status, mcpserver.Status{}) {
				t.Fatalf("status = %+v, want no partial success", status)
			}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, test.wantMethods) {
				t.Fatalf("methods = %v, want %v", got, test.wantMethods)
			}
		})
	}
}

type statusProbeContext struct {
	done chan struct{}
	err  error
}

func (*statusProbeContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (ctx *statusProbeContext) Done() <-chan struct{}   { return ctx.done }
func (ctx *statusProbeContext) Err() error              { return ctx.err }
func (*statusProbeContext) Value(any) any               { return nil }

func (ctx *statusProbeContext) cancel(err error) {
	ctx.err = err
	close(ctx.done)
}

func TestManagerStatusKeepsOrdinaryProbeFailuresAsWarnings(t *testing.T) {
	tests := []struct {
		name         string
		extension    string
		failedMethod string
		warning      mcpserver.StatusWarning
		wantMethods  []string
	}{
		{name: "version", extension: "atx-power", failedMethod: methodLocalVersion, warning: mcpserver.StatusWarningVersionUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}},
		{name: "active extension", extension: "atx-power", failedMethod: methodActiveExtension, warning: mcpserver.StatusWarningActiveExtensionUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState}},
		{name: "virtual media", extension: "atx-power", failedMethod: methodVirtualMediaState, warning: mcpserver.StatusWarningVirtualMediaUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}},
		{name: "video", extension: "atx-power", failedMethod: methodVideoState, warning: mcpserver.StatusWarningVideoUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}},
		{name: "USB", extension: "atx-power", failedMethod: methodUSBState, warning: mcpserver.StatusWarningUSBUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}},
		{name: "ATX", extension: "atx-power", failedMethod: methodATXState, warning: mcpserver.StatusWarningATXUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}},
		{name: "DC", extension: "dc-power", failedMethod: methodDCPowerState, warning: mcpserver.StatusWarningDCUnavailable, wantMethods: []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodDCPowerState}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{
				results: statusProbeResults(test.extension),
				err:     map[string]error{test.failedMethod: errors.New("ordinary probe failure")},
			}

			status, err := testManager(t, session).Status(context.Background(), "lab")
			if err != nil {
				t.Fatal(err)
			}
			if !status.Connected || !reflect.DeepEqual(status.Warnings, []mcpserver.StatusWarning{test.warning}) {
				t.Fatalf("status = %+v, want connected status with warning %q", status, test.warning)
			}
			if got := calledMethods(session.calls); !reflect.DeepEqual(got, test.wantMethods) {
				t.Fatalf("methods = %v, want %v", got, test.wantMethods)
			}
		})
	}
}

func TestManagerStatusKeepsInternalRequestTimeoutAsWarning(t *testing.T) {
	session := &fakeSession{
		results: statusProbeResults("atx-power"),
		err: map[string]error{
			methodVideoState: classifyOperationError(context.DeadlineExceeded, ToolOutcomeUnknown),
		},
	}

	status, err := testManager(t, session).Status(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Connected || !reflect.DeepEqual(status.Warnings, []mcpserver.StatusWarning{mcpserver.StatusWarningVideoUnavailable}) {
		t.Fatalf("status = %+v, want connected partial status with video warning", status)
	}
	wantMethods := []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState, methodUSBState, methodATXState}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want continued probes %v", got, wantMethods)
	}
}

func TestManagerStatusPreservesOriginalCallerCancellationError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var probeErr error
	session := &fakeSession{
		results: statusProbeResults("atx-power"),
		callHook: func(callCtx context.Context, method string, _ any) error {
			if method != methodVideoState {
				return nil
			}
			cancel()
			probeErr = classifyOperationError(fmt.Errorf("video probe interrupted: %w", callCtx.Err()), ToolOutcomeUnknown)
			return probeErr
		},
	}

	status, err := testManager(t, session).Status(ctx, "lab")
	if err != probeErr {
		t.Fatalf("error = %#v, want original probe error %#v", err, probeErr)
	}
	if !reflect.DeepEqual(status, mcpserver.Status{}) {
		t.Fatalf("status = %+v, want no partial success", status)
	}
	var classified interface{ ToolErrorOutcome() string }
	if !errors.As(err, &classified) || classified.ToolErrorOutcome() != ToolOutcomeUnknown {
		t.Fatalf("error = %#v, want preserved unknown outcome", err)
	}
}

func TestManagerStatusStopsAfterSuccessfulProbeWhenCallerCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := &fakeSession{
		results: statusProbeResults("atx-power"),
		callHook: func(_ context.Context, method string, _ any) error {
			if method == methodVideoState {
				cancel()
			}
			return nil
		},
	}

	status, err := testManager(t, session).Status(ctx, "lab")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want caller cancellation", err)
	}
	if !reflect.DeepEqual(status, mcpserver.Status{}) {
		t.Fatalf("status = %+v, want no partial success", status)
	}
	wantMethods := []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want no probes after cancellation %v", got, wantMethods)
	}
}

func TestManagerStatusCallerCancellationWinsOverUnrelatedProbeError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := &fakeSession{
		results: statusProbeResults("atx-power"),
		callHook: func(_ context.Context, method string, _ any) error {
			if method != methodVideoState {
				return nil
			}
			cancel()
			return errors.New("unrelated probe failure")
		},
	}

	status, err := testManager(t, session).Status(ctx, "lab")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want caller cancellation", err)
	}
	if !reflect.DeepEqual(status, mcpserver.Status{}) {
		t.Fatalf("status = %+v, want no partial success", status)
	}
	wantMethods := []string{methodPing, methodLocalVersion, methodActiveExtension, methodVirtualMediaState, methodVideoState}
	if got := calledMethods(session.calls); !reflect.DeepEqual(got, wantMethods) {
		t.Fatalf("methods = %v, want no probes after cancellation %v", got, wantMethods)
	}
}

func TestManagerStatusCancellationDuringProviderSetupStaysNotSent(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(
		[]DeviceConfig{{Name: "lab", BaseURL: *base}},
		NewWebRTCProvider(WebRTCProviderOptions{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := manager.Status(ctx, "lab")
		done <- err
	}()

	<-started
	cancel()
	err = <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %#v, want caller cancellation", err)
	}
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "canceled" || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
		t.Fatalf("error = %#v, want canceled/not_sent", err)
	}
}

func statusProbeResults(extension string) map[string]any {
	return map[string]any{
		methodPing:              "pong",
		methodLocalVersion:      map[string]any{"appVersion": "0.6.0", "systemVersion": "1.2.3"},
		methodActiveExtension:   extension,
		methodVirtualMediaState: nil,
		methodVideoState:        map[string]any{"ready": true, "width": 1920, "height": 1080, "fps": 30},
		methodUSBState:          "not attached",
		methodATXState:          map[string]any{"power": true},
		methodDCPowerState:      map[string]any{"isOn": true, "voltage": 12.0},
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
