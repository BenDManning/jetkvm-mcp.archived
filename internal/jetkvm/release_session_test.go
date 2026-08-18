package jetkvm

import (
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

func TestManagerReleaseSessionFromIdleIsStickyAndIdempotentWithoutConnecting(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	connector := new(deterministicConnector)
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	for range 2 {
		result, releaseErr := manager.ReleaseSession(context.Background(), "lab")
		if releaseErr != nil {
			t.Fatal(releaseErr)
		}
		want := (mcpserver.SessionReleaseResult{Device: "lab", Status: mcpserver.SessionStatusReleased})
		if result != want {
			t.Fatalf("release result = %#v, want %#v", result, want)
		}
	}
	if connector.calls != 0 {
		t.Fatalf("idle/idempotent release opened %d connections", connector.calls)
	}

	_, err = manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "session_released" || classified.Outcome != ToolOutcomeNotSent || classified.Retryable {
		t.Fatalf("ordinary work after release = %#v, want nonretryable session_released/not_sent", err)
	}
	if connector.calls != 0 {
		t.Fatalf("ordinary work after release opened %d connections", connector.calls)
	}
}

func TestDeviceOwnerReleaseSessionFromTakenOverDoesNotConnect(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	connector := new(deterministicConnector)
	owner := newDeviceOwnerWithSettings(DeviceConfig{Name: "lab", BaseURL: *base}, connector, ownerSettings{
		initialOwnership: ownerOwnershipTakenOver,
	})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })

	if err := owner.Release(); err != nil {
		t.Fatal(err)
	}
	if connector.calls != 0 || owner.Snapshot().Ownership != ownerOwnershipReleased {
		t.Fatalf("connects = %d snapshot = %+v", connector.calls, owner.Snapshot())
	}
}

type releaseAdmissionSession struct {
	started      chan struct{}
	finish       chan struct{}
	done         chan struct{}
	closeStarted chan struct{}
	closeFinish  chan struct{}
}

type releaseConnector struct{ session ConnectedSession }

func (connector *releaseConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	return connector.session, nil
}

func (session *releaseAdmissionSession) Call(ctx context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	select {
	case <-session.started:
	default:
		close(session.started)
	}
	select {
	case <-session.finish:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (*releaseAdmissionSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*releaseAdmissionSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *releaseAdmissionSession) Done() <-chan struct{} { return session.done }
func (session *releaseAdmissionSession) Close(ctx context.Context) error {
	if session.closeStarted != nil {
		close(session.closeStarted)
	}
	if session.closeFinish != nil {
		select {
		case <-session.closeFinish:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestManagerReleaseSessionClosesActiveGenerationBeforeReportingReleased(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &residentConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	result, err := manager.ReleaseSession(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != mcpserver.SessionStatusReleased {
		t.Fatalf("release result = %#v", result)
	}
	session.mu.Lock()
	closes := session.closes
	session.mu.Unlock()
	if closes != 1 || manager.owners["lab"].Snapshot().Ownership != ownerOwnershipReleased {
		t.Fatalf("close calls = %d snapshot = %+v", closes, manager.owners["lab"].Snapshot())
	}
}

func TestManagerReleaseSessionRetriesUncertainCleanup(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{}), closeErr: context.DeadlineExceeded}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	_, err = manager.ReleaseSession(context.Background(), "lab")
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "ownership_uncertain" || classified.Outcome != ToolOutcomeFailed || classified.Retryable {
		t.Fatalf("failed cleanup release = %#v", err)
	}
	if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipUncertain {
		t.Fatalf("snapshot = %+v", manager.owners["lab"].Snapshot())
	}
	_, err = manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
	if !errors.As(err, &classified) || classified.Code != "ownership_uncertain" || classified.Outcome != ToolOutcomeNotSent || classified.Retryable {
		t.Fatalf("ordinary work after failed cleanup = %#v", err)
	}

	session.closeErr = nil
	if _, err := manager.ReleaseSession(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
	if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipReleased {
		t.Fatalf("retry snapshot = %+v", manager.owners["lab"].Snapshot())
	}
	session.mu.Lock()
	closes := session.closes
	session.mu.Unlock()
	if closes != 2 {
		t.Fatalf("cleanup calls = %d, want 2", closes)
	}
}

func TestManagerReleaseSessionRejectsWhileDeviceWorkIsAdmitted(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &releaseAdmissionSession{started: make(chan struct{}), finish: make(chan struct{}), done: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	operationDone := make(chan error, 1)
	go func() {
		_, operationErr := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
		operationDone <- operationErr
	}()
	<-session.started
	_, err = manager.ReleaseSession(context.Background(), "lab")
	assertBusyNotSent(t, err)
	if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipActive {
		t.Fatalf("busy release changed owner = %+v", manager.owners["lab"].Snapshot())
	}
	close(session.finish)
	if err := <-operationDone; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ReleaseSession(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
}

func TestManagerReleaseSessionLatchesLifecycleBeforeCleanup(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &releaseAdmissionSession{
		started: make(chan struct{}), finish: make(chan struct{}), done: make(chan struct{}),
		closeStarted: make(chan struct{}), closeFinish: make(chan struct{}),
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}

	releaseDone := make(chan error, 1)
	go func() {
		_, releaseErr := manager.ReleaseSession(context.Background(), "lab")
		releaseDone <- releaseErr
	}()
	<-session.closeStarted
	_, err = manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
	assertBusyNotSent(t, err)
	close(session.closeFinish)
	if err := <-releaseDone; err != nil {
		t.Fatal(err)
	}
	_, err = manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "session_released" || classified.Outcome != ToolOutcomeNotSent {
		t.Fatalf("post-release work = %#v", err)
	}
}

func TestManagerReleaseSessionCancellationIsPreDispatchOrDetachedAfterLatch(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	preCanceled, cancelPreCanceled := context.WithCancel(context.Background())
	cancelPreCanceled()
	idleManager, err := NewManager([]DeviceConfig{{Name: "idle", BaseURL: *base}}, new(deterministicConnector))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idleManager.Close(context.Background()) })
	_, err = idleManager.ReleaseSession(preCanceled, "idle")
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "canceled" || classified.Outcome != ToolOutcomeNotSent || idleManager.owners["idle"].Snapshot().Ownership != ownerOwnershipIdle {
		t.Fatalf("pre-canceled release = %#v snapshot = %+v", err, idleManager.owners["idle"].Snapshot())
	}

	session := &releaseAdmissionSession{
		started: make(chan struct{}), finish: make(chan struct{}), done: make(chan struct{}),
		closeStarted: make(chan struct{}), closeFinish: make(chan struct{}),
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	releaseCtx, cancelRelease := context.WithCancel(context.Background())
	releaseDone := make(chan error, 1)
	go func() {
		_, releaseErr := manager.ReleaseSession(releaseCtx, "lab")
		releaseDone <- releaseErr
	}()
	<-session.closeStarted
	cancelRelease()
	close(session.closeFinish)
	if err := <-releaseDone; err != nil {
		t.Fatalf("latched release was canceled: %v", err)
	}
	if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipReleased {
		t.Fatalf("latched release snapshot = %+v", manager.owners["lab"].Snapshot())
	}
}

func TestManagerReleaseSessionRequiresGlobalCapacity(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &releaseAdmissionSession{started: make(chan struct{}), finish: make(chan struct{}), done: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}, {Name: "rack", BaseURL: *base}}, &releaseConnector{session: session}, WithLimits(Limits{
		MaxOperations: 1, MaxOperationsPerDevice: 1, MaxConnectionAttempts: 1, MaxCaptures: 1, MaxDecoders: 1,
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	operationDone := make(chan error, 1)
	go func() {
		_, operationErr := manager.Power(context.Background(), "rack", mcpserver.PowerActionPressHostPowerButton, "")
		operationDone <- operationErr
	}()
	<-session.started
	_, err = manager.ReleaseSession(context.Background(), "lab")
	assertBusyNotSent(t, err)
	if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipIdle {
		t.Fatalf("capacity rejection changed idle owner = %+v", manager.owners["lab"].Snapshot())
	}
	close(session.finish)
	if err := <-operationDone; err != nil {
		t.Fatal(err)
	}
}

func TestManagerShutdownPreservesReleasedOwnershipWithoutConnecting(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	connector := new(deterministicConnector)
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ReleaseSession(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if connector.calls != 0 || manager.owners["lab"].Snapshot().Ownership != ownerOwnershipStopped {
		t.Fatalf("connects = %d snapshot = %+v", connector.calls, manager.owners["lab"].Snapshot())
	}
	_, err = manager.ReleaseSession(context.Background(), "lab")
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("post-shutdown release = %v, want session closed", err)
	}
}
