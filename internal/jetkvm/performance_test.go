package jetkvm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

var performancePNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")

type performanceSession struct {
	callFn    func(context.Context, string) error
	captureFn func(context.Context) ([]byte, time.Time, error)
}

func (session *performanceSession) Call(ctx context.Context, method string, _ any, result any) error {
	if session.callFn != nil {
		if err := session.callFn(ctx, method); err != nil {
			return err
		}
	}
	values := map[string]any{
		methodPing: "pong", methodLocalVersion: map[string]any{"appVersion": "fixture", "systemVersion": "fixture"},
		methodActiveExtension: "atx-power", methodVirtualMediaState: nil,
		methodVideoState: map[string]any{"ready": true, "streaming": 1, "width": 1, "height": 1, "fps": 1},
		methodUSBState:   "configured", methodATXState: map[string]any{"power": true, "hdd": false},
	}
	if result == nil {
		return nil
	}
	raw, err := json.Marshal(values[method])
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func (*performanceSession) Upload(context.Context, string, io.Reader, int64) error {
	return errors.New("fixture upload forbidden")
}

func (session *performanceSession) CaptureH264(ctx context.Context) ([]byte, time.Time, error) {
	if session.captureFn != nil {
		return session.captureFn(ctx)
	}
	return []byte{0, 0, 0, 1, 0x65}, time.Unix(1, 0).UTC(), nil
}

type performanceProvider struct {
	session *performanceSession
	setup   func(context.Context) error
	active  atomic.Int64
	calls   atomic.Int64
}

func (provider *performanceProvider) WithSession(ctx context.Context, _ DeviceConfig, _ SessionProfile, operation func(Session) error) error {
	provider.calls.Add(1)
	provider.active.Add(1)
	defer provider.active.Add(-1)
	if provider.setup != nil {
		if err := provider.setup(ctx); err != nil {
			return err
		}
	}
	return operation(provider.session)
}

type performanceDecoder struct {
	decode func(context.Context) error
	active atomic.Int64
	calls  atomic.Int64
}

func (decoder *performanceDecoder) Decode(ctx context.Context, _ []byte, _, _ int) ([]byte, int, int, error) {
	decoder.calls.Add(1)
	decoder.active.Add(1)
	defer decoder.active.Add(-1)
	if decoder.decode != nil {
		if err := decoder.decode(ctx); err != nil {
			return nil, 0, 0, err
		}
	}
	return append([]byte(nil), performancePNG...), 1, 1, nil
}

func performanceManager(tb testing.TB, provider SessionProvider, decoder Decoder) *Manager {
	tb.Helper()
	base, err := url.Parse("https://fixture.invalid")
	if err != nil {
		tb.Fatal(err)
	}
	options := []ManagerOption{WithLimits(Limits{MaxOperations: 2, MaxOperationsPerDevice: 2, MaxSessions: 2, MaxCaptures: 1, MaxDecoders: 1})}
	if decoder != nil {
		options = append(options, WithDecoder(decoder))
	}
	manager, err := NewManager([]DeviceConfig{{Name: "fixture", BaseURL: *base}}, provider, options...)
	if err != nil {
		tb.Fatal(err)
	}
	return manager
}

func assertPerformanceCleanup(tb testing.TB, manager *Manager, provider *performanceProvider, decoder *performanceDecoder) {
	tb.Helper()
	if provider != nil && provider.active.Load() != 0 {
		tb.Fatalf("active provider sessions=%d", provider.active.Load())
	}
	if decoder != nil && decoder.active.Load() != 0 {
		tb.Fatalf("active decoders=%d", decoder.active.Load())
	}
	if len(manager.operations) != 0 || len(manager.deviceOps["fixture"]) != 0 || len(manager.sessions) != 0 || len(manager.captures) != 0 || len(manager.decoders) != 0 || len(manager.mutations["fixture"]) != 0 {
		tb.Fatal("fixture operation retained admission permits")
	}
}

func TestManagerResourceSnapshotUsesAdmissionOccupancy(t *testing.T) {
	manager := performanceManager(t, &performanceProvider{session: &performanceSession{}}, new(performanceDecoder))
	manager.sessions <- struct{}{}
	manager.decoders <- struct{}{}
	snapshot := manager.ResourceSnapshot()
	if snapshot.ActiveSessions != 1 || snapshot.ActiveDecoders != 1 {
		t.Fatalf("resources=%+v", snapshot)
	}
	release(manager.sessions)
	release(manager.decoders)
	if snapshot := manager.ResourceSnapshot(); snapshot.ActiveSessions != 0 || snapshot.ActiveDecoders != 0 {
		t.Fatalf("resources after release=%+v", snapshot)
	}
}

func BenchmarkReadOnlyDiscovery(b *testing.B) {
	provider := &performanceProvider{session: &performanceSession{}}
	manager := performanceManager(b, provider, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		result, err := manager.ListDevices(context.Background())
		if err != nil || len(result.Devices) != 1 {
			b.Fatalf("result=%+v err=%v", result, err)
		}
	}
	b.StopTimer()
	if provider.calls.Load() != 0 {
		b.Fatalf("discovery opened %d provider sessions", provider.calls.Load())
	}
	assertPerformanceCleanup(b, manager, provider, nil)
}

func BenchmarkReadOnlyStatusFakeProvider(b *testing.B) {
	provider := &performanceProvider{session: &performanceSession{}}
	manager := performanceManager(b, provider, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := manager.Status(context.Background(), "fixture"); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if provider.calls.Load() != int64(b.N) {
		b.Fatalf("provider calls=%d want=%d", provider.calls.Load(), b.N)
	}
	assertPerformanceCleanup(b, manager, provider, nil)
}

func BenchmarkReadOnlyCaptureFakeProvider(b *testing.B) {
	provider := &performanceProvider{session: &performanceSession{}}
	decoder := new(performanceDecoder)
	manager := performanceManager(b, provider, decoder)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := manager.CaptureScreen(context.Background(), "fixture", mcpserver.CaptureRequest{MaxWidth: 1, MaxHeight: 1}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if provider.calls.Load() != int64(b.N) || decoder.calls.Load() != int64(b.N) {
		b.Fatalf("provider=%d decoder=%d want=%d", provider.calls.Load(), decoder.calls.Load(), b.N)
	}
	assertPerformanceCleanup(b, manager, provider, decoder)
}

func TestReadOnlyFixtureSoak(t *testing.T) {
	provider := &performanceProvider{session: &performanceSession{}}
	decoder := new(performanceDecoder)
	manager := performanceManager(t, provider, decoder)
	for index := 0; index < 100; index++ {
		if _, err := manager.Status(context.Background(), "fixture"); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.CaptureScreen(context.Background(), "fixture", mcpserver.CaptureRequest{MaxWidth: 1, MaxHeight: 1}); err != nil {
			t.Fatal(err)
		}
	}
	assertPerformanceCleanup(t, manager, provider, decoder)

	for _, test := range []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "provider setup", run: soakCancelProviderSetup},
		{name: "RPC", run: soakCancelRPC},
		{name: "video wait", run: soakCancelVideo},
		{name: "decode", run: soakCancelDecode},
	} {
		t.Run(test.name, test.run)
	}
}

func soakCancelProviderSetup(t *testing.T) {
	entered := make(chan struct{})
	var once sync.Once
	provider := &performanceProvider{session: &performanceSession{}, setup: func(ctx context.Context) error {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return ctx.Err()
	}}
	manager := performanceManager(t, provider, nil)
	runCanceledOperation(t, entered, func(ctx context.Context) error { _, err := manager.Status(ctx, "fixture"); return err })
	assertPerformanceCleanup(t, manager, provider, nil)
}

func soakCancelRPC(t *testing.T) {
	entered := make(chan struct{})
	var once sync.Once
	provider := &performanceProvider{session: &performanceSession{callFn: func(ctx context.Context, _ string) error {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return ctx.Err()
	}}}
	manager := performanceManager(t, provider, nil)
	runCanceledOperation(t, entered, func(ctx context.Context) error { _, err := manager.Status(ctx, "fixture"); return err })
	assertPerformanceCleanup(t, manager, provider, nil)
}

func soakCancelVideo(t *testing.T) {
	entered := make(chan struct{})
	var once sync.Once
	provider := &performanceProvider{session: &performanceSession{captureFn: func(ctx context.Context) ([]byte, time.Time, error) {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return nil, time.Time{}, ctx.Err()
	}}}
	decoder := new(performanceDecoder)
	manager := performanceManager(t, provider, decoder)
	runCanceledOperation(t, entered, func(ctx context.Context) error {
		_, err := manager.CaptureScreen(ctx, "fixture", mcpserver.CaptureRequest{MaxWidth: 1, MaxHeight: 1})
		return err
	})
	assertPerformanceCleanup(t, manager, provider, decoder)
}

func soakCancelDecode(t *testing.T) {
	entered := make(chan struct{})
	var once sync.Once
	provider := &performanceProvider{session: &performanceSession{}}
	decoder := &performanceDecoder{decode: func(ctx context.Context) error {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return ctx.Err()
	}}
	manager := performanceManager(t, provider, decoder)
	runCanceledOperation(t, entered, func(ctx context.Context) error {
		_, err := manager.CaptureScreen(ctx, "fixture", mcpserver.CaptureRequest{MaxWidth: 1, MaxHeight: 1})
		return err
	})
	assertPerformanceCleanup(t, manager, provider, decoder)
}

func runCanceledOperation(t *testing.T, entered <-chan struct{}, operation func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- operation(ctx) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("fixture operation did not reach cancellation stage")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v want cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("fixture operation did not stop after cancellation")
	}
}
