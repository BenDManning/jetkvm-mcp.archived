package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/BenDManning/jetkvm-mcp/internal/jetkvm"
)

func TestRunFixtureProducesSanitizedAggregateOnly(t *testing.T) {
	const privateSentinel = "PRIVATE-device-url-password-path-token-firmware-image"
	var output bytes.Buffer
	err := run(context.Background(), []string{"--mode", "fixture", "--iterations", "5"}, &output, func(string) (string, bool) {
		return privateSentinel, true
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), privateSentinel) || strings.Contains(output.String(), "fixture.invalid") || strings.Contains(output.String(), "ping") {
		t.Fatalf("report retained prohibited details: %s", output.String())
	}
	var report aggregateReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 1 || report.Mode != "fixture" || report.IterationsRequested != 5 || report.IterationsCompleted != 5 {
		t.Fatalf("report=%+v", report)
	}
	if report.StageSamplesDropped != 0 {
		t.Fatalf("stage samples dropped=%d", report.StageSamplesDropped)
	}
	for _, name := range []string{"discovery", "status", "capture"} {
		operation := report.Operations[name]
		if operation.Attempts != 5 || operation.OK != 5 || operation.Failed != 0 || operation.Canceled != 0 || operation.Deadline != 0 {
			t.Fatalf("%s=%+v", name, operation)
		}
	}
	if report.Cleanup.ActiveSessions != 0 || report.Cleanup.ActiveRPCRequests != 0 || report.Cleanup.ActiveVideoWaiters != 0 || report.Cleanup.ActiveDecoders != 0 || report.Cleanup.ChildProcessDelta != 0 {
		t.Fatalf("cleanup=%+v", report.Cleanup)
	}
	if report.Decision.SessionArchitecture != "fixture_only" || report.Decision.DecoderArchitecture != "fixture_only" {
		t.Fatalf("decision=%+v", report.Decision)
	}
}

func TestCleanupReportUsesMeasuredResourceCounters(t *testing.T) {
	report := buildCleanupReport(
		jetkvm.ResourceSnapshot{ActiveSessions: 1, ActiveDecoders: 4},
		jetkvm.StageSnapshot{Active: map[jetkvm.Stage]int{jetkvm.StageRPC: 2, jetkvm.StageVideoWait: 3}},
		5, 6, 7, 8,
	)
	if report.ActiveSessions != 1 || report.ActiveRPCRequests != 2 || report.ActiveVideoWaiters != 3 || report.ActiveDecoders != 4 || report.GoroutineDelta != 5 || report.HeapLiveDeltaBytes != 6 || report.HeapObjectsDelta != 7 || report.ChildProcessDelta != 8 {
		t.Fatalf("cleanup=%+v", report)
	}
}

func TestParseOptionsFailsClosedAroundHardwareAndBounds(t *testing.T) {
	for _, args := range [][]string{
		{"--mode", "unknown"},
		{"--iterations", "0"},
		{"--iterations", "10001"},
		{"--unknown"},
		{"extra"},
		{"--mode", "fixture", "--config", "private.yaml"},
		{"--mode", "hardware", "--config", "private.yaml", "--device", "lab"},
		{"--mode", "hardware", "--device", "lab", "--acknowledge-read-only-hardware"},
	} {
		if _, err := parseOptions(args); err == nil {
			t.Fatalf("args %q unexpectedly accepted", args)
		}
	}
	options, err := parseOptions([]string{"--mode", "hardware", "--config", "private.yaml", "--device", "lab", "--acknowledge-read-only-hardware", "--iterations", "100"})
	if err != nil {
		t.Fatal(err)
	}
	if options.mode != "hardware" || !options.ackHardware || options.device != "lab" || options.iterations != 100 {
		t.Fatalf("options=%+v", options)
	}
}

func TestLatencyUsesDeterministicNearestRanks(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []int64
		p50    int64
		p95    int64
	}{
		{name: "two", values: []int64{10, 20}, p50: 10, p95: 20},
		{name: "five", values: []int64{100, 20, 50, 10, 80}, p50: 50, p95: 100},
		{name: "one_hundred", values: integerRange(1, 100), p50: 50, p95: 95},
		{name: "one_hundred_two", values: integerRange(1, 102), p50: 51, p95: 97},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := latency(test.values)
			if report.P50 != test.p50 || report.P95 != test.p95 {
				t.Fatalf("latency=%+v, want p50=%d p95=%d", report, test.p50, test.p95)
			}
		})
	}
}

func integerRange(first, last int64) []int64 {
	values := make([]int64, 0, last-first+1)
	for value := first; value <= last; value++ {
		values = append(values, value)
	}
	return values
}
