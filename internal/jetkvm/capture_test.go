package jetkvm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

func TestVideoReceiverCapturesFreshDecodableIDRAndRequestsPLI(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	var pliCalls atomic.Int32
	receiver.SetPLI(func() error { pliCalls.Add(1); return nil })
	now := time.Now().UTC()
	receiver.Observe(rtpPacket(0, 99, true, []byte{0x61, 0}), now.Add(-time.Nanosecond))
	receiver.Observe(rtpPacket(1, 100, true, stapA(
		[]byte{0x67, 0x42, 0x00, 0x1f}, []byte{0x68, 0xce, 0x06, 0xe2},
	)), now)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	type capture struct {
		data []byte
		at   time.Time
		err  error
	}
	done := make(chan capture, 1)
	go func() {
		data, at, err := receiver.Capture(ctx)
		done <- capture{data: data, at: at, err: err}
	}()

	deadline := time.Now().Add(time.Second)
	for !receiver.Waiting() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !receiver.Waiting() {
		t.Fatal("capture waiter did not register")
	}
	receiver.Observe(rtpPacket(2, 101, true, []byte{0x65, 0x88, 0x84}), now.Add(time.Millisecond))

	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if pliCalls.Load() != 1 || !bytes.Equal(result.data, captureAnnexB(
		[]byte{0x67, 0x42, 0x00, 0x1f}, []byte{0x68, 0xce, 0x06, 0xe2}, []byte{0x65, 0x88, 0x84},
	)) || result.at.IsZero() {
		t.Fatalf("pli=%d data=%x at=%v", pliCalls.Load(), result.data, result.at)
	}
}

func captureAnnexB(nalus ...[]byte) []byte {
	var result []byte
	for _, nalu := range nalus {
		result = append(result, 0, 0, 0, 1)
		result = append(result, nalu...)
	}
	return result
}

func TestVideoReceiverAllowsOnlyOneWaiter(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() { _, _, err := receiver.Capture(ctx); first <- err }()
	deadline := time.Now().Add(time.Second)
	for !receiver.Waiting() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if _, _, err := receiver.Capture(context.Background()); !errors.Is(err, ErrVideoBusy) {
		t.Fatalf("second capture error = %v", err)
	}
	cancel()
	<-first
}

func TestManagerCaptureScreenUsesVideoSessionAndDecoder(t *testing.T) {
	capturedAt := time.Date(2026, time.August, 9, 12, 34, 56, 0, time.UTC)
	session := &fakeSession{results: map[string]any{}, h264: []byte{0, 0, 0, 1, 0x65}, capturedAt: capturedAt}
	decoder := &fakeDecoder{png: testPNG(t, 2, 1), width: 2, height: 1}
	base, _ := url.Parse("https://jetkvm.invalid")
	provider := &fakeProvider{session: session}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, provider, WithDecoder(decoder))
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{MaxWidth: 800, MaxHeight: 600})
	if err != nil {
		t.Fatal(err)
	}
	if provider.profile != SessionProfileVideo || decoder.maxWidth != 800 || decoder.maxHeight != 600 || result.Width != 2 || result.Height != 1 || result.MIMEType != "image/png" || !result.CapturedAt.Equal(capturedAt) {
		t.Fatalf("profile=%v decoder=%+v result=%+v", provider.profile, decoder, result)
	}
}

func TestManagerCaptureScreenRequiresDecoder(t *testing.T) {
	manager := testManager(t, &fakeSession{results: map[string]any{}})
	if _, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}); !errors.Is(err, ErrDecoderUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

func TestManagerCaptureScreenDistinguishesEstablishedNoSignal(t *testing.T) {
	session := &fakeSession{results: map[string]any{
		methodVideoState: map[string]any{"ready": false},
	}}
	base, _ := url.Parse("https://jetkvm.invalid")
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, &fakeProvider{session: session}, WithDecoder(&fakeDecoder{}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
	if !errors.Is(err, ErrNoSignal) {
		t.Fatalf("error = %v, want no-signal distinction", err)
	}
	if got := calledMethods(session.calls); len(got) != 1 || got[0] != methodVideoState {
		t.Fatalf("methods = %v, want state probe without capture", got)
	}
}

func TestCaptureScreenServerDeadlineExpiresNoFrameAndCleansWaiter(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	manager := newCaptureTestManager(t, &captureTestSession{receiver: receiver}, &fakeDecoder{})

	_, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want server deadline", err)
	}
	var classified interface{ ToolErrorCode() string }
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "timeout" {
		t.Fatalf("error = %#v, want timeout classification", err)
	}
	if receiver.Waiting() {
		t.Fatal("timed out capture left a video waiter")
	}
	if len(manager.operations) != 0 || len(manager.deviceOps["lab"]) != 0 || len(manager.sessions) != 0 || len(manager.captures) != 0 || len(manager.decoders) != 0 {
		t.Fatal("timed out capture leaked admission permits")
	}
}

func TestCaptureScreenAppliesServerOwnedDefaultDeadline(t *testing.T) {
	var remaining time.Duration
	provider := &captureTestProvider{setup: func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("capture context has no deadline")
		}
		remaining = time.Until(deadline)
		return context.Canceled
	}}
	manager := newCaptureTestManagerWithProvider(t, provider, &fakeDecoder{})

	_, err := manager.CaptureScreen(context.Background(), "lab", mcpserver.CaptureRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want probe cancellation", err)
	}
	if remaining <= 0 || remaining > defaultCaptureTimeout {
		t.Fatalf("default deadline remaining = %v, want within (0, %v]", remaining, defaultCaptureTimeout)
	}
}

func TestCaptureScreenServerDeadlineDuringSessionSetupUsesReadTimeout(t *testing.T) {
	provider := &captureTestProvider{setup: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	manager := newCaptureTestManagerWithProvider(t, provider, &fakeDecoder{})

	_, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want server deadline", err)
	}
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "timeout" || classified.ToolErrorOutcome() != ToolOutcomeFailed {
		t.Fatalf("error = %#v, want read timeout/failed", err)
	}
	if len(manager.operations) != 0 || len(manager.deviceOps["lab"]) != 0 || len(manager.sessions) != 0 || len(manager.captures) != 0 || len(manager.decoders) != 0 {
		t.Fatal("timed out session setup leaked admission permits")
	}
}

func TestCaptureScreenCallerCancellationDuringSessionSetupStaysNotSent(t *testing.T) {
	provider := &captureTestProvider{setup: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	manager := newCaptureTestManagerWithProvider(t, provider, &fakeDecoder{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want caller cancellation", err)
	}
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "canceled" || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
		t.Fatalf("error = %#v, want canceled/not_sent", err)
	}
}

func TestCaptureScreenCallerDeadlineDuringSessionSetupStaysNotSent(t *testing.T) {
	provider := &captureTestProvider{setup: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	manager := newCaptureTestManagerWithProvider(t, provider, &fakeDecoder{})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want caller deadline", err)
	}
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "timeout" || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
		t.Fatalf("error = %#v, want timeout/not_sent", err)
	}
}

func TestCaptureScreenRestoresCallerContextErasedDuringSessionSetup(t *testing.T) {
	tests := []struct {
		name     string
		context  func() (context.Context, context.CancelFunc)
		wantErr  error
		wantCode string
	}{
		{
			name: "cancellation",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr:  context.Canceled,
			wantCode: "canceled",
		},
		{
			name: "deadline",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			wantErr:  context.DeadlineExceeded,
			wantCode: "timeout",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &captureTestProvider{setup: func(ctx context.Context) error {
				if ctx.Err() == nil {
					t.Fatal("session setup did not receive completed caller context")
				}
				return fmt.Errorf("%w: device probe", ErrDeviceUnreachable)
			}}
			manager := newCaptureTestManagerWithProvider(t, provider, &fakeDecoder{})
			ctx, cancel := test.context()
			defer cancel()

			_, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, time.Second)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			var classified interface {
				ToolErrorCode() string
				ToolErrorOutcome() string
			}
			if !errors.As(err, &classified) || classified.ToolErrorCode() != test.wantCode || classified.ToolErrorOutcome() != ToolOutcomeNotSent {
				t.Fatalf("error = %#v, want %s/not_sent", err, test.wantCode)
			}
		})
	}
}

func TestCaptureScreenServerDeadlineWinsWhenCallerCancelsWhileSetupUnwinds(t *testing.T) {
	serverExpired := make(chan struct{})
	releaseSetup := make(chan struct{})
	provider := &captureTestProvider{setup: func(ctx context.Context) error {
		<-ctx.Done()
		close(serverExpired)
		<-releaseSetup
		return ctx.Err()
	}}
	manager := newCaptureTestManagerWithProvider(t, provider, &fakeDecoder{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, 10*time.Millisecond)
		errCh <- err
	}()

	<-serverExpired
	cancel()
	close(releaseSetup)
	err := <-errCh
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want server deadline", err)
	}
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "timeout" || classified.ToolErrorOutcome() != ToolOutcomeFailed {
		t.Fatalf("error = %#v, want server timeout/failed", err)
	}
}

func TestCaptureScreenEarlierCallerCancellationWins(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	manager := newCaptureTestManager(t, &captureTestSession{receiver: receiver}, &fakeDecoder{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, time.Second)
		errCh <- err
	}()
	waitForVideoWaiter(t, receiver)
	cancel()
	err := <-errCh
	if !errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want caller cancellation", err)
	}
	if receiver.Waiting() {
		t.Fatal("canceled capture left a video waiter")
	}
}

func TestCaptureScreenEarlierCallerDeadlineWins(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	manager := newCaptureTestManager(t, &captureTestSession{receiver: receiver}, &fakeDecoder{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want earlier caller deadline", err)
	}
	if receiver.Waiting() {
		t.Fatal("caller deadline left a video waiter")
	}
}

func TestCaptureScreenFrameAtDeadlineReturnsOneTerminalResultAndCleansWaiter(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	now := time.Now().UTC()
	receiver.Observe(rtpPacket(0, 99, true, []byte{0x61, 0}), now.Add(-time.Nanosecond))
	receiver.Observe(rtpPacket(1, 100, true, stapA(
		[]byte{0x67, 0x42, 0x00, 0x1f}, []byte{0x68, 0xce, 0x06, 0xe2},
	)), now)
	manager := newCaptureTestManager(t, &captureTestSession{receiver: receiver}, &fakeDecoder{png: testPNG(t, 1, 1), width: 1, height: 1})
	resultCh := make(chan struct {
		result mcpserver.CaptureResult
		err    error
	}, 1)
	const timeout = 25 * time.Millisecond
	go func() {
		result, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, timeout)
		resultCh <- struct {
			result mcpserver.CaptureResult
			err    error
		}{result, err}
	}()
	waitForVideoWaiter(t, receiver)
	time.AfterFunc(timeout, func() {
		receiver.Observe(rtpPacket(2, 101, true, []byte{0x65, 0x88, 0x84}), now.Add(time.Millisecond))
	})
	terminal := <-resultCh
	if terminal.err != nil && !errors.Is(terminal.err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline or successful frame", terminal.err)
	}
	if terminal.err == nil && (terminal.result.Width != 1 || terminal.result.Height != 1 || terminal.result.MIMEType != "image/png") {
		t.Fatalf("result = %+v, want valid capture", terminal.result)
	}
	if receiver.Waiting() {
		t.Fatal("frame/deadline race left a video waiter")
	}
}

func TestCaptureScreenDecodeUsesServerDeadline(t *testing.T) {
	decoder := &deadlineBlockingDecoder{done: make(chan struct{})}
	manager := newCaptureTestManager(t, &captureTestSession{
		h264:       []byte{0, 0, 0, 1, 0x65},
		capturedAt: time.Now().UTC(),
	}, decoder)

	_, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want server deadline", err)
	}
	if !decoder.hasDeadline.Load() {
		t.Fatal("decoder did not receive a deadline")
	}
	select {
	case <-decoder.done:
	default:
		t.Fatal("capture returned before its decoder stopped")
	}
}

func TestCaptureScreenRejectsSuccessfulDecodeAfterServerDeadline(t *testing.T) {
	decoder := &successfulAfterDeadlineDecoder{png: testPNG(t, 1, 1)}
	manager := newCaptureTestManager(t, &captureTestSession{
		h264:       []byte{0, 0, 0, 1, 0x65},
		capturedAt: time.Now().UTC(),
	}, decoder)

	result, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("result = %+v error = %v, want server deadline", result, err)
	}
	var classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	if !errors.As(err, &classified) || classified.ToolErrorCode() != "timeout" || classified.ToolErrorOutcome() != ToolOutcomeFailed {
		t.Fatalf("error = %#v, want server timeout/failed", err)
	}
}

func TestCaptureScreenContextWinsOverRawDecoderError(t *testing.T) {
	tests := []struct {
		name     string
		context  func(started <-chan struct{}) (context.Context, context.CancelFunc)
		timeout  time.Duration
		wantErr  error
		wantCode string
	}{
		{
			name: "server deadline",
			context: func(<-chan struct{}) (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			timeout: 10 * time.Millisecond, wantErr: context.DeadlineExceeded, wantCode: "timeout",
		},
		{
			name: "caller deadline",
			context: func(<-chan struct{}) (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Millisecond)
			},
			timeout: time.Second, wantErr: context.DeadlineExceeded, wantCode: "timeout",
		},
		{
			name: "caller cancellation",
			context: func(started <-chan struct{}) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					<-started
					cancel()
				}()
				return ctx, cancel
			},
			timeout: time.Second, wantErr: context.Canceled, wantCode: "canceled",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := &rawErrorAfterContextDecoder{started: make(chan struct{})}
			manager := newCaptureTestManager(t, &captureTestSession{
				h264:       []byte{0, 0, 0, 1, 0x65},
				capturedAt: time.Now().UTC(),
			}, decoder)
			ctx, cancel := test.context(decoder.started)
			defer cancel()

			result, err := manager.captureScreen(ctx, "lab", mcpserver.CaptureRequest{}, test.timeout)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("result = %+v error = %v, want %v", result, err, test.wantErr)
			}
			var classified interface {
				ToolErrorCode() string
				ToolErrorOutcome() string
			}
			if !errors.As(err, &classified) || classified.ToolErrorCode() != test.wantCode || classified.ToolErrorOutcome() != ToolOutcomeFailed {
				t.Fatalf("error = %#v, want %s/failed", err, test.wantCode)
			}
		})
	}
}

func TestCaptureScreenCanProceedAfterTimedOutCapture(t *testing.T) {
	receiver := newVideoReceiver()
	defer receiver.Close()
	now := time.Now().UTC()
	receiver.Observe(rtpPacket(0, 99, true, []byte{0x61, 0}), now.Add(-time.Nanosecond))
	receiver.Observe(rtpPacket(1, 100, true, stapA(
		[]byte{0x67, 0x42, 0x00, 0x1f}, []byte{0x68, 0xce, 0x06, 0xe2},
	)), now)
	manager := newCaptureTestManager(t, &captureTestSession{receiver: receiver}, &fakeDecoder{png: testPNG(t, 1, 1), width: 1, height: 1})

	if _, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, 10*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first capture error = %v, want deadline", err)
	}
	if receiver.Waiting() {
		t.Fatal("timed out capture left a video waiter")
	}
	resultCh := make(chan struct {
		result mcpserver.CaptureResult
		err    error
	}, 1)
	go func() {
		result, err := manager.captureScreen(context.Background(), "lab", mcpserver.CaptureRequest{}, time.Second)
		resultCh <- struct {
			result mcpserver.CaptureResult
			err    error
		}{result, err}
	}()
	waitForVideoWaiter(t, receiver)
	receiver.Observe(rtpPacket(2, 101, true, []byte{0x65, 0x88, 0x84}), now.Add(time.Millisecond))
	terminal := <-resultCh
	if terminal.err != nil {
		t.Fatal(terminal.err)
	}
	if terminal.result.Width != 1 || terminal.result.Height != 1 {
		t.Fatalf("result = %+v, want successful later capture", terminal.result)
	}
	if len(manager.operations) != 0 || len(manager.deviceOps["lab"]) != 0 || len(manager.sessions) != 0 || len(manager.captures) != 0 || len(manager.decoders) != 0 {
		t.Fatal("capture sequence leaked admission permits")
	}
}

type captureTestSession struct {
	receiver   *videoReceiver
	h264       []byte
	capturedAt time.Time
}

func (session *captureTestSession) Call(_ context.Context, method string, _ any, result any) error {
	if method != methodVideoState {
		return errors.New("unexpected method")
	}
	return json.Unmarshal([]byte(`{"ready":true}`), result)
}

func (session *captureTestSession) Upload(context.Context, string, io.Reader, int64) error {
	return errors.New("unexpected upload")
}

func (session *captureTestSession) CaptureH264(ctx context.Context) ([]byte, time.Time, error) {
	if session.receiver == nil {
		return append([]byte(nil), session.h264...), session.capturedAt, nil
	}
	return session.receiver.Capture(ctx)
}

type deadlineBlockingDecoder struct {
	done        chan struct{}
	hasDeadline atomic.Bool
}

type successfulAfterDeadlineDecoder struct {
	png []byte
}

type rawErrorAfterContextDecoder struct {
	started chan struct{}
}

func (decoder *rawErrorAfterContextDecoder) Decode(ctx context.Context, _ []byte, _, _ int) ([]byte, int, int, error) {
	close(decoder.started)
	<-ctx.Done()
	return nil, 0, 0, ErrInvalidResponse
}

func (decoder *successfulAfterDeadlineDecoder) Decode(ctx context.Context, _ []byte, _, _ int) ([]byte, int, int, error) {
	<-ctx.Done()
	return append([]byte(nil), decoder.png...), 1, 1, nil
}

func (decoder *deadlineBlockingDecoder) Decode(ctx context.Context, _ []byte, _, _ int) ([]byte, int, int, error) {
	_, decoderDeadline := ctx.Deadline()
	decoder.hasDeadline.Store(decoderDeadline)
	<-ctx.Done()
	close(decoder.done)
	return nil, 0, 0, ctx.Err()
}

type captureTestProvider struct {
	session Session
	setup   func(context.Context) error
}

func (provider *captureTestProvider) WithSession(ctx context.Context, _ DeviceConfig, _ SessionProfile, operation func(Session) error) error {
	if provider.setup != nil {
		if err := provider.setup(ctx); err != nil {
			return err
		}
	}
	return operation(provider.session)
}

func newCaptureTestManager(t *testing.T, session Session, decoder Decoder) *Manager {
	return newCaptureTestManagerWithProvider(t, &captureTestProvider{session: session}, decoder)
}

func newCaptureTestManagerWithProvider(t *testing.T, provider SessionProvider, decoder Decoder) *Manager {
	t.Helper()
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, provider, WithDecoder(decoder))
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func waitForVideoWaiter(t *testing.T, receiver *videoReceiver) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !receiver.Waiting() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !receiver.Waiting() {
		t.Fatal("capture waiter did not register")
	}
}

type fakeDecoder struct {
	png                 []byte
	width, height       int
	maxWidth, maxHeight int
}

func (decoder *fakeDecoder) Decode(_ context.Context, _ []byte, maxWidth, maxHeight int) ([]byte, int, int, error) {
	decoder.maxWidth, decoder.maxHeight = maxWidth, maxHeight
	return append([]byte(nil), decoder.png...), decoder.width, decoder.height, nil
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	pixels := image.NewRGBA(image.Rect(0, 0, width, height))
	pixels.Set(0, 0, color.RGBA{R: 0x22, G: 0x44, B: 0x66, A: 0xff})
	var output bytes.Buffer
	if err := png.Encode(&output, pixels); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
