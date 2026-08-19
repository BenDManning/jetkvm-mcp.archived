package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type mixedAcceptanceConnector struct {
	calls   atomic.Int32
	session *mixedAcceptanceSession
}

type lifecycleAcceptanceConnector struct {
	mu       sync.Mutex
	sessions []*lifecycleAcceptanceSession
	calls    int
}

func (connector *lifecycleAcceptanceConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	connector.mu.Lock()
	defer connector.mu.Unlock()
	session := connector.sessions[connector.calls]
	connector.calls++
	return session, nil
}

func (connector *lifecycleAcceptanceConnector) count() int {
	connector.mu.Lock()
	defer connector.mu.Unlock()
	return connector.calls
}

type lifecycleAcceptanceSession struct {
	done          chan struct{}
	closed        chan struct{}
	closeOnce     sync.Once
	closeErr      error
	takenOver     atomic.Bool
	callHook      func(context.Context, string) error
	mutationCalls atomic.Int32
}

func (session *lifecycleAcceptanceSession) Call(ctx context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	if session.callHook != nil {
		return session.callHook(ctx, method)
	}
	responses := map[string]any{
		methodLocalVersion:      map[string]any{"appVersion": "fixture", "systemVersion": "fixture"},
		methodActiveExtension:   "",
		methodVirtualMediaState: nil,
		methodVideoState:        map[string]any{"ready": true},
		methodUSBState:          "configured",
	}
	if result == nil {
		return nil
	}
	encoded, err := json.Marshal(responses[method])
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}

func (*lifecycleAcceptanceSession) Upload(context.Context, string, io.Reader, int64) error {
	return nil
}
func (*lifecycleAcceptanceSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *lifecycleAcceptanceSession) Done() <-chan struct{} { return session.done }
func (session *lifecycleAcceptanceSession) Close(context.Context) error {
	session.closeOnce.Do(func() { close(session.closed) })
	return session.closeErr
}
func (session *lifecycleAcceptanceSession) RecognizedTakeover() bool {
	return session.takenOver.Load()
}

func newLifecycleAcceptanceSession() *lifecycleAcceptanceSession {
	return &lifecycleAcceptanceSession{done: make(chan struct{}), closed: make(chan struct{})}
}

func newLifecycleAcceptanceManager(t *testing.T, connector SessionConnector, options ...ManagerOption) *Manager {
	t.Helper()
	base, _ := url.Parse("https://jetkvm.invalid")
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector, options...)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func assertAcceptanceError(t *testing.T, err error, code, outcome string) {
	t.Helper()
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != code || classified.Outcome != outcome || classified.Retryable {
		t.Fatalf("error = %#v, want code=%s outcome=%s nonretryable", err, code, outcome)
	}
}

func (connector *mixedAcceptanceConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	connector.calls.Add(1)
	return connector.session, nil
}

type mixedAcceptanceSession struct {
	done           chan struct{}
	captures       atomic.Int32
	overlapMu      sync.Mutex
	captureStarted chan struct{}
	statusStarted  chan struct{}
}

func (session *mixedAcceptanceSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodLocalVersion {
		session.overlapMu.Lock()
		statusStarted := session.statusStarted
		session.overlapMu.Unlock()
		if statusStarted != nil {
			close(statusStarted)
		}
	}
	responses := map[string]any{
		methodPing:              "pong",
		methodLocalVersion:      map[string]any{"appVersion": "fixture", "systemVersion": "fixture"},
		methodActiveExtension:   "",
		methodVirtualMediaState: nil,
		methodVideoState:        map[string]any{"ready": true, "streaming": 1, "width": 1, "height": 1, "fps": 1},
		methodUSBState:          "configured",
	}
	if result == nil {
		return nil
	}
	encoded, err := json.Marshal(responses[method])
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}

func (*mixedAcceptanceSession) Upload(context.Context, string, io.Reader, int64) error { return nil }

func (session *mixedAcceptanceSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	sequence := session.captures.Add(1)
	session.overlapMu.Lock()
	captureStarted, statusStarted := session.captureStarted, session.statusStarted
	session.overlapMu.Unlock()
	if captureStarted != nil {
		close(captureStarted)
		<-statusStarted
	}
	return []byte{0, 0, 0, 1, 0x65, byte(sequence)}, time.Unix(int64(sequence), 0).UTC(), nil
}

func (session *mixedAcceptanceSession) Done() <-chan struct{} { return session.done }
func (*mixedAcceptanceSession) Close(context.Context) error   { return nil }

type mixedAcceptanceDecoder struct {
	mu     sync.Mutex
	frames map[string]struct{}
	png    []byte
}

func (decoder *mixedAcceptanceDecoder) Decode(_ context.Context, frame []byte, _, _ int) ([]byte, int, int, error) {
	decoder.mu.Lock()
	decoder.frames[string(frame)] = struct{}{}
	decoder.mu.Unlock()
	return append([]byte(nil), decoder.png...), 1, 1, nil
}

func TestManagedSessionRepeatedMixedAndConcurrentReadsReuseOneGenerationWithFreshCaptures(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &mixedAcceptanceSession{done: make(chan struct{})}
	connector := &mixedAcceptanceConnector{session: session}
	decoder := &mixedAcceptanceDecoder{frames: make(map[string]struct{}), png: testPNG(t, 1, 1)}
	manager, err := NewManager(
		[]DeviceConfig{{Name: "lab", BaseURL: *base}},
		connector,
		WithDecoder(decoder),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	const sequential = 10
	for range sequential {
		if _, err := manager.Status(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaStatus}); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}); err != nil {
			t.Fatal(err)
		}
	}

	const concurrentPairs = 10
	for range concurrentPairs {
		captureStarted := make(chan struct{})
		statusStarted := make(chan struct{})
		session.overlapMu.Lock()
		session.captureStarted, session.statusStarted = captureStarted, statusStarted
		session.overlapMu.Unlock()
		results := make(chan error, 2)
		go func() {
			_, captureErr := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
			results <- captureErr
		}()
		<-captureStarted
		go func() {
			_, statusErr := manager.Status(context.Background(), "lab")
			results <- statusErr
		}()
		for range 2 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		session.overlapMu.Lock()
		session.captureStarted, session.statusStarted = nil, nil
		session.overlapMu.Unlock()
	}

	if got := connector.calls.Load(); got != 1 {
		t.Fatalf("connector attempts = %d, want one resident generation", got)
	}
	wantCaptures := sequential + concurrentPairs
	if got := int(session.captures.Load()); got != wantCaptures {
		t.Fatalf("captured frames = %d, want %d", got, wantCaptures)
	}
	decoder.mu.Lock()
	distinctFrames := len(decoder.frames)
	decoder.mu.Unlock()
	if distinctFrames != wantCaptures {
		t.Fatalf("distinct decoded frames = %d, want %d", distinctFrames, wantCaptures)
	}
}

func TestManagedSessionPublicLifecycleAcceptance(t *testing.T) {
	t.Run("release and takeover", func(t *testing.T) {
		first, second := newLifecycleAcceptanceSession(), newLifecycleAcceptanceSession()
		connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{first, second}}
		manager := newLifecycleAcceptanceManager(t, connector)
		t.Cleanup(func() { _ = manager.Close(context.Background()) })

		if _, err := manager.Status(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.ReleaseSession(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		<-first.closed
		_, err := manager.Status(context.Background(), "lab")
		assertAcceptanceError(t, err, "session_released", ToolOutcomeNotSent)
		if connector.count() != 1 {
			t.Fatal("released ordinary work reconnected")
		}
		if _, err := manager.TakeOverSession(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.Status(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		if connector.count() != 2 {
			t.Fatalf("connector attempts = %d, want one replacement generation", connector.count())
		}
	})

	for _, test := range []struct {
		name       string
		code       string
		recognized bool
	}{
		{name: "uncertain loss", code: "ownership_uncertain"},
		{name: "recognized takeover", code: "session_taken_over", recognized: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := newLifecycleAcceptanceSession()
			connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{session}}
			manager := newLifecycleAcceptanceManager(t, connector)
			t.Cleanup(func() { _ = manager.Close(context.Background()) })
			if _, err := manager.Status(context.Background(), "lab"); err != nil {
				t.Fatal(err)
			}
			session.takenOver.Store(test.recognized)
			close(session.done)
			<-session.closed
			_, err := manager.Status(context.Background(), "lab")
			assertAcceptanceError(t, err, test.code, ToolOutcomeNotSent)
			if connector.count() != 1 {
				t.Fatalf("sticky state opened %d connections", connector.count())
			}
		})
	}

	t.Run("idle expiry", func(t *testing.T) {
		first, second := newLifecycleAcceptanceSession(), newLifecycleAcceptanceSession()
		connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{first, second}}
		timers := make(chan *manualOwnerTimer, 2)
		manager := newLifecycleAcceptanceManager(t, connector, withOwnerAfterFunc(func(_ time.Duration, fire func()) ownerTimer {
			timer := &manualOwnerTimer{fire: fire}
			timers <- timer
			return timer
		}))
		t.Cleanup(func() { _ = manager.Close(context.Background()) })
		if _, err := manager.Status(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		(<-timers).fire()
		<-first.closed
		if _, err := manager.Status(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		if connector.count() != 2 {
			t.Fatalf("connector attempts = %d, want reacquisition after idle expiry", connector.count())
		}
	})

	t.Run("cleanup timeout", func(t *testing.T) {
		session := newLifecycleAcceptanceSession()
		session.closeErr = context.DeadlineExceeded
		connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{session}}
		manager := newLifecycleAcceptanceManager(t, connector)
		t.Cleanup(func() { _ = manager.Close(context.Background()) })
		if _, err := manager.Status(context.Background(), "lab"); err != nil {
			t.Fatal(err)
		}
		_, err := manager.ReleaseSession(context.Background(), "lab")
		assertAcceptanceError(t, err, "ownership_uncertain", ToolOutcomeFailed)
		_, err = manager.Status(context.Background(), "lab")
		assertAcceptanceError(t, err, "ownership_uncertain", ToolOutcomeNotSent)
		if connector.count() != 1 {
			t.Fatal("cleanup timeout allowed automatic reconnect")
		}
	})
}

type cancellationAcceptanceConnector struct {
	started  chan struct{}
	canceled chan struct{}
	calls    atomic.Int32
}

type registeredWaiterContext struct {
	context.Context
	doneCalls  atomic.Int32
	registered chan struct{}
	once       sync.Once
}

func (ctx *registeredWaiterContext) Done() <-chan struct{} {
	if ctx.doneCalls.Add(1) >= 2 {
		ctx.once.Do(func() { close(ctx.registered) })
	}
	return ctx.Context.Done()
}

func (connector *cancellationAcceptanceConnector) Connect(ctx context.Context, _ DeviceConfig) (ConnectedSession, error) {
	connector.calls.Add(1)
	close(connector.started)
	<-ctx.Done()
	close(connector.canceled)
	return nil, ctx.Err()
}

func TestManagedSessionCancellationAndShutdownAcceptance(t *testing.T) {
	t.Run("all waiters canceled", func(t *testing.T) {
		connector := &cancellationAcceptanceConnector{started: make(chan struct{}), canceled: make(chan struct{})}
		manager := newLifecycleAcceptanceManager(t, connector)
		t.Cleanup(func() { _ = manager.Close(context.Background()) })
		firstCtx, cancelFirst := context.WithCancel(context.Background())
		firstDone := make(chan error, 1)
		go func() {
			_, err := manager.Status(firstCtx, "lab")
			firstDone <- err
		}()
		<-connector.started

		secondBase, cancelSecond := context.WithCancel(context.Background())
		secondCtx := &registeredWaiterContext{Context: secondBase, registered: make(chan struct{})}
		secondDone := make(chan error, 1)
		go func() {
			_, err := manager.Status(secondCtx, "lab")
			secondDone <- err
		}()
		<-secondCtx.registered

		cancelFirst()
		firstErr := <-firstDone
		var classified *OperationError
		if !errors.As(firstErr, &classified) || classified.Outcome != ToolOutcomeNotSent {
			t.Fatalf("first canceled waiter error=%#v", firstErr)
		}
		select {
		case <-connector.canceled:
			t.Fatal("shared attempt canceled while second waiter remained")
		default:
		}

		cancelSecond()
		<-connector.canceled
		secondErr := <-secondDone
		if !errors.As(secondErr, &classified) || classified.Outcome != ToolOutcomeNotSent || connector.calls.Load() != 1 {
			t.Fatalf("second canceled waiter error=%#v calls=%d", secondErr, connector.calls.Load())
		}
	})

	t.Run("shutdown cancels active work", func(t *testing.T) {
		session := newLifecycleAcceptanceSession()
		started := make(chan struct{})
		session.callHook = func(ctx context.Context, method string) error {
			if method == methodLocalVersion {
				close(started)
				<-ctx.Done()
				return ctx.Err()
			}
			return nil
		}
		connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{session}}
		manager := newLifecycleAcceptanceManager(t, connector)
		done := make(chan error, 1)
		go func() {
			_, err := manager.Status(context.Background(), "lab")
			done <- err
		}()
		<-started
		if err := manager.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		<-session.closed
		assertAcceptanceError(t, <-done, "ownership_uncertain", ToolOutcomeFailed)
		_, err := manager.Status(context.Background(), "lab")
		if !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("post-shutdown error = %v", err)
		}
	})

	t.Run("shutdown closes resident generations in parallel", func(t *testing.T) {
		first := newParallelCloseSession()
		second := newParallelCloseSession()
		base, _ := url.Parse("https://jetkvm.invalid")
		connector := &deviceAcceptanceConnector{sessions: map[string]ConnectedSession{"alpha": first, "beta": second}}
		manager, err := NewManager([]DeviceConfig{{Name: "alpha", BaseURL: *base}, {Name: "beta", BaseURL: *base}}, connector)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := manager.Status(context.Background(), "alpha"); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.Status(context.Background(), "beta"); err != nil {
			t.Fatal(err)
		}
		closed := make(chan error, 1)
		go func() { closed <- manager.Close(context.Background()) }()
		<-first.closeStarted
		<-second.closeStarted
		close(first.releaseClose)
		close(second.releaseClose)
		if err := <-closed; err != nil {
			t.Fatal(err)
		}
	})

	t.Run("shutdown preserves dispatched mutation uncertainty", func(t *testing.T) {
		session := newLifecycleAcceptanceSession()
		started := make(chan struct{})
		session.callHook = func(ctx context.Context, _ string) error {
			session.mutationCalls.Add(1)
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}
		connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{session}}
		manager := newLifecycleAcceptanceManager(t, connector)
		mutationDone := make(chan error, 1)
		go func() {
			_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
			mutationDone <- err
		}()
		<-started
		if err := manager.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		err := <-mutationDone
		var classified *OperationError
		if !errors.As(err, &classified) || classified.Outcome != ToolOutcomeUnknown || classified.Retryable {
			t.Fatalf("shutdown mutation error = %#v", err)
		}
		if session.mutationCalls.Load() != 1 || connector.count() != 1 {
			t.Fatalf("mutation calls=%d connector attempts=%d", session.mutationCalls.Load(), connector.count())
		}
	})
}

type deviceAcceptanceConnector struct{ sessions map[string]ConnectedSession }

func (connector *deviceAcceptanceConnector) Connect(_ context.Context, device DeviceConfig) (ConnectedSession, error) {
	return connector.sessions[device.Name], nil
}

type parallelCloseSession struct {
	*lifecycleReadSession
	closeStarted chan struct{}
	releaseClose chan struct{}
}

type lifecycleReadSession struct{ done chan struct{} }

func (session *lifecycleReadSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	responses := map[string]any{
		methodLocalVersion:    map[string]any{"appVersion": "fixture", "systemVersion": "fixture"},
		methodActiveExtension: "", methodVirtualMediaState: nil,
		methodVideoState: map[string]any{"ready": true}, methodUSBState: "configured",
	}
	if result == nil {
		return nil
	}
	encoded, err := json.Marshal(responses[method])
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}
func (*lifecycleReadSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*lifecycleReadSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *lifecycleReadSession) Done() <-chan struct{} { return session.done }

func newParallelCloseSession() *parallelCloseSession {
	return &parallelCloseSession{
		lifecycleReadSession: &lifecycleReadSession{done: make(chan struct{})},
		closeStarted:         make(chan struct{}), releaseClose: make(chan struct{}),
	}
}

func (session *parallelCloseSession) Close(context.Context) error {
	close(session.closeStarted)
	<-session.releaseClose
	return nil
}

func TestManagedSessionMutationLossIsUnknownAndNeverReplayed(t *testing.T) {
	session := newLifecycleAcceptanceSession()
	started := make(chan struct{})
	session.callHook = func(ctx context.Context, _ string) error {
		session.mutationCalls.Add(1)
		close(started)
		<-ctx.Done()
		return ErrSessionClosed
	}
	connector := &lifecycleAcceptanceConnector{sessions: []*lifecycleAcceptanceSession{session}}
	manager := newLifecycleAcceptanceManager(t, connector)
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	done := make(chan error, 1)
	go func() {
		_, err := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
		done <- err
	}()
	<-started
	close(session.done)
	<-session.closed
	err := <-done
	assertAcceptanceError(t, err, "ownership_uncertain", ToolOutcomeUnknown)
	if session.mutationCalls.Load() != 1 || connector.count() != 1 {
		t.Fatalf("mutation calls=%d connector attempts=%d", session.mutationCalls.Load(), connector.count())
	}
}

type uploadAcceptanceSession struct {
	done          chan struct{}
	uploadStarted chan struct{}
	releaseUpload chan struct{}
}

func (session *uploadAcceptanceSession) Call(_ context.Context, method string, _ any, result any) error {
	responses := map[string]any{
		methodPing:               "pong",
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
		methodLocalVersion:       map[string]any{"appVersion": "fixture", "systemVersion": "fixture"},
		methodActiveExtension:    "",
		methodVirtualMediaState:  nil,
		methodVideoState:         map[string]any{"ready": true},
		methodUSBState:           "configured",
	}
	if result == nil {
		return nil
	}
	encoded, err := json.Marshal(responses[method])
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}

func (session *uploadAcceptanceSession) Upload(ctx context.Context, _ string, reader io.Reader, _ int64) error {
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	close(session.uploadStarted)
	select {
	case <-session.releaseUpload:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (*uploadAcceptanceSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *uploadAcceptanceSession) Done() <-chan struct{} { return session.done }
func (*uploadAcceptanceSession) Close(context.Context) error   { return nil }

func TestManagedSessionUploadOverlapAcceptance(t *testing.T) {
	mediaDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDirectory, "fixture.iso"), []byte("bounded fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &uploadAcceptanceSession{done: make(chan struct{}), uploadStarted: make(chan struct{}), releaseUpload: make(chan struct{})}
	connector := sessionConnectorFunc(func(context.Context, DeviceConfig) (ConnectedSession, error) { return session, nil })
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base, MediaDirectory: mediaDirectory}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	uploadDone := make(chan error, 1)
	go func() {
		_, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: "fixture.iso"})
		uploadDone <- err
	}()
	<-session.uploadStarted
	if _, err := manager.Status(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
	close(session.releaseUpload)
	if err := <-uploadDone; err != nil {
		t.Fatal(err)
	}
}
