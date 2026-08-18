package jetkvm

import (
	"context"
	"errors"
	"io"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type validationSession struct {
	mu                sync.Mutex
	pings             int
	done              chan struct{}
	validationStarted chan struct{}
	validationFinish  chan struct{}
}

type sessionAndErrorConnector struct {
	session ConnectedSession
	err     error
	calls   int
}

func (connector *sessionAndErrorConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	connector.calls++
	return connector.session, connector.err
}

func (session *validationSession) Call(ctx context.Context, method string, _ any, result any) error {
	if method != methodPing {
		return nil
	}
	session.mu.Lock()
	session.pings++
	pings := session.pings
	session.mu.Unlock()
	if pings > 1 {
		close(session.validationStarted)
		select {
		case <-session.validationFinish:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if pong, ok := result.(*string); ok {
		*pong = "pong"
	}
	return nil
}
func (*validationSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*validationSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *validationSession) Done() <-chan struct{} { return session.done }
func (*validationSession) Close(context.Context) error   { return nil }

func TestManagerTakeOverSessionAcquiresOnceFromIdleAndReusesGeneration(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &residentConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	result, err := manager.TakeOverSession(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	want := (mcpserver.SessionTakeoverResult{Device: "lab", Status: mcpserver.SessionStatusAuthoritative})
	if result != want {
		t.Fatalf("takeover result = %#v, want %#v", result, want)
	}
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	if connects != 1 {
		t.Fatalf("takeover plus ordinary reuse opened %d connections", connects)
	}
}

func TestManagerTakeOverSessionReacquiresFromStickyOwnership(t *testing.T) {
	for _, test := range []struct {
		name      string
		takenOver bool
		wantState ownerOwnership
	}{
		{name: "taken over", takenOver: true, wantState: ownerOwnershipTakenOver},
		{name: "uncertain", wantState: ownerOwnershipUncertain},
	} {
		t.Run(test.name, func(t *testing.T) {
			base, _ := url.Parse("https://jetkvm.invalid")
			first := &deterministicConnectedSession{
				fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
				lost:        make(chan struct{}),
			}
			second := &residentSession{done: make(chan struct{})}
			connector := &residentSequenceConnector{sessions: []ConnectedSession{first, second}}
			manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = manager.Close(context.Background()) })
			if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
				t.Fatal(err)
			}
			first.takenOver = test.takenOver
			close(first.lost)
			deadline := time.After(time.Second)
			for manager.owners["lab"].Snapshot().Ownership != test.wantState {
				select {
				case <-deadline:
					t.Fatalf("sticky snapshot = %+v", manager.owners["lab"].Snapshot())
				default:
					time.Sleep(time.Millisecond)
				}
			}
			if _, err := manager.TakeOverSession(context.Background(), "lab"); err != nil {
				t.Fatal(err)
			}
			if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipActive || connector.calls != 2 {
				t.Fatalf("snapshot=%+v connects=%d", manager.owners["lab"].Snapshot(), connector.calls)
			}
		})
	}
}

func TestManagerTakeOverSessionDoesNotConnectWhenPriorCleanupFails(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	first := &residentSession{done: make(chan struct{}), closeErr: context.DeadlineExceeded}
	second := &residentSession{done: make(chan struct{})}
	connector := &residentSequenceConnector{sessions: []ConnectedSession{first, second}}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	close(first.done)
	deadline := time.After(time.Second)
	for manager.owners["lab"].Snapshot().Ownership != ownerOwnershipUncertain {
		select {
		case <-deadline:
			t.Fatal("cleanup failure did not latch uncertainty")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	_, err = manager.TakeOverSession(context.Background(), "lab")
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "ownership_uncertain" || classified.Outcome != ToolOutcomeFailed || classified.Retryable {
		t.Fatalf("takeover cleanup failure = %#v", err)
	}
	if connector.calls != 1 {
		t.Fatalf("cleanup failure opened replacement connection: calls=%d", connector.calls)
	}
}

func TestManagerTakeOverSessionReportsUncertaintyWhenFailedSetupCannotCleanUp(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{}), closeErr: context.DeadlineExceeded}
	connector := &sessionAndErrorConnector{session: session, err: ErrDeviceUnreachable}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	_, err = manager.TakeOverSession(context.Background(), "lab")
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "ownership_uncertain" || classified.Outcome != ToolOutcomeFailed || classified.Retryable {
		t.Fatalf("failed setup cleanup = %#v", err)
	}
	if connector.calls != 1 || manager.owners["lab"].Snapshot().Ownership != ownerOwnershipUncertain {
		t.Fatalf("connects=%d snapshot=%+v", connector.calls, manager.owners["lab"].Snapshot())
	}
}

func TestManagerTakeOverSessionWaitsForAutomaticStickyCleanupBeforeConnecting(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	first := &deterministicConnectedSession{
		fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
		lost:        make(chan struct{}), closeStart: make(chan struct{}), closeDone: make(chan struct{}),
	}
	second := &residentSession{done: make(chan struct{})}
	connector := &residentSequenceConnector{sessions: []ConnectedSession{first, second}}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	first.takenOver = true
	close(first.lost)
	<-first.closeStart
	takeoverDone := make(chan error, 1)
	go func() {
		_, takeoverErr := manager.TakeOverSession(context.Background(), "lab")
		takeoverDone <- takeoverErr
	}()
	connector.mu.Lock()
	connectsBeforeCleanup := connector.calls
	connector.mu.Unlock()
	if connectsBeforeCleanup != 1 {
		t.Fatalf("replacement connected before cleanup joined: calls=%d", connectsBeforeCleanup)
	}
	close(first.closeDone)
	if err := <-takeoverDone; err != nil {
		connector.mu.Lock()
		connects := connector.calls
		connector.mu.Unlock()
		t.Fatalf("takeover error=%#v connects=%d ops=%d attempts=%d snapshot=%+v", err, connects, len(manager.operations), len(manager.connectionAttempts), manager.owners["lab"].Snapshot())
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	if connects != 2 || manager.owners["lab"].Snapshot().Ownership != ownerOwnershipActive {
		t.Fatalf("connects=%d snapshot=%+v", connects, manager.owners["lab"].Snapshot())
	}
}

func TestManagerTakeOverSessionAcquiresFromReleasedWithoutOrdinaryReconnect(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &residentConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if _, err := manager.ReleaseSession(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); !errors.Is(err, ErrSessionReleased) {
		t.Fatalf("ordinary work while released = %v", err)
	}
	if _, err := manager.TakeOverSession(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	if connects != 1 || manager.owners["lab"].Snapshot().Ownership != ownerOwnershipActive {
		t.Fatalf("connects=%d snapshot=%+v", connects, manager.owners["lab"].Snapshot())
	}
}

func TestManagerTakeOverSessionValidatesHealthyGenerationWithoutReplacingIt(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &residentConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.TakeOverSession(context.Background(), "lab"); err != nil {
		t.Fatal(err)
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	session.mu.Lock()
	pings, closes := session.pings, session.closes
	session.mu.Unlock()
	if connects != 1 || pings != 2 || closes != 0 {
		t.Fatalf("connects=%d pings=%d closes=%d, want one connection and validation ping", connects, pings, closes)
	}
}

func TestManagerTakeOverSessionRejectsConflictingLifecycleWithoutQueueing(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &validationSession{done: make(chan struct{}), validationStarted: make(chan struct{}), validationFinish: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	takeoverDone := make(chan error, 1)
	go func() {
		_, takeoverErr := manager.TakeOverSession(context.Background(), "lab")
		takeoverDone <- takeoverErr
	}()
	<-session.validationStarted
	_, err = manager.ReleaseSession(context.Background(), "lab")
	assertBusyNotSent(t, err)
	_, err = manager.TakeOverSession(context.Background(), "lab")
	assertBusyNotSent(t, err)
	close(session.validationFinish)
	if err := <-takeoverDone; err != nil {
		t.Fatal(err)
	}
}

func TestManagerTakeOverSessionValidationLossDoesNotReconnectInCall(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &validationSession{done: make(chan struct{}), validationStarted: make(chan struct{}), validationFinish: make(chan struct{})}
	connector := &releaseConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, takeoverErr := manager.TakeOverSession(context.Background(), "lab")
		result <- takeoverErr
	}()
	<-session.validationStarted
	close(session.done)
	var classified *OperationError
	if err := <-result; !errors.As(err, &classified) || classified.Code != "ownership_uncertain" || classified.Outcome != ToolOutcomeFailed || classified.Retryable {
		t.Fatalf("validation loss = %#v", err)
	}
	if manager.owners["lab"].Snapshot().Ownership != ownerOwnershipUncertain {
		t.Fatalf("validation loss snapshot = %+v", manager.owners["lab"].Snapshot())
	}
}
