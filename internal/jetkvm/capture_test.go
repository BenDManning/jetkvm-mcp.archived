package jetkvm

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
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
