package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type blockingProvider struct {
	mu      sync.Mutex
	calls   int
	entered chan struct{}
	release chan struct{}
	session Session
}

func (provider *blockingProvider) Connect(ctx context.Context, _ DeviceConfig) (ConnectedSession, error) {
	provider.mu.Lock()
	provider.calls++
	provider.mu.Unlock()
	provider.entered <- struct{}{}
	select {
	case <-provider.release:
		return testConnected(provider.session), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (provider *blockingProvider) count() int {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.calls
}

type admissionSession struct{}

func (admissionSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodVideoState && result != nil {
		ready := true
		if target, ok := result.(*struct {
			Ready *bool `json:"ready"`
		}); ok {
			target.Ready = &ready
		}
	}
	return nil
}
func (admissionSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (admissionSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return []byte{0, 0, 0, 1, 0x65}, time.Now(), nil
}

func admissionManager(t *testing.T, limits Limits, provider SessionConnector, devices ...string) *Manager {
	t.Helper()
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	configured := make([]DeviceConfig, 0, len(devices))
	for _, name := range devices {
		configured = append(configured, DeviceConfig{Name: name, BaseURL: *base})
	}
	manager, err := NewManager(configured, provider, WithLimits(limits), WithDecoder(&admissionDecoder{}))
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func assertBusyNotSent(t *testing.T, err error) {
	t.Helper()
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "busy" || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
		t.Fatalf("error = %#v, want busy/not_sent", err)
	}
}

func TestManagerAdmissionRejectsGlobalAndPerDeviceCapacityBeforeDispatch(t *testing.T) {
	for _, test := range []struct {
		name      string
		limits    Limits
		second    string
		wantCalls int
	}{
		{"global", Limits{MaxOperations: 1, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 1, MaxCaptures: 1, MaxDecoders: 1}, "rack", 1},
		{"per-device", Limits{MaxOperations: 2, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 2, MaxCaptures: 1, MaxDecoders: 1}, "lab", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &blockingProvider{entered: make(chan struct{}, 2), release: make(chan struct{}), session: admissionSession{}}
			manager := admissionManager(t, test.limits, provider, "lab", "rack")
			done := make(chan error, 1)
			go func() {
				_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
				done <- err
			}()
			<-provider.entered
			_, err := manager.Power(context.Background(), test.second, mcpserver.PowerActionPressHostPowerButton, "")
			assertBusyNotSent(t, err)
			if provider.count() != test.wantCalls {
				t.Fatalf("provider calls = %d, want %d", provider.count(), test.wantCalls)
			}
			close(provider.release)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManagerAdmissionAllowsUnrelatedDevicesAndBoundsConnectionAttempts(t *testing.T) {
	provider := &blockingProvider{entered: make(chan struct{}, 3), release: make(chan struct{}), session: admissionSession{}}
	manager := admissionManager(t, Limits{MaxOperations: 2, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 1, MaxCaptures: 1, MaxDecoders: 1}, provider, "lab", "rack")
	done := make(chan error, 1)
	go func() {
		_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
		done <- err
	}()
	<-provider.entered
	_, err := manager.Power(context.Background(), "rack", mcpserver.PowerActionPressHostPowerButton, "")
	assertBusyNotSent(t, err)
	if provider.count() != 1 {
		t.Fatalf("session capacity dispatched %d calls", provider.count())
	}
	close(provider.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	provider = &blockingProvider{entered: make(chan struct{}, 3), release: make(chan struct{}), session: admissionSession{}}
	manager = admissionManager(t, Limits{MaxOperations: 2, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 2, MaxCaptures: 1, MaxDecoders: 1}, provider, "lab", "rack")
	done = make(chan error, 2)
	go func() {
		_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
		done <- err
	}()
	<-provider.entered
	go func() {
		_, err := manager.Power(context.Background(), "rack", mcpserver.PowerActionPressHostPowerButton, "")
		done <- err
	}()
	<-provider.entered
	if provider.count() != 2 {
		t.Fatalf("unrelated device calls = %d, want 2", provider.count())
	}
	close(provider.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

type admissionDecoder struct{}

func (*admissionDecoder) Decode(context.Context, []byte, int, int) ([]byte, int, int, error) {
	return nil, 0, 0, ErrDecoderUnavailable
}

func TestManagerAdmissionCaptureAndDecoderCapacityRejectBeforeSecondDispatch(t *testing.T) {
	limits := Limits{MaxOperations: 2, MaxOperationsPerDevice: 2, MaxConnectionAttempts: 2, MaxCaptures: 1, MaxDecoders: 2}
	provider := &blockingProvider{entered: make(chan struct{}, 2), release: make(chan struct{}), session: admissionSession{}}
	manager := admissionManager(t, limits, provider, "lab")
	done := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		done <- err
	}()
	<-provider.entered
	_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
	assertBusyNotSent(t, err)
	if provider.count() != 1 {
		t.Fatalf("capture capacity dispatched %d calls", provider.count())
	}
	close(provider.release)
	<-done
}

type blockingDecoder struct {
	entered chan struct{}
	release chan struct{}
}

func (decoder *blockingDecoder) Decode(context.Context, []byte, int, int) ([]byte, int, int, error) {
	decoder.entered <- struct{}{}
	<-decoder.release
	return nil, 0, 0, ErrDecoderUnavailable
}

func TestManagerAdmissionDecoderCapacityRejectsBeforeProviderDispatch(t *testing.T) {
	provider := &blockingProvider{entered: make(chan struct{}, 2), release: make(chan struct{}), session: admissionSession{}}
	close(provider.release)
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	decoder := &blockingDecoder{entered: make(chan struct{}, 1), release: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, provider, WithLimits(Limits{MaxOperations: 2, MaxOperationsPerDevice: 2, MaxConnectionAttempts: 2, MaxCaptures: 2, MaxDecoders: 1}), WithDecoder(decoder))
	if err != nil {
		t.Fatal(err)
	}
	first := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		first <- err
	}()
	<-decoder.entered
	_, err = manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
	assertBusyNotSent(t, err)
	if provider.count() != 1 {
		t.Fatalf("decoder capacity dispatched %d calls", provider.count())
	}
	close(decoder.release)
	<-first
}

func TestManagerMutationWaitIsCancelableAndPermitReleases(t *testing.T) {
	provider := &blockingProvider{entered: make(chan struct{}, 3), release: make(chan struct{}), session: admissionSession{}}
	manager := admissionManager(t, Limits{MaxOperations: 3, MaxOperationsPerDevice: 3, MaxConnectionAttempts: 3, MaxCaptures: 1, MaxDecoders: 1}, provider, "lab")
	first := make(chan error, 1)
	go func() {
		_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
		first <- err
	}()
	<-provider.entered
	ctx, cancel := context.WithCancel(context.Background())
	second := make(chan error, 1)
	go func() {
		_, err := manager.Power(ctx, "lab", mcpserver.PowerActionPressHostPowerButton, "")
		second <- err
	}()
	cancel()
	err := <-second
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter error = %v", err)
	}
	if provider.count() != 1 {
		t.Fatalf("canceled waiter dispatched %d calls", provider.count())
	}
	close(provider.release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
}

type panicOnceProvider struct {
	session  Session
	panicked bool
	mu       sync.Mutex
}

func (provider *panicOnceProvider) Connect(_ context.Context, _ DeviceConfig) (ConnectedSession, error) {
	provider.mu.Lock()
	panicNow := !provider.panicked
	provider.panicked = true
	provider.mu.Unlock()
	if panicNow {
		panic("test provider panic")
	}
	return testConnected(provider.session), nil
}

func TestManagerAdmissionReleasesPermitsAfterProviderPanic(t *testing.T) {
	provider := &panicOnceProvider{session: admissionSession{}}
	manager := admissionManager(t, Limits{MaxOperations: 1, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 1, MaxCaptures: 1, MaxDecoders: 1}, provider, "lab")
	panicked := make(chan struct{})
	go func() {
		defer close(panicked)
		defer func() { _ = recover() }()
		_, _ = manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
	}()
	<-panicked
	if _, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, ""); err != nil {
		t.Fatalf("operation after panic = %v", err)
	}
}

type errorOnceProvider struct {
	session Session
	failed  bool
	mu      sync.Mutex
}

func (provider *errorOnceProvider) Connect(_ context.Context, _ DeviceConfig) (ConnectedSession, error) {
	provider.mu.Lock()
	fail := !provider.failed
	provider.failed = true
	provider.mu.Unlock()
	if fail {
		return nil, ErrDeviceUnreachable
	}
	return testConnected(provider.session), nil
}

func TestManagerAdmissionReleasesPermitsAfterProviderError(t *testing.T) {
	provider := &errorOnceProvider{session: admissionSession{}}
	manager := admissionManager(t, Limits{MaxOperations: 1, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 1, MaxCaptures: 1, MaxDecoders: 1}, provider, "lab")
	_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
	if !errors.Is(err, ErrDeviceUnreachable) {
		t.Fatalf("first error = %v", err)
	}
	if _, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, ""); err != nil {
		t.Fatalf("operation after error = %v", err)
	}
}

func TestManagerSerializesCompleteHIDAndVirtualMediaMutations(t *testing.T) {
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	session := &sequenceSession{entered: make(chan struct{}), release: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &sequenceProvider{session: session}, WithLimits(Limits{MaxOperations: 3, MaxOperationsPerDevice: 3, MaxConnectionAttempts: 3, MaxCaptures: 1, MaxDecoders: 1}))
	if err != nil {
		t.Fatal(err)
	}
	keyboardDone := make(chan error, 1)
	go func() {
		_, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardTypeText, Text: "ab"})
		keyboardDone <- err
	}()
	<-session.entered
	mouseDone := make(chan error, 1)
	go func() {
		_, err := manager.Mouse(context.Background(), "lab", mcpserver.MouseRequest{Operation: mcpserver.MouseClick, Button: "left"})
		mouseDone <- err
	}()
	mediaDone := make(chan error, 1)
	go func() {
		_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUnmount})
		mediaDone <- err
	}()
	close(session.release)
	if err := <-keyboardDone; err != nil {
		t.Fatal(err)
	}
	if err := <-mouseDone; err != nil {
		t.Fatal(err)
	}
	if err := <-mediaDone; err != nil {
		t.Fatal(err)
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if len(session.methods) != 7 {
		t.Fatalf("methods = %v, want seven complete calls", session.methods)
	}
	for index := range 4 {
		if session.methods[index] != "keyboardReport" {
			t.Fatalf("keyboard sequence interleaved: %v", session.methods)
		}
	}
	if session.methods[4] == "keyboardReport" || session.methods[5] == "keyboardReport" || session.methods[6] == "keyboardReport" {
		t.Fatalf("keyboard sequence interleaved: %v", session.methods)
	}
	if !(session.methods[4] == "relMouseReport" && session.methods[5] == "relMouseReport" && session.methods[6] == "unmountImage") &&
		!(session.methods[4] == "unmountImage" && session.methods[5] == "relMouseReport" && session.methods[6] == "relMouseReport") {
		t.Fatalf("mouse/media mutations interleaved: %v", session.methods)
	}
}

func TestManagerSerializesCompleteSameDeviceOperations(t *testing.T) {
	provider := &blockingProvider{entered: make(chan struct{}, 1), release: make(chan struct{}), session: admissionSession{}}
	close(provider.release)
	manager := admissionManager(t, Limits{
		MaxOperations: 2, MaxOperationsPerDevice: 2, MaxConnectionAttempts: 1,
		MaxCaptures: 1, MaxDecoders: 1,
	}, provider, "lab")
	device := manager.devices["lab"]
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- manager.withOperation(context.Background(), device, false, false, func(context.Context, Session) error {
			close(firstStarted)
			<-releaseFirst
			return nil
		})
	}()
	<-firstStarted

	secondStarted := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- manager.withOperation(context.Background(), device, false, false, func(context.Context, Session) error {
			close(secondStarted)
			return nil
		})
	}()
	select {
	case <-secondStarted:
		t.Fatal("same-device operation overlapped the complete first operation")
	case <-time.After(10 * time.Millisecond):
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
}

type sequenceProvider struct{ session *sequenceSession }

func (provider *sequenceProvider) Connect(_ context.Context, _ DeviceConfig) (ConnectedSession, error) {
	return testConnected(provider.session), nil
}

type sequenceSession struct {
	mu      sync.Mutex
	methods []string
	entered chan struct{}
	release chan struct{}
}

func (session *sequenceSession) Call(_ context.Context, method string, _ any, _ any) error {
	session.mu.Lock()
	session.methods = append(session.methods, method)
	first := len(session.methods) == 1
	session.mu.Unlock()
	if first {
		close(session.entered)
		<-session.release
	}
	return nil
}
func (*sequenceSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*sequenceSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, ErrDecoderUnavailable
}

func TestStatelessHTTPAdmissionRejectsExcessWorkWithoutSecondProviderDispatch(t *testing.T) {
	provider := &blockingProvider{entered: make(chan struct{}, 2), release: make(chan struct{}), session: admissionSession{}}
	manager := admissionManager(t, Limits{MaxOperations: 1, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 1, MaxCaptures: 1, MaxDecoders: 1}, provider, "lab")
	server := httptest.NewServer(mcpserver.NewHTTPHandler(mcpserver.New(manager, "test"), ""))
	defer server.Close()
	client, err := mcp.NewClient(&mcp.Implementation{Name: "admission-test", Version: "test"}, nil).Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: server.URL + mcpserver.MCPPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	first := make(chan error, 1)
	go func() {
		_, callErr := client.CallTool(context.Background(), &mcp.CallToolParams{Name: mcpserver.PressHostPowerButtonToolName, Arguments: map[string]any{"device": "lab"}})
		first <- callErr
	}()
	<-provider.entered
	result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: mcpserver.PressHostPowerButtonToolName, Arguments: map[string]any{"device": "lab"}})
	if err != nil || result == nil || !result.IsError || len(result.Content) != 1 {
		t.Fatalf("excess HTTP result = %#v, error = %v", result, err)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T, want text", result.Content[0])
	}
	var failure struct {
		Code    string `json:"code"`
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(text.Text), &failure); err != nil || failure.Code != "busy" || failure.Outcome != ToolOutcomeNotSent {
		t.Fatalf("busy failure = %q, error = %v", text.Text, err)
	}
	if provider.count() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.count())
	}
	close(provider.release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
}
