package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

type schedulingSession struct {
	mu          sync.Mutex
	methods     []string
	runningRPCs int
	maximumRPCs int
	firstRPC    chan struct{}
	releaseRPC  chan struct{}
	readRPC     chan struct{}
}

func (session *schedulingSession) Call(ctx context.Context, method string, _ any, result any) error {
	session.mu.Lock()
	session.methods = append(session.methods, method)
	session.runningRPCs++
	if session.runningRPCs > session.maximumRPCs {
		session.maximumRPCs = session.runningRPCs
	}
	session.mu.Unlock()

	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
	} else if method == "mutation-one" {
		close(session.firstRPC)
		select {
		case <-session.releaseRPC:
		case <-ctx.Done():
			return ctx.Err()
		}
	} else if method == "read" {
		close(session.readRPC)
	}

	session.mu.Lock()
	session.runningRPCs--
	session.mu.Unlock()
	return nil
}

func (*schedulingSession) Upload(context.Context, string, io.Reader, int64) error {
	return nil
}

func (*schedulingSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}

type schedulingConnector struct{ session Session }

func (connector *schedulingConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	return testConnected(connector.session), nil
}

func newSchedulingManager(t *testing.T, session Session) *Manager {
	t.Helper()
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &schedulingConnector{session: session})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	return manager
}

func TestInitialCapabilityProfileInterleavesReadAtMutationRPCBoundary(t *testing.T) {
	session := &schedulingSession{
		firstRPC: make(chan struct{}), releaseRPC: make(chan struct{}), readRPC: make(chan struct{}),
	}
	manager := newSchedulingManager(t, session)
	device := manager.devices["lab"]
	betweenRPCs := make(chan struct{})
	releaseMutation := make(chan struct{})
	mutationDone := make(chan error, 1)
	go func() {
		mutationDone <- manager.withOperation(context.Background(), device, true, false, func(ctx context.Context, session Session) error {
			if err := session.Call(ctx, "mutation-one", nil, nil); err != nil {
				return err
			}
			close(betweenRPCs)
			<-releaseMutation
			return session.Call(ctx, "mutation-two", nil, nil)
		})
	}()
	<-session.firstRPC
	close(session.releaseRPC)
	<-betweenRPCs

	readDone := make(chan error, 1)
	go func() {
		readDone <- manager.withOperation(context.Background(), device, false, false, func(ctx context.Context, session Session) error {
			return session.Call(ctx, "read", nil, nil)
		})
	}()
	<-session.readRPC
	close(releaseMutation)
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	if err := <-mutationDone; err != nil {
		t.Fatal(err)
	}
	if session.maximumRPCs != 1 {
		t.Fatalf("maximum outstanding firmware RPCs = %d, want 1", session.maximumRPCs)
	}
}

type hidSafetySession struct {
	pressed        chan struct{}
	neutralizing   chan struct{}
	releaseNeutral chan struct{}
	read           chan struct{}
}

type overlapSession struct {
	frameStarted chan struct{}
	releaseFrame chan struct{}
	read         chan struct{}
	capturedAt   time.Time
}

func (session *overlapSession) Call(_ context.Context, method string, _ any, result any) error {
	switch method {
	case methodPing:
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
	case methodVideoState:
		return json.Unmarshal([]byte(`{"ready":true}`), result)
	case "read":
		close(session.read)
	}
	return nil
}

func (*overlapSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (session *overlapSession) CaptureH264(ctx context.Context) ([]byte, time.Time, error) {
	close(session.frameStarted)
	select {
	case <-session.releaseFrame:
		return []byte{0, 0, 0, 1, 0x65}, session.capturedAt, nil
	case <-ctx.Done():
		return nil, time.Time{}, ctx.Err()
	}
}

func newOverlapManager(t *testing.T, session Session, options ...ManagerOption) *Manager {
	t.Helper()
	base, _ := url.Parse("https://jetkvm.invalid")
	options = append(options, WithDecoder(&fakeDecoder{png: testPNG(t, 1, 1), width: 1, height: 1}))
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &schedulingConnector{session: session}, options...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	return manager
}

func TestInitialCapabilityProfileOverlapsOneRPCWithFrameAcquisition(t *testing.T) {
	session := &overlapSession{
		frameStarted: make(chan struct{}), releaseFrame: make(chan struct{}), read: make(chan struct{}), capturedAt: time.Now().UTC(),
	}
	manager := newOverlapManager(t, session)
	captureDone := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		captureDone <- err
	}()
	<-session.frameStarted
	readDone := make(chan error, 1)
	go func() {
		readDone <- manager.withOperation(context.Background(), manager.devices["lab"], false, false, func(ctx context.Context, session Session) error {
			return session.Call(ctx, "read", nil, nil)
		})
	}()
	<-session.read
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	close(session.releaseFrame)
	if err := <-captureDone; err != nil {
		t.Fatal(err)
	}
}

func TestSessionWideFallbackDoesNotOverlapRPCAndFrameAcquisition(t *testing.T) {
	session := &overlapSession{
		frameStarted: make(chan struct{}), releaseFrame: make(chan struct{}), read: make(chan struct{}), capturedAt: time.Now().UTC(),
	}
	manager := newOverlapManager(t, session, withCapabilityProfile(capabilityProfile{revision: 2, schedule: capabilityScheduleSessionWide}))
	captureDone := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		captureDone <- err
	}()
	<-session.frameStarted
	readDone := make(chan error, 1)
	go func() {
		readDone <- manager.withOperation(context.Background(), manager.devices["lab"], false, false, func(ctx context.Context, session Session) error {
			return session.Call(ctx, "read", nil, nil)
		})
	}()
	select {
	case <-session.read:
		t.Fatal("session-wide fallback overlapped RPC with frame acquisition")
	default:
	}
	close(session.releaseFrame)
	if err := <-captureDone; err != nil {
		t.Fatal(err)
	}
	<-session.read
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	if revision := manager.owners["lab"].Snapshot().CapabilityProfileRevision; revision != 2 {
		t.Fatalf("capability profile revision = %d, want 2", revision)
	}
}

type leaseSession struct {
	done     chan struct{}
	closed   chan struct{}
	frame    []byte
	captured time.Time
}

func (session *leaseSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	if method == methodVideoState {
		return json.Unmarshal([]byte(`{"ready":true}`), result)
	}
	return nil
}

func (*leaseSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (session *leaseSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return session.frame, session.captured, nil
}
func (session *leaseSession) Done() <-chan struct{} { return session.done }
func (session *leaseSession) Close(context.Context) error {
	select {
	case <-session.closed:
	default:
		close(session.closed)
	}
	return nil
}

type leaseConnector struct{ session *leaseSession }

func (connector *leaseConnector) Connect(context.Context, DeviceConfig) (ConnectedSession, error) {
	return connector.session, nil
}

type blockingCaptureDecoder struct {
	started chan []byte
	release chan struct{}
	png     []byte
}

func (decoder *blockingCaptureDecoder) Decode(_ context.Context, frame []byte, _, _ int) ([]byte, int, int, error) {
	decoder.started <- frame
	<-decoder.release
	return append([]byte(nil), decoder.png...), 1, 1, nil
}

func TestCaptureReleasesGenerationLeaseAfterImmutableFrameCopy(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	original := []byte{0, 0, 0, 1, 0x65}
	session := &leaseSession{done: make(chan struct{}), closed: make(chan struct{}), frame: original, captured: time.Now().UTC()}
	decoder := &blockingCaptureDecoder{started: make(chan []byte, 1), release: make(chan struct{}), png: testPNG(t, 1, 1)}
	timers := make(chan *manualOwnerTimer, 1)
	manager, err := NewManager(
		[]DeviceConfig{{Name: "lab", BaseURL: *base}},
		&leaseConnector{session: session},
		WithDecoder(decoder),
		withOwnerAfterFunc(func(_ time.Duration, fire func()) ownerTimer {
			timer := &manualOwnerTimer{fire: fire}
			timers <- timer
			return timer
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	captureDone := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		captureDone <- err
	}()
	decodedFrame := <-decoder.started
	original[4] = 0
	if decodedFrame[4] != 0x65 {
		t.Fatalf("decoder frame changed with transport buffer: %x", decodedFrame)
	}
	(<-timers).fire()
	<-session.closed
	select {
	case err := <-captureDone:
		t.Fatalf("capture completed before decoder release: %v", err)
	default:
	}
	close(decoder.release)
	if err := <-captureDone; err != nil {
		t.Fatal(err)
	}
}

type queuedCaptureSession struct {
	mu       sync.Mutex
	frames   [][]byte
	started  chan int
	releases []chan struct{}
	next     int
}

func (session *queuedCaptureSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	if method == methodVideoState {
		return json.Unmarshal([]byte(`{"ready":true}`), result)
	}
	return nil
}

func (*queuedCaptureSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (session *queuedCaptureSession) CaptureH264(ctx context.Context) ([]byte, time.Time, error) {
	session.mu.Lock()
	index := session.next
	session.next++
	frame := session.frames[index]
	releaseFrame := session.releases[index]
	session.mu.Unlock()
	session.started <- index
	select {
	case <-releaseFrame:
		return frame, time.Unix(int64(index+1), 0).UTC(), nil
	case <-ctx.Done():
		return nil, time.Time{}, ctx.Err()
	}
}

type recordingDecoder struct {
	mu     sync.Mutex
	frames [][]byte
	png    []byte
}

func (decoder *recordingDecoder) Decode(_ context.Context, frame []byte, _, _ int) ([]byte, int, int, error) {
	decoder.mu.Lock()
	decoder.frames = append(decoder.frames, append([]byte(nil), frame...))
	decoder.mu.Unlock()
	return append([]byte(nil), decoder.png...), 1, 1, nil
}

func TestCapturesQueueFreshFramesAndRemoveCanceledWaiter(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &queuedCaptureSession{
		frames:  [][]byte{{0, 0, 0, 1, 0x61}, {0, 0, 0, 1, 0x62}},
		started: make(chan int, 2), releases: []chan struct{}{make(chan struct{}), make(chan struct{})},
	}
	decoder := &recordingDecoder{png: testPNG(t, 1, 1)}
	limits := DefaultLimits()
	limits.MaxCaptures = 3
	limits.MaxDecoders = 3
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &schedulingConnector{session: session}, WithLimits(limits), WithDecoder(decoder))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	firstDone := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		firstDone <- err
	}()
	if index := <-session.started; index != 0 {
		t.Fatalf("first capture index = %d", index)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	canceledDone := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(canceledCtx, "lab", mcpserver.CaptureRequest{})
		canceledDone <- err
	}()
	cancel()
	canceledErr := <-canceledDone
	var operationErr *OperationError
	if !errors.As(canceledErr, &operationErr) || operationErr.Outcome != ToolOutcomeNotSent {
		t.Fatalf("canceled queued capture = %v, want not_sent", canceledErr)
	}

	secondDone := make(chan error, 1)
	go func() {
		_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
		secondDone <- err
	}()
	close(session.releases[0])
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if index := <-session.started; index != 1 {
		t.Fatalf("second capture index = %d", index)
	}
	close(session.releases[1])
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	decoder.mu.Lock()
	defer decoder.mu.Unlock()
	if !reflect.DeepEqual(decoder.frames, session.frames) {
		t.Fatalf("decoded frames = %x, want distinct fresh frames %x", decoder.frames, session.frames)
	}
}

type uploadOverlapSession struct {
	uploadStarted chan struct{}
	releaseUpload chan struct{}
	read          chan struct{}
}

func (session *uploadOverlapSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
	} else if method == "read" {
		close(session.read)
	}
	return nil
}
func (session *uploadOverlapSession) Upload(ctx context.Context, _ string, _ io.Reader, _ int64) error {
	close(session.uploadStarted)
	select {
	case <-session.releaseUpload:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (*uploadOverlapSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}

func TestInitialCapabilityProfileAllowsReadDuringBoundedUpload(t *testing.T) {
	session := &uploadOverlapSession{uploadStarted: make(chan struct{}), releaseUpload: make(chan struct{}), read: make(chan struct{})}
	manager := newSchedulingManager(t, session)
	device := manager.devices["lab"]
	uploadDone := make(chan error, 1)
	go func() {
		uploadDone <- manager.withOperation(context.Background(), device, true, false, func(ctx context.Context, session Session) error {
			if err := session.Call(ctx, "start-upload", nil, nil); err != nil {
				return err
			}
			return session.Upload(ctx, "upload", strings.NewReader("bounded"), 7)
		})
	}()
	<-session.uploadStarted
	readDone := make(chan error, 1)
	go func() {
		readDone <- manager.withOperation(context.Background(), device, false, false, func(ctx context.Context, session Session) error {
			return session.Call(ctx, "read", nil, nil)
		})
	}()
	<-session.read
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	close(session.releaseUpload)
	if err := <-uploadDone; err != nil {
		t.Fatal(err)
	}
}

type statusLossSession struct {
	done chan struct{}
}

func (session *statusLossSession) Call(ctx context.Context, method string, _ any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	if method == methodLocalVersion {
		close(session.done)
		return ErrSessionClosed
	}
	return nil
}
func (*statusLossSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*statusLossSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *statusLossSession) Done() <-chan struct{} { return session.done }
func (*statusLossSession) Close(context.Context) error   { return nil }

func TestStatusFailsWholeOperationWhenGenerationEndsDuringProbes(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &statusLossSession{done: make(chan struct{})}
	connector := sessionConnectorFunc(func(context.Context, DeviceConfig) (ConnectedSession, error) { return session, nil })
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, connector)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	status, err := manager.Status(context.Background(), "lab")
	if err == nil || status.Connected {
		t.Fatalf("status = %+v, error = %v; want whole-operation failure", status, err)
	}
}

type sessionConnectorFunc func(context.Context, DeviceConfig) (ConnectedSession, error)

func (connect sessionConnectorFunc) Connect(ctx context.Context, device DeviceConfig) (ConnectedSession, error) {
	return connect(ctx, device)
}

func (session *hidSafetySession) Call(ctx context.Context, method string, params any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	if method == "read" {
		close(session.read)
		return nil
	}
	if method != "keyboardReport" {
		return nil
	}
	report, _ := params.(map[string]any)
	if reflect.DeepEqual(report["keys"], []int{}) {
		close(session.neutralizing)
		<-session.releaseNeutral
		return nil
	}
	close(session.pressed)
	<-ctx.Done()
	return ctx.Err()
}

func (*hidSafetySession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*hidSafetySession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}

func TestHIDCancellationRetainsRPCGateThroughNeutralization(t *testing.T) {
	session := &hidSafetySession{
		pressed: make(chan struct{}), neutralizing: make(chan struct{}), releaseNeutral: make(chan struct{}), read: make(chan struct{}),
	}
	manager := newSchedulingManager(t, session)
	ctx, cancel := context.WithCancel(context.Background())
	hidDone := make(chan error, 1)
	go func() {
		_, err := manager.Keyboard(ctx, "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardPressKey, Key: "a"})
		hidDone <- err
	}()
	<-session.pressed
	readDone := make(chan error, 1)
	go func() {
		readDone <- manager.withOperation(context.Background(), manager.devices["lab"], false, false, func(ctx context.Context, session Session) error {
			return session.Call(ctx, "read", nil, nil)
		})
	}()
	cancel()
	<-session.neutralizing
	select {
	case <-session.read:
		t.Fatal("read entered while HID neutralization retained the RPC gate")
	default:
	}
	close(session.releaseNeutral)
	<-session.read
	if err := <-hidDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("HID cancellation error = %v, want canceled", err)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
}

type terminalHIDSession struct {
	done        chan struct{}
	neutralized chan struct{}
	suppress    bool
}

func (session *terminalHIDSession) Call(_ context.Context, method string, params any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	if method == "keyboardReport" {
		report, _ := params.(map[string]any)
		if reflect.DeepEqual(report["keys"], []int{}) {
			close(session.neutralized)
			return nil
		}
		close(session.done)
		return ErrSessionClosed
	}
	return nil
}
func (*terminalHIDSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*terminalHIDSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}
func (session *terminalHIDSession) Done() <-chan struct{}    { return session.done }
func (*terminalHIDSession) Close(context.Context) error      { return nil }
func (session *terminalHIDSession) SuppressHIDCleanup() bool { return session.suppress }

func TestHIDGenerationEndStillAttemptsNeutralization(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &terminalHIDSession{done: make(chan struct{}), neutralized: make(chan struct{})}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, sessionConnectorFunc(func(context.Context, DeviceConfig) (ConnectedSession, error) {
		return session, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if _, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardPressKey, Key: "a"}); !errors.Is(err, ErrOwnershipUncertain) {
		t.Fatalf("terminal HID error = %v, want ownership uncertain", err)
	}
	<-session.neutralized
}

func TestHIDRecognizedTakeoverSuppressesNeutralization(t *testing.T) {
	base, _ := url.Parse("https://jetkvm.invalid")
	session := &terminalHIDSession{done: make(chan struct{}), neutralized: make(chan struct{}), suppress: true}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, sessionConnectorFunc(func(context.Context, DeviceConfig) (ConnectedSession, error) {
		return session, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	_, _ = manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardPressKey, Key: "a"})
	select {
	case <-session.neutralized:
		t.Fatal("recognized takeover received a post-takeover neutralization RPC")
	default:
	}
}

type panicHIDSession struct{ neutralized chan struct{} }

func (session *panicHIDSession) Call(_ context.Context, method string, params any, result any) error {
	if method == methodPing {
		if pong, ok := result.(*string); ok {
			*pong = "pong"
		}
		return nil
	}
	report, _ := params.(map[string]any)
	if reflect.DeepEqual(report["keys"], []int{}) {
		close(session.neutralized)
		return nil
	}
	panic("controlled HID panic")
}
func (*panicHIDSession) Upload(context.Context, string, io.Reader, int64) error { return nil }
func (*panicHIDSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}

func TestHIDPanicNeutralizesBeforeReleasingRPCGate(t *testing.T) {
	session := &panicHIDSession{neutralized: make(chan struct{})}
	manager := newSchedulingManager(t, session)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("HID operation did not propagate controlled panic")
			}
		}()
		_, _ = manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardPressKey, Key: "a"})
	}()
	<-session.neutralized
}
