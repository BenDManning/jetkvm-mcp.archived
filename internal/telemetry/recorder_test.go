package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecorderEmitsBoundedSchemaAndCorrelation(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output)
	ctx, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	time.Sleep(time.Millisecond)
	span.Record(StageRPC, CodeSuccess, OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	var event map[string]any
	decoder := json.NewDecoder(&output)
	if err := decoder.Decode(&event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if len(event) != 8 || event["schema"] != "jetkvm.operation.v1" || event["transport"] != "stdio" || event["operation"] != "status" || event["stage"] != "rpc" || event["code"] != "success" || event["outcome"] != "succeeded" {
		t.Fatalf("event = %#v", event)
	}
	correlation, ok := event["correlation_id"].(string)
	if !ok || !regexp.MustCompile(`^op_[0-9a-f]{24}$`).MatchString(correlation) || CorrelationID(ctx) != correlation {
		t.Fatalf("correlation event=%q context=%q", correlation, CorrelationID(ctx))
	}
	duration, ok := event["duration_ms"].(float64)
	if !ok || duration < 0 || duration > 60_000 {
		t.Fatalf("duration_ms = %#v", event["duration_ms"])
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		t.Fatalf("recorder emitted trailing data: %v", err)
	}
}

func TestStageSpanUsesOperationCorrelationAndOwnDuration(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output)
	ctx, operation := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	stage := BeginStage(ctx, StageRPC)
	stage.Record(CodeSuccess, OutcomeSucceeded)
	operation.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	var first, second event
	if err := decoder.Decode(&first); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&second); err != nil {
		t.Fatal(err)
	}
	if first.CorrelationID == "" || first.CorrelationID != second.CorrelationID || first.Stage != StageRPC || second.Stage != StageTool || first.DurationMS < 0 || second.DurationMS < 0 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

type blockingWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	output  bytes.Buffer
}

func (writer *blockingWriter) Write(data []byte) (int, error) {
	writer.once.Do(func() { close(writer.started) })
	<-writer.release
	return writer.output.Write(data)
}

func TestRecorderSlowWriterDoesNotDelayOperation(t *testing.T) {
	writer := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	recorder := New(writer)
	_, span := recorder.Start(context.Background(), TransportHTTP, OperationCapture)
	recorded := make(chan struct{})
	go func() {
		span.Record(StageCleanup, CodeSuccess, OutcomeSucceeded)
		close(recorded)
	}()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("telemetry writer was not invoked")
	}
	select {
	case <-recorded:
		close(writer.release)
	case <-time.After(25 * time.Millisecond):
		close(writer.release)
		<-recorded
		t.Fatal("slow telemetry writer delayed the operation")
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRecorderStagePressureRetainsTerminalToolEvent(t *testing.T) {
	writer := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	recorder := New(writer)
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageRPC, CodeSuccess, OutcomeSucceeded)
	<-writer.started
	for index := 0; index < cap(recorder.events)+1; index++ {
		span.Record(StageRPC, CodeSuccess, OutcomeSucceeded)
	}
	span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	close(writer.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(writer.output.Bytes()))
	terminal := 0
	for {
		var got event
		if err := decoder.Decode(&got); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if got.Stage == StageTool {
			terminal++
		}
	}
	if terminal != 1 {
		t.Fatalf("terminal events = %d, want 1", terminal)
	}
}

func TestRecorderDropsUnsafeOrInvalidFields(t *testing.T) {
	const sentinel = "PRIVATE\nINJECTED-device-url-token-path-firmware-rpc-child"
	var output bytes.Buffer
	recorder := New(&output)
	for _, test := range []struct {
		transport string
		operation string
		stage     string
		code      string
		outcome   string
	}{
		{sentinel, OperationStatus, StageTool, CodeSuccess, OutcomeSucceeded},
		{TransportStdio, sentinel, StageTool, CodeSuccess, OutcomeSucceeded},
		{TransportStdio, OperationStatus, sentinel, CodeSuccess, OutcomeSucceeded},
		{TransportStdio, OperationStatus, StageTool, sentinel, OutcomeSucceeded},
		{TransportStdio, OperationStatus, StageTool, CodeSuccess, sentinel},
	} {
		_, span := recorder.Start(context.Background(), test.transport, test.operation)
		span.Record(test.stage, test.code, test.outcome)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 || strings.Contains(output.String(), sentinel) {
		t.Fatalf("invalid telemetry was emitted: %q", output.String())
	}
}

func TestRecorderConcurrentLinesAreAtomic(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output)
	_, span := recorder.Start(context.Background(), TransportHTTP, OperationMedia)
	var workers sync.WaitGroup
	for index := 0; index < 100; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			span.Record(StageRPC, CodeSuccess, OutcomeSucceeded)
		}()
	}
	workers.Wait()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	for index := 0; index < 100; index++ {
		var event map[string]any
		if err := decoder.Decode(&event); err != nil {
			t.Fatalf("decode line %d: %v", index, err)
		}
		if len(event) != 8 {
			t.Fatalf("line %d has fields %#v", index, event)
		}
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		t.Fatalf("trailing telemetry: %v", err)
	}
}

func TestRecorderSyntheticLoadHasBoundedGoroutinesAndMemory(t *testing.T) {
	runtime.GC()
	beforeGoroutines := runtime.NumGoroutine()
	var beforeMemory runtime.MemStats
	runtime.ReadMemStats(&beforeMemory)
	recorder := New(io.Discard)
	ctx, _ := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	var workers sync.WaitGroup
	for index := 0; index < 1_000; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			BeginStage(ctx, StageRPC).Record(CodeSuccess, OutcomeSucceeded)
		}()
	}
	workers.Wait()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	afterGoroutines := runtime.NumGoroutine()
	var afterMemory runtime.MemStats
	runtime.ReadMemStats(&afterMemory)
	if afterGoroutines > beforeGoroutines+5 {
		t.Fatalf("goroutines grew from %d to %d", beforeGoroutines, afterGoroutines)
	}
	if afterMemory.HeapAlloc > beforeMemory.HeapAlloc+16<<20 {
		t.Fatalf("heap grew from %d to %d bytes", beforeMemory.HeapAlloc, afterMemory.HeapAlloc)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("private writer failure") }

func TestRecorderWriterFailureDoesNotChangeOperation(t *testing.T) {
	recorder := New(failingWriter{})
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close propagated writer failure: %v", err)
	}
}
