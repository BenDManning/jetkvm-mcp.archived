package jetkvm

import (
	"context"
	"errors"
	"io"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type routingConnector struct {
	mu       sync.Mutex
	devices  []string
	sessions []*routingSession
}

func (connector *routingConnector) Connect(_ context.Context, device DeviceConfig) (ConnectedSession, error) {
	session := &routingSession{
		fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
		done:        make(chan struct{}),
	}
	connector.mu.Lock()
	connector.devices = append(connector.devices, device.Name)
	connector.sessions = append(connector.sessions, session)
	connector.mu.Unlock()
	return session, nil
}

type routingSession struct {
	*fakeSession
	done   chan struct{}
	closed chan struct{}
}

type residentConnector struct {
	mu      sync.Mutex
	calls   int
	session *residentSession
}

func (connector *residentConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	connector.mu.Lock()
	defer connector.mu.Unlock()
	connector.calls++
	return connector.session, nil
}

type residentSession struct {
	mu       sync.Mutex
	pings    int
	closes   int
	done     chan struct{}
	closeErr error
	pingErr  error
}

func (session *residentSession) Call(_ context.Context, method string, _ any, result any) error {
	if method != methodPing {
		return nil
	}
	session.mu.Lock()
	session.pings++
	session.mu.Unlock()
	if session.pingErr != nil {
		return session.pingErr
	}
	if pong, ok := result.(*string); ok {
		*pong = "pong"
	}
	return nil
}

func (*residentSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*residentSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *residentSession) Done() <-chan struct{} { return session.done }
func (session *residentSession) Close(context.Context) error {
	session.mu.Lock()
	session.closes++
	session.mu.Unlock()
	return session.closeErr
}

func TestDeviceOwnerPingValidatesAndReusesResidentGeneration(t *testing.T) {
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
	for range cap(manager.connectionAttempts) {
		manager.connectionAttempts <- struct{}{}
	}
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatalf("healthy generation consumed a connection-attempt permit: %v", err)
	}
	for range cap(manager.connectionAttempts) {
		<-manager.connectionAttempts
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	session.mu.Lock()
	pings, closes := session.pings, session.closes
	session.mu.Unlock()
	if connects != 1 || pings != 1 || closes != 0 {
		t.Fatalf("connects=%d pings=%d closes=%d, want one validated resident generation", connects, pings, closes)
	}
	snapshot := manager.owners["lab"].Snapshot()
	if snapshot.Ownership != ownerOwnershipActive || snapshot.Transition != ownerTransitionNone || snapshot.Health != ownerHealthHealthy {
		t.Fatalf("owner snapshot = %+v", snapshot)
	}
}

type sharedAttemptConnector struct {
	started chan struct{}
	release chan struct{}
	session ConnectedSession
	calls   atomic.Int32
}

type residentSequenceConnector struct {
	mu       sync.Mutex
	sessions []ConnectedSession
	calls    int
}

func (connector *residentSequenceConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	connector.mu.Lock()
	defer connector.mu.Unlock()
	session := connector.sessions[connector.calls]
	connector.calls++
	return session, nil
}

func (connector *sharedAttemptConnector) Connect(ctx context.Context, _ DeviceConfig) (ConnectedSession, error) {
	connector.calls.Add(1)
	select {
	case connector.started <- struct{}{}:
	default:
	}
	select {
	case <-connector.release:
		return connector.session, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func registerOwnerOperation(t *testing.T, owner *deviceOwner, ctx context.Context, operation func(context.Context, Session) error) ownerRegistration {
	t.Helper()
	command := registerOwnerWorker{ctx: ctx, operation: operation, reply: make(chan ownerRegistration, 1)}
	owner.commands <- command
	registered := <-command.reply
	if registered.err != nil {
		t.Fatal(registered.err)
	}
	return registered
}

func TestDeviceOwnerSharesSetupAndSerializesSameDeviceWork(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &sharedAttemptConnector{
		started: make(chan struct{}, 1), release: make(chan struct{}), session: session,
	}
	owner := newDeviceOwner(DeviceConfig{Name: "lab", BaseURL: *base}, connector)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })

	var running, maximum atomic.Int32
	operation := func(context.Context, Session) error {
		current := running.Add(1)
		for previous := maximum.Load(); current > previous && !maximum.CompareAndSwap(previous, current); previous = maximum.Load() {
		}
		running.Add(-1)
		return nil
	}
	first := registerOwnerOperation(t, owner, context.Background(), operation)
	second := registerOwnerOperation(t, owner, context.Background(), operation)
	<-connector.started
	close(connector.release)
	if result := <-first.reply; result.err != nil {
		t.Fatal(result.err)
	}
	if result := <-second.reply; result.err != nil {
		t.Fatal(result.err)
	}
	if connector.calls.Load() != 1 || maximum.Load() != 1 {
		t.Fatalf("connects=%d concurrent operations=%d, want one shared setup and serialization", connector.calls.Load(), maximum.Load())
	}
}

type canceledAttemptConnector struct {
	started chan struct{}
	session *residentSession
	calls   atomic.Int32
}

func (connector *canceledAttemptConnector) Connect(ctx context.Context, _ DeviceConfig) (ConnectedSession, error) {
	connector.calls.Add(1)
	close(connector.started)
	<-ctx.Done()
	return connector.session, nil
}

func TestDeviceOwnerCancelsAndCleansSetupAfterEveryWaiterLeaves(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &canceledAttemptConnector{started: make(chan struct{}), session: session}
	owner := newDeviceOwner(DeviceConfig{Name: "lab", BaseURL: *base}, connector)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })

	firstCtx, cancelFirst := context.WithCancel(context.Background())
	secondCtx, cancelSecond := context.WithCancel(context.Background())
	first := registerOwnerOperation(t, owner, firstCtx, func(context.Context, Session) error { return nil })
	second := registerOwnerOperation(t, owner, secondCtx, func(context.Context, Session) error { return nil })
	<-connector.started
	cancelFirst()
	firstCancellation := cancelOwnerWorker{evidence: first.evidence, err: context.Canceled, reply: make(chan bool, 1)}
	owner.commands <- firstCancellation
	if !<-firstCancellation.reply {
		t.Fatal("first waiter was not canceled before dispatch")
	}
	if connector.calls.Load() != 1 {
		t.Fatalf("connection attempt ended while another waiter remained: calls=%d", connector.calls.Load())
	}
	cancelSecond()
	secondCancellation := cancelOwnerWorker{evidence: second.evidence, err: context.Canceled, reply: make(chan bool, 1)}
	owner.commands <- secondCancellation
	if <-secondCancellation.reply {
		t.Fatal("last waiter returned before canceled setup cleanup completed")
	}

	deadline := time.After(time.Second)
	for {
		session.mu.Lock()
		closes := session.closes
		session.mu.Unlock()
		if closes == 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("canceled setup did not close its late session")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if result := <-first.reply; !errors.Is(result.err, context.Canceled) {
		t.Fatalf("first result = %v", result.err)
	}
	if result := <-second.reply; !errors.Is(result.err, context.Canceled) {
		t.Fatalf("second result = %v", result.err)
	}
}

type manualOwnerTimer struct {
	fire    func()
	stopped atomic.Bool
}

func (timer *manualOwnerTimer) Stop() bool { return timer.stopped.CompareAndSwap(false, true) }

func TestDeviceOwnerIdleLeaseStartsAfterWorkAndReleasesGeneration(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &residentSession{done: make(chan struct{})}
	connector := &residentConnector{session: session}
	timers := make(chan *manualOwnerTimer, 1)
	owner := newDeviceOwnerWithSettings(DeviceConfig{Name: "lab", BaseURL: *base}, connector, ownerSettings{
		idleTimeout: time.Minute,
		afterFunc: func(_ time.Duration, fire func()) ownerTimer {
			timer := &manualOwnerTimer{fire: fire}
			timers <- timer
			return timer
		},
	})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })

	operationStarted := make(chan struct{})
	releaseOperation := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- owner.Run(context.Background(), func(context.Context, Session) error {
			close(operationStarted)
			<-releaseOperation
			return nil
		})
	}()
	<-operationStarted
	select {
	case <-timers:
		t.Fatal("idle lease started while generation-bound work was active")
	default:
	}
	close(releaseOperation)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	timer := <-timers
	timer.fire()

	deadline := time.After(time.Second)
	for {
		snapshot := owner.Snapshot()
		session.mu.Lock()
		closes := session.closes
		session.mu.Unlock()
		if closes == 1 && snapshot.Ownership == ownerOwnershipIdle && snapshot.Transition == ownerTransitionNone {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("idle cleanup incomplete: closes=%d snapshot=%+v", closes, snapshot)
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestDeviceOwnerDoesNotReconnectAfterIncompleteIdleCleanup(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	cleanupErr := errors.New("controlled cleanup failure")
	session := &residentSession{done: make(chan struct{}), closeErr: cleanupErr}
	connector := &residentConnector{session: session}
	timers := make(chan *manualOwnerTimer, 1)
	owner := newDeviceOwnerWithSettings(DeviceConfig{Name: "lab", BaseURL: *base}, connector, ownerSettings{
		idleTimeout: time.Minute,
		afterFunc: func(_ time.Duration, fire func()) ownerTimer {
			timer := &manualOwnerTimer{fire: fire}
			timers <- timer
			return timer
		},
	})

	if err := owner.Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	(<-timers).fire()
	deadline := time.After(time.Second)
	for {
		snapshot := owner.Snapshot()
		if snapshot.Transition == ownerTransitionClosing && snapshot.Health == ownerHealthDegraded {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("cleanup failure was not latched: %+v", snapshot)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := owner.Run(context.Background(), func(context.Context, Session) error { return nil }); !errors.Is(err, ErrBusy) {
		t.Fatalf("ordinary work after incomplete cleanup = %v, want busy", err)
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	if connects != 1 {
		t.Fatalf("incomplete cleanup allowed %d connection attempts", connects)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := owner.Close(closeCtx); !errors.Is(err, cleanupErr) {
		t.Fatalf("shutdown cleanup error = %v, want %v", err, cleanupErr)
	}
}

func TestDeviceOwnerDoesNotReconnectAfterIncompleteSetupCleanup(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	pingErr := errors.New("controlled readiness failure")
	cleanupErr := errors.New("controlled setup cleanup failure")
	session := &residentSession{done: make(chan struct{}), pingErr: pingErr, closeErr: cleanupErr}
	connector := &residentConnector{session: session}
	owner := newDeviceOwner(DeviceConfig{Name: "lab", BaseURL: *base}, connector)
	if err := owner.Run(context.Background(), func(context.Context, Session) error { return nil }); !errors.Is(err, pingErr) {
		t.Fatalf("setup error = %v, want %v", err, pingErr)
	}
	snapshot := owner.Snapshot()
	if snapshot.Transition != ownerTransitionClosing || snapshot.Health != ownerHealthDegraded {
		t.Fatalf("setup cleanup failure was not latched: %+v", snapshot)
	}
	if err := owner.Run(context.Background(), func(context.Context, Session) error { return nil }); !errors.Is(err, ErrBusy) {
		t.Fatalf("ordinary work after incomplete setup cleanup = %v, want busy", err)
	}
	connector.mu.Lock()
	connects := connector.calls
	connector.mu.Unlock()
	if connects != 1 {
		t.Fatalf("incomplete setup cleanup allowed %d attempts", connects)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := owner.Close(closeCtx); !errors.Is(err, cleanupErr) {
		t.Fatalf("shutdown setup cleanup error = %v, want %v", err, cleanupErr)
	}
}

func TestDeviceOwnerIgnoresWorkerCompletionFromOlderGeneration(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	firstSession := &residentSession{done: make(chan struct{})}
	secondSession := &residentSession{done: make(chan struct{})}
	connector := &residentSequenceConnector{sessions: []ConnectedSession{firstSession, secondSession}}
	timers := make(chan *manualOwnerTimer, 2)
	owner := newDeviceOwnerWithSettings(DeviceConfig{Name: "lab", BaseURL: *base}, connector, ownerSettings{
		idleTimeout: time.Minute,
		afterFunc: func(_ time.Duration, fire func()) ownerTimer {
			timer := &manualOwnerTimer{fire: fire}
			timers <- timer
			return timer
		},
	})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })

	first := registerOwnerOperation(t, owner, context.Background(), func(context.Context, Session) error { return nil })
	if result := <-first.reply; result.err != nil {
		t.Fatal(result.err)
	}
	(<-timers).fire()
	deadline := time.After(time.Second)
	for owner.Snapshot().Ownership != ownerOwnershipIdle {
		select {
		case <-deadline:
			t.Fatal("first generation did not become idle")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := owner.Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	deadline = time.After(time.Second)
	for owner.Snapshot().Observations.Completed != 2 {
		select {
		case <-deadline:
			t.Fatal("replacement operation observation was not published")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	before := owner.Snapshot()
	staleEvidence := first.evidence
	staleEvidence.generation = 1
	staleEvidence.dispatch = ownerDispatchCompleted
	owner.commands <- completeOwnerWorker{completion: ownerCompletion{
		evidence: staleEvidence,
		err:      errors.New("stale completion sentinel"),
	}}
	time.Sleep(10 * time.Millisecond)
	after := owner.Snapshot()
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("stale completion changed replacement generation: before=%+v after=%+v", before, after)
	}
}

func (session *routingSession) Done() <-chan struct{} { return session.done }
func (session *routingSession) Close(context.Context) error {
	if session.closed != nil {
		close(session.closed)
		session.closed = nil
	}
	return nil
}

func TestManagerCreatesOneOwnerPerConfiguredDeviceAndRoutesWorkThroughIt(t *testing.T) {
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	connector := new(routingConnector)
	manager, err := NewManager([]DeviceConfig{
		{Name: "lab", BaseURL: *base},
		{Name: "rack", BaseURL: *base},
	}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	if len(manager.owners) != 2 || manager.owners["lab"] == nil || manager.owners["rack"] == nil || manager.owners["lab"] == manager.owners["rack"] {
		t.Fatalf("owners = %#v, want one distinct owner per device", manager.owners)
	}
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.DebugRPC(context.Background(), "rack", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	connector.mu.Lock()
	defer connector.mu.Unlock()
	if !reflect.DeepEqual(connector.devices, []string{"lab", "rack"}) {
		t.Fatalf("connector devices = %v", connector.devices)
	}
	for index, session := range connector.sessions {
		if got := calledMethods(session.calls); !reflect.DeepEqual(got, []string{methodPing, methodPing}) {
			t.Fatalf("session %d methods = %v", index, got)
		}
	}
}

type cancellationConnector struct {
	started chan struct{}
	release chan struct{}
	closed  chan struct{}
	mu      sync.Mutex
	calls   int
}

func (connector *cancellationConnector) Connect(_ context.Context, _ DeviceConfig) (ConnectedSession, error) {
	connector.mu.Lock()
	connector.calls++
	call := connector.calls
	connector.mu.Unlock()
	if call == 1 {
		close(connector.started)
		<-connector.release
		return &routingSession{
			fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
			done:        make(chan struct{}),
			closed:      connector.closed,
		}, nil
	}
	return &routingSession{
		fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
		done:        make(chan struct{}),
	}, nil
}

func TestDeviceOwnerFencesLateConnectionAfterPreDispatchCancellation(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	connector := &cancellationConnector{
		started: make(chan struct{}), release: make(chan struct{}), closed: make(chan struct{}),
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() {
		_, callErr := manager.DebugRPC(ctx, "lab", methodPing, nil, false)
		first <- callErr
	}()
	<-connector.started
	cancel()
	close(connector.release)
	<-connector.closed
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled operation error = %v", err)
	}

	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatalf("current operation failed: %v", err)
	}
	before := manager.owners["lab"].Snapshot()
	after := manager.owners["lab"].Snapshot()
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("stale completion changed owner snapshot: before=%+v after=%+v", before, after)
	}
}

type shutdownConnector struct {
	started chan struct{}
	closed  chan struct{}
}

func (connector *shutdownConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	return &shutdownSession{started: connector.started, closed: connector.closed, done: make(chan struct{})}, nil
}

type shutdownSession struct {
	started chan struct{}
	closed  chan struct{}
	done    chan struct{}
	ready   atomic.Bool
}

func (session *shutdownSession) Call(ctx context.Context, method string, _ any, result any) error {
	if method == methodPing && session.ready.CompareAndSwap(false, true) {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	close(session.started)
	<-ctx.Done()
	return ctx.Err()
}
func (*shutdownSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*shutdownSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *shutdownSession) Done() <-chan struct{} { return session.done }
func (session *shutdownSession) Close(context.Context) error {
	close(session.closed)
	return nil
}

func TestManagerShutdownCancelsWorkersAndStopsFurtherDispatch(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	connector := &shutdownConnector{started: make(chan struct{}), closed: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}

	callDone := make(chan error, 1)
	go func() {
		_, callErr := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
		callDone <- callErr
	}()
	<-connector.started
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	<-connector.closed
	if err := <-callDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("shutdown operation error = %v", err)
	}
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("post-shutdown error = %v, want session closed", err)
	}
	snapshot := manager.owners["lab"].Snapshot()
	if snapshot.Ownership != ownerOwnershipStopped || snapshot.Transition != ownerTransitionNone {
		t.Fatalf("shutdown snapshot = %+v", snapshot)
	}
}

type shutdownDecoder struct {
	started chan struct{}
	stopped chan struct{}
}

func (decoder *shutdownDecoder) Decode(ctx context.Context, _ []byte, _, _ int) ([]byte, int, int, error) {
	close(decoder.started)
	<-ctx.Done()
	close(decoder.stopped)
	return nil, 0, 0, ctx.Err()
}

func TestManagerShutdownCancelsAndJoinsActiveDecoder(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	ready := true
	session := &fakeSession{
		results:    map[string]any{methodVideoState: map[string]any{"ready": ready}},
		h264:       []byte{0, 0, 0, 1, 0x65},
		capturedAt: time.Now().UTC(),
	}
	decoder := &shutdownDecoder{started: make(chan struct{}), stopped: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &fakeProvider{session: session}, WithDecoder(decoder))
	if err != nil {
		t.Fatal(err)
	}
	captureDone := make(chan error, 1)
	go func() {
		_, captureErr := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		captureDone <- captureErr
	}()
	<-decoder.started
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-decoder.stopped:
	default:
		t.Fatal("manager shutdown returned before decoder stopped")
	}
	if err := <-captureDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("capture shutdown error = %v, want canceled", err)
	}
}
