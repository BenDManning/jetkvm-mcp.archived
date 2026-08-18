package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type deterministicConnector struct {
	session *deterministicConnectedSession
	err     error
	calls   int
}

func (connector *deterministicConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	connector.calls++
	if connector.err != nil {
		return nil, connector.err
	}
	return connector.session, nil
}

type deterministicConnectedSession struct {
	*fakeSession
	lost       chan struct{}
	takenOver  bool
	closeCalls int
	closeStart chan struct{}
	closeDone  chan struct{}
}

func (session *deterministicConnectedSession) Done() <-chan struct{}    { return session.lost }
func (session *deterministicConnectedSession) RecognizedTakeover() bool { return session.takenOver }

func (session *deterministicConnectedSession) Close(ctx context.Context) error {
	session.closeCalls++
	if session.closeStart != nil {
		select {
		case <-session.closeStart:
		default:
			close(session.closeStart)
		}
	}
	if session.closeDone != nil {
		select {
		case <-session.closeDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type lateTakeoverSession struct {
	*deterministicConnectedSession
	takeover   chan struct{}
	recognized atomic.Bool
}

func (session *lateTakeoverSession) RecognizedTakeover() bool          { return session.recognized.Load() }
func (session *lateTakeoverSession) TakeoverDetected() <-chan struct{} { return session.takeover }

func (session *lateTakeoverSession) recognizeTakeover() {
	session.recognized.Store(true)
	close(session.takeover)
}

func TestRecognizedTakeoverLatchesAndRejectsOrdinaryReconnect(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &deterministicConnectedSession{
		fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
		lost:        make(chan struct{}),
	}
	connector := &deterministicConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	session.takenOver = true
	close(session.lost)
	deadline := time.After(time.Second)
	for manager.owners["lab"].Snapshot().Ownership != ownerOwnershipTakenOver {
		select {
		case <-deadline:
			t.Fatal("recognized takeover did not latch taken-over ownership")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	_, err = manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "session_taken_over" || classified.Outcome != ToolOutcomeNotSent || classified.Retryable {
		t.Fatalf("ordinary work after takeover = %#v", err)
	}
	if connector.calls != 1 {
		t.Fatalf("ordinary work after takeover opened %d connections", connector.calls)
	}
}

func TestRecognizedTakeoverWinsWhenLossWasProcessedFirst(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &lateTakeoverSession{
		deterministicConnectedSession: &deterministicConnectedSession{
			fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
			lost:        make(chan struct{}), closeStart: make(chan struct{}), closeDone: make(chan struct{}),
		},
		takeover: make(chan struct{}),
	}
	connector := sessionConnectorFunc(func(context.Context, DeviceConfig) (ConnectedSession, error) {
		return session, nil
	})
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
	})
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	close(session.lost)
	select {
	case <-session.closeStart:
	case <-time.After(time.Second):
		t.Fatal("terminal cleanup did not start")
	}
	deadline := time.After(time.Second)
	for manager.owners["lab"].Snapshot().Ownership != ownerOwnershipUncertain {
		select {
		case <-deadline:
			t.Fatal("loss was not classified as ownership uncertainty")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	close(session.closeDone)
	deadline = time.After(time.Second)
	for {
		snapshot := manager.owners["lab"].Snapshot()
		if snapshot.Ownership == ownerOwnershipUncertain && snapshot.Health == ownerHealthUnavailable {
			break
		}
		select {
		case <-deadline:
			t.Fatal("terminal cleanup did not complete with ownership uncertainty")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	session.recognizeTakeover()
	deadline = time.After(time.Second)
	for manager.owners["lab"].Snapshot().Ownership != ownerOwnershipTakenOver {
		select {
		case <-deadline:
			t.Fatal("late takeover recognition did not replace ownership uncertainty")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

type terminalOperationSession struct {
	started   chan struct{}
	done      chan struct{}
	takenOver bool
}

type conclusiveMutationSession struct {
	started chan struct{}
	finish  chan struct{}
	done    chan struct{}
}

type takeoverFenceSession struct {
	done              chan struct{}
	takenOver         bool
	postTakeoverCalls atomic.Int32
}

func (session *takeoverFenceSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	session.postTakeoverCalls.Add(1)
	return nil
}
func (*takeoverFenceSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*takeoverFenceSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *takeoverFenceSession) Done() <-chan struct{}    { return session.done }
func (*takeoverFenceSession) Close(context.Context) error      { return nil }
func (session *takeoverFenceSession) RecognizedTakeover() bool { return session.takenOver }

func (session *conclusiveMutationSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	close(session.started)
	<-session.finish
	return nil
}
func (*conclusiveMutationSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*conclusiveMutationSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *conclusiveMutationSession) Done() <-chan struct{} { return session.done }
func (*conclusiveMutationSession) Close(context.Context) error   { return nil }

func (session *terminalOperationSession) Call(ctx context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	close(session.started)
	<-ctx.Done()
	return ErrSessionClosed
}
func (*terminalOperationSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*terminalOperationSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *terminalOperationSession) Done() <-chan struct{}    { return session.done }
func (*terminalOperationSession) Close(context.Context) error      { return nil }
func (session *terminalOperationSession) RecognizedTakeover() bool { return session.takenOver }
func (session *terminalOperationSession) SuppressHIDCleanup() bool { return session.takenOver }

func TestTerminalGenerationClassifiesInconclusiveDispatchByOwnershipAndOperation(t *testing.T) {
	for _, test := range []struct {
		name        string
		takenOver   bool
		mutation    bool
		wantCode    string
		wantOutcome string
	}{
		{name: "takeover read", takenOver: true, wantCode: "session_taken_over", wantOutcome: ToolOutcomeFailed},
		{name: "loss read", wantCode: "ownership_uncertain", wantOutcome: ToolOutcomeFailed},
		{name: "takeover mutation", takenOver: true, mutation: true, wantCode: "session_taken_over", wantOutcome: ToolOutcomeUnknown},
		{name: "loss mutation", mutation: true, wantCode: "ownership_uncertain", wantOutcome: ToolOutcomeUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			base, _ := url.Parse("https://jetkvm.invalid")
			session := &terminalOperationSession{started: make(chan struct{}), done: make(chan struct{})}
			manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = manager.Close(context.Background()) })
			result := make(chan error, 1)
			go func() {
				if test.mutation {
					_, callErr := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
					result <- callErr
					return
				}
				_, callErr := manager.DebugRPC(context.Background(), "lab", methodLocalVersion, json.RawMessage(`{}`), false)
				result <- callErr
			}()
			<-session.started
			session.takenOver = test.takenOver
			close(session.done)
			var classified *OperationError
			if err := <-result; !errors.As(err, &classified) || classified.Code != test.wantCode || classified.Outcome != test.wantOutcome || classified.Retryable {
				t.Fatalf("terminal operation = %#v, want %s/%s nonretryable", err, test.wantCode, test.wantOutcome)
			}
		})
	}
}

func TestConclusiveMutationCompletionAfterGenerationLossRemainsSuccessful(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &conclusiveMutationSession{started: make(chan struct{}), finish: make(chan struct{}), done: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	result := make(chan error, 1)
	go func() {
		_, callErr := manager.Power(context.Background(), "lab", mcpserver.PowerActionPressHostPowerButton, "")
		result <- callErr
	}()
	<-session.started
	close(session.done)
	close(session.finish)
	if err := <-result; err != nil {
		t.Fatalf("conclusive mutation was discarded after loss: %v", err)
	}
}

func TestRecognizedTakeoverFencesNewOwnerDispatchBeforeTerminalCommand(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &takeoverFenceSession{done: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &releaseConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.owners["lab"].Run(context.Background(), func(context.Context, Session) error { return nil }); err != nil {
		t.Fatal(err)
	}
	session.takenOver = true
	close(session.done)
	_, err = manager.DebugRPC(context.Background(), "lab", methodLocalVersion, json.RawMessage(`{}`), false)
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "session_taken_over" || classified.Outcome != ToolOutcomeNotSent || classified.Retryable {
		t.Fatalf("post-takeover work = %#v", err)
	}
	if calls := session.postTakeoverCalls.Load(); calls != 0 {
		t.Fatalf("recognized takeover allowed %d new RPC dispatches", calls)
	}
}

func TestDeterministicConnectorControlsFailureLossAndCleanupCompletion(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	newManager := func(connector SessionConnector) *Manager {
		manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = manager.Close(context.Background()) })
		return manager
	}
	connectErr := errors.New("controlled connect failure")
	manager := newManager(&deterministicConnector{err: connectErr})
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); !errors.Is(err, connectErr) {
		t.Fatalf("connect error = %v, want controlled failure", err)
	}
	pingErr := errors.New("controlled ping failure")
	manager = newManager(&deterministicConnector{session: &deterministicConnectedSession{
		fakeSession: &fakeSession{err: map[string]error{methodPing: pingErr}},
		lost:        make(chan struct{}),
	}})
	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); !errors.Is(err, pingErr) {
		t.Fatalf("ping error = %v, want controlled RPC failure", err)
	}

	session := &deterministicConnectedSession{
		fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
		lost:        make(chan struct{}),
		closeStart:  make(chan struct{}),
		closeDone:   make(chan struct{}),
	}
	manager = newManager(&deterministicConnector{session: session})
	result := make(chan error, 1)
	go func() {
		_, callErr := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
		result <- callErr
	}()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	close(session.lost)
	<-session.closeStart
	select {
	case <-session.Done():
	default:
		t.Fatal("controlled session loss was not observable")
	}
	close(session.closeDone)
	deadline := time.After(time.Second)
	for manager.owners["lab"].Snapshot().Ownership != ownerOwnershipUncertain {
		select {
		case <-deadline:
			t.Fatal("controlled loss cleanup did not latch ownership uncertainty")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	_, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false)
	var classified *OperationError
	if !errors.As(err, &classified) || classified.Code != "ownership_uncertain" || classified.Outcome != ToolOutcomeNotSent || classified.Retryable {
		t.Fatalf("ordinary work after loss = %#v", err)
	}
	if connectorCalls := manager.owners["lab"].connector.(*deterministicConnector).calls; connectorCalls != 1 {
		t.Fatalf("ordinary work after loss opened %d connections", connectorCalls)
	}
}

func TestConnectedSessionCleanupCanBeRetriedAfterCallerContextExpires(t *testing.T) {
	connectedCtx, cancelConnected := context.WithCancel(context.Background())
	connected := &connectedSession{ctx: connectedCtx, cancel: cancelConnected}
	releasePump := make(chan struct{})
	connected.pumps.Add(1)
	go func() {
		<-releasePump
		connected.pumps.Done()
	}()

	expired, cancelExpired := context.WithCancel(context.Background())
	cancelExpired()
	if err := connected.Close(expired); !errors.Is(err, context.Canceled) {
		t.Fatalf("expired cleanup error = %v, want canceled", err)
	}
	select {
	case <-connected.Done():
	default:
		t.Fatal("cleanup did not close the session after caller cancellation")
	}
	close(releasePump)
	if err := connected.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestManagerUsesConnectorAndClosesResidentSessionAtShutdown(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &deterministicConnectedSession{
		fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
		lost:        make(chan struct{}),
	}
	connector := &deterministicConnector{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	if connector.calls != 1 || session.closeCalls != 0 {
		t.Fatalf("connect calls = %d, close calls = %d; want one resident session", connector.calls, session.closeCalls)
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if session.closeCalls != 1 {
		t.Fatalf("shutdown close calls = %d, want 1", session.closeCalls)
	}
}
