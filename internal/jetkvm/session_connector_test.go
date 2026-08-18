package jetkvm

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
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
	closeCalls int
	closeStart chan struct{}
	closeDone  chan struct{}
}

func (session *deterministicConnectedSession) Done() <-chan struct{} { return session.lost }

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
	for manager.owners["lab"].Snapshot().Ownership != ownerOwnershipIdle {
		select {
		case <-deadline:
			t.Fatal("controlled loss cleanup did not return owner to idle")
		default:
			time.Sleep(time.Millisecond)
		}
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
