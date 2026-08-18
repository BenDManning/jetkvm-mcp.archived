package jetkvm

import (
	"context"
	"errors"
	"io"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"
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
		if got := calledMethods(session.calls); !reflect.DeepEqual(got, []string{methodPing}) {
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
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled operation error = %v", err)
	}

	if _, err := manager.DebugRPC(context.Background(), "lab", methodPing, nil, false); err != nil {
		t.Fatalf("current operation failed: %v", err)
	}
	before := manager.owners["lab"].Snapshot()
	close(connector.release)
	<-connector.closed
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
}

func (session *shutdownSession) Call(ctx context.Context, _ string, _ any, _ any) error {
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
