package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/url"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/config"
	"github.com/BenDManning/jetkvm-mcp/internal/jetkvm"
	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

const maxIterations = 10_000

var fixturePNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")

type benchmarkOptions struct {
	mode        string
	iterations  int
	configPath  string
	device      string
	ackHardware bool
}

type latencyReport struct {
	Min int64 `json:"min"`
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	Max int64 `json:"max"`
}

type operationReport struct {
	Attempts  int           `json:"attempts"`
	OK        int           `json:"ok"`
	Failed    int           `json:"failed"`
	Canceled  int           `json:"canceled"`
	Deadline  int           `json:"deadline"`
	LatencyUS latencyReport `json:"latency_us"`
}

type stageReport struct {
	Samples   int           `json:"samples"`
	OK        int           `json:"ok"`
	Failed    int           `json:"failed"`
	Canceled  int           `json:"canceled"`
	Deadline  int           `json:"deadline"`
	LatencyUS latencyReport `json:"latency_us"`
}

type cleanupReport struct {
	ActiveSessions     int   `json:"active_sessions"`
	ActiveRPCRequests  int   `json:"active_rpc_requests"`
	ActiveVideoWaiters int   `json:"active_video_waiters"`
	ActiveDecoders     int   `json:"active_decoders"`
	GoroutineDelta     int   `json:"goroutine_delta"`
	HeapLiveDeltaBytes int64 `json:"heap_live_delta_bytes"`
	HeapObjectsDelta   int64 `json:"heap_objects_delta"`
	ChildProcessDelta  int   `json:"child_process_delta"`
}

type decisionReport struct {
	SessionArchitecture string `json:"session_architecture"`
	DecoderArchitecture string `json:"decoder_architecture"`
}

type aggregateReport struct {
	SchemaVersion       int                        `json:"schema_version"`
	Mode                string                     `json:"mode"`
	IterationsRequested int                        `json:"iterations_requested"`
	IterationsCompleted int                        `json:"iterations_completed"`
	Operations          map[string]operationReport `json:"operations"`
	Stages              map[string]stageReport     `json:"stages"`
	StageSamplesDropped int64                      `json:"stage_samples_dropped"`
	Cleanup             cleanupReport              `json:"cleanup"`
	Decision            decisionReport             `json:"decision"`
}

type operationSamples struct {
	latency                        []int64
	ok, failed, canceled, deadline int
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.LookupEnv); err != nil {
		_, _ = io.WriteString(os.Stderr, "jetkvm-mcp-benchmark: benchmark failed\n")
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, lookup func(string) (string, bool)) error {
	options, err := parseOptions(args)
	if err != nil {
		return err
	}
	manager, err := newBenchmarkManager(options, lookup)
	if err != nil {
		return err
	}
	recorder := jetkvm.NewStageRecorder(min(options.iterations*16, 4096))
	ctx = jetkvm.WithStageRecorder(ctx, recorder)
	runtime.GC()
	beforeGoroutines := runtime.NumGoroutine()
	beforeChildren := jetkvm.ActiveHelperProcesses()
	var beforeMemory runtime.MemStats
	runtime.ReadMemStats(&beforeMemory)
	operations := map[string]*operationSamples{"discovery": {}, "status": {}, "capture": {}}
	completed := 0
	for index := 0; index < options.iterations; index++ {
		measure(ctx, operations["discovery"], func() error { _, err := manager.ListDevices(ctx); return err })
		measure(ctx, operations["status"], func() error { _, err := manager.Status(ctx, options.device); return err })
		measure(ctx, operations["capture"], func() error {
			_, err := manager.CaptureScreen(ctx, options.device, mcpserver.CaptureRequest{MaxWidth: 1920, MaxHeight: 1080})
			return err
		})
		completed++
		if ctx.Err() != nil {
			break
		}
	}
	runtime.GC()
	var afterMemory runtime.MemStats
	runtime.ReadMemStats(&afterMemory)
	modeDecision := "fixture_only"
	if options.mode == "hardware" {
		modeDecision = "inconclusive"
	}
	stageSnapshot := recorder.Snapshot()
	report := aggregateReport{
		SchemaVersion: 1, Mode: options.mode, IterationsRequested: options.iterations, IterationsCompleted: completed,
		Operations: make(map[string]operationReport, len(operations)), Stages: aggregateStages(stageSnapshot), StageSamplesDropped: stageSnapshot.Dropped,
		Cleanup: buildCleanupReport(
			manager.ResourceSnapshot(), stageSnapshot,
			runtime.NumGoroutine()-beforeGoroutines,
			signedDelta(afterMemory.HeapAlloc, beforeMemory.HeapAlloc),
			signedDelta(afterMemory.HeapObjects, beforeMemory.HeapObjects),
			jetkvm.ActiveHelperProcesses()-beforeChildren,
		),
		Decision: decisionReport{SessionArchitecture: modeDecision, DecoderArchitecture: modeDecision},
	}
	failed := false
	for name, samples := range operations {
		report.Operations[name] = samples.report()
		failed = failed || samples.failed != 0 || samples.canceled != 0 || samples.deadline != 0
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		return err
	}
	if failed {
		return errors.New("one or more read-only operations failed")
	}
	return nil
}

func buildCleanupReport(resources jetkvm.ResourceSnapshot, stages jetkvm.StageSnapshot, goroutineDelta int, heapLiveDeltaBytes, heapObjectsDelta int64, childProcessDelta int) cleanupReport {
	return cleanupReport{
		ActiveSessions: resources.ActiveSessions, ActiveRPCRequests: stages.Active[jetkvm.StageRPC],
		ActiveVideoWaiters: stages.Active[jetkvm.StageVideoWait], ActiveDecoders: resources.ActiveDecoders,
		GoroutineDelta: goroutineDelta, HeapLiveDeltaBytes: heapLiveDeltaBytes,
		HeapObjectsDelta: heapObjectsDelta, ChildProcessDelta: childProcessDelta,
	}
}

func parseOptions(args []string) (benchmarkOptions, error) {
	flags := flag.NewFlagSet("jetkvm-mcp-benchmark", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "fixture", "fixture or hardware")
	iterations := flags.Int("iterations", 100, "read-only iterations")
	configPath := flags.String("config", "", "hardware-mode configuration")
	device := flags.String("device", "", "designated non-production device")
	ack := flags.Bool("acknowledge-read-only-hardware", false, "acknowledge separately approved read-only hardware contact")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return benchmarkOptions{}, errors.New("invalid benchmark arguments")
	}
	if *iterations < 1 || *iterations > maxIterations || *mode != "fixture" && *mode != "hardware" {
		return benchmarkOptions{}, errors.New("mode or iteration bound is invalid")
	}
	if *mode == "fixture" {
		if *configPath != "" || *device != "" || *ack {
			return benchmarkOptions{}, errors.New("fixture mode does not accept hardware options")
		}
		return benchmarkOptions{mode: *mode, iterations: *iterations, device: "fixture"}, nil
	}
	if *configPath == "" || *device == "" || !*ack {
		return benchmarkOptions{}, errors.New("hardware mode requires config, device, and acknowledgement")
	}
	return benchmarkOptions{mode: *mode, iterations: *iterations, configPath: *configPath, device: *device, ackHardware: true}, nil
}

func newBenchmarkManager(options benchmarkOptions, lookup func(string) (string, bool)) (*jetkvm.Manager, error) {
	if options.mode == "fixture" {
		base, _ := url.Parse("https://fixture.invalid")
		return jetkvm.NewManager([]jetkvm.DeviceConfig{{Name: "fixture", BaseURL: *base}}, fixtureProvider{}, jetkvm.WithDecoder(fixtureDecoder{}))
	}
	loaded, err := config.Load(options.configPath, lookup)
	if err != nil {
		return nil, err
	}
	var selected *jetkvm.DeviceConfig
	for index := range loaded.Devices {
		if loaded.Devices[index].Name == options.device {
			candidate := loaded.Devices[index]
			selected = &candidate
			break
		}
	}
	if selected == nil {
		return nil, errors.New("designated device is not configured")
	}
	decoder, err := jetkvm.NewFFmpegDecoder()
	if err != nil {
		return nil, err
	}
	return jetkvm.NewManager([]jetkvm.DeviceConfig{*selected}, jetkvm.NewWebRTCProvider(jetkvm.WebRTCProviderOptions{}), jetkvm.WithDecoder(decoder), jetkvm.WithLimits(loaded.Limits))
}

func measure(ctx context.Context, samples *operationSamples, operation func() error) {
	started := time.Now()
	err := operation()
	samples.latency = append(samples.latency, max(time.Since(started).Microseconds(), 0))
	switch {
	case err == nil:
		samples.ok++
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		samples.canceled++
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		samples.deadline++
	default:
		samples.failed++
	}
}

func (samples *operationSamples) report() operationReport {
	return operationReport{Attempts: len(samples.latency), OK: samples.ok, Failed: samples.failed, Canceled: samples.canceled, Deadline: samples.deadline, LatencyUS: latency(samples.latency)}
}

func aggregateStages(snapshot jetkvm.StageSnapshot) map[string]stageReport {
	type values struct {
		latency                        []int64
		ok, failed, canceled, deadline int
	}
	grouped := make(map[string]*values)
	for _, sample := range snapshot.Samples {
		key := string(sample.Stage)
		value := grouped[key]
		if value == nil {
			value = new(values)
			grouped[key] = value
		}
		value.latency = append(value.latency, sample.DurationUS)
		switch sample.Outcome {
		case jetkvm.StageOutcomeOK:
			value.ok++
		case jetkvm.StageOutcomeCanceled:
			value.canceled++
		case jetkvm.StageOutcomeDeadline:
			value.deadline++
		default:
			value.failed++
		}
	}
	result := make(map[string]stageReport, len(grouped))
	for key, value := range grouped {
		result[key] = stageReport{Samples: len(value.latency), OK: value.ok, Failed: value.failed, Canceled: value.canceled, Deadline: value.deadline, LatencyUS: latency(value.latency)}
	}
	return result
}

func latency(values []int64) latencyReport {
	if len(values) == 0 {
		return latencyReport{}
	}
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return latencyReport{Min: ordered[0], P50: ordered[nearestRankIndex(len(ordered), 50)], P95: ordered[nearestRankIndex(len(ordered), 95)], Max: ordered[len(ordered)-1]}
}

func nearestRankIndex(length, percentile int) int {
	return (length*percentile+99)/100 - 1
}

func signedDelta(after, before uint64) int64 {
	if after >= before {
		return int64(after - before)
	}
	return -int64(before - after)
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

type fixtureProvider struct{}

func (fixtureProvider) WithSession(ctx context.Context, _ jetkvm.DeviceConfig, _ jetkvm.SessionProfile, operation func(jetkvm.Session) error) error {
	return operation(fixtureSession{})
}

type fixtureSession struct{}

func (fixtureSession) Call(_ context.Context, method string, _ any, result any) error {
	values := map[string]any{
		"ping": "pong", "getLocalVersion": map[string]any{"appVersion": "fixture", "systemVersion": "fixture"},
		"getActiveExtension": "atx-power", "getVirtualMediaState": nil,
		"getVideoState": map[string]any{"ready": true, "streaming": 1, "width": 1, "height": 1, "fps": 1},
		"getUSBState":   "configured", "getATXState": map[string]any{"power": true, "hdd": false},
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

func (fixtureSession) Upload(context.Context, string, io.Reader, int64) error {
	return errors.New("fixture upload forbidden")
}
func (fixtureSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return []byte{0, 0, 0, 1, 0x65}, time.Unix(1, 0).UTC(), nil
}

type fixtureDecoder struct{}

func (fixtureDecoder) Decode(context.Context, []byte, int, int) ([]byte, int, int, error) {
	return append([]byte(nil), fixturePNG...), 1, 1, nil
}
