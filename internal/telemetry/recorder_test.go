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
	recorder := New(&output, "1.2.3")
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
	if len(event) != 11 || event["schema"] != "jetkvm.operation.v3" || event["server_version"] != "1.2.3" || event["transport"] != "stdio" || event["operation"] != "status" || event["stage"] != "rpc" || event["code"] != "success" || event["outcome"] != "succeeded" {
		t.Fatalf("event = %#v", event)
	}
	eventTime, ok := event["time"].(string)
	parsedTime, err := time.Parse(time.RFC3339Nano, eventTime)
	if !ok || err != nil || eventTime[len(eventTime)-1:] != "Z" || time.Since(parsedTime) > time.Minute {
		t.Fatalf("time = %q, parse error = %v", eventTime, err)
	}
	processID, ok := event["process_instance_id"].(string)
	if !ok || !regexp.MustCompile(`^proc_[0-9a-f]{24}$`).MatchString(processID) {
		t.Fatalf("process_instance_id = %#v", event["process_instance_id"])
	}
	correlation, ok := event["correlation_id"].(string)
	if !ok || !regexp.MustCompile(`^op_[0-9a-f]{24}$`).MatchString(correlation) || CorrelationID(ctx) != correlation {
		t.Fatalf("correlation event=%q context=%q", correlation, CorrelationID(ctx))
	}
	duration, ok := event["duration_ms"].(float64)
	if !ok || duration < 0 || duration > 60_000 {
		t.Fatalf("duration_ms = %#v", event["duration_ms"])
	}
	var summary shutdownEvent
	if err := decoder.Decode(&summary); err != nil || summary.Operation != OperationLifecycle || summary.Stage != StageShutdown || summary.Code != "telemetry_summary" || summary.DroppedEvents != 0 || summary.WriterFailed {
		t.Fatalf("shutdown summary = %#v, error = %v", summary, err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		t.Fatalf("recorder emitted data after shutdown summary: %v", err)
	}
}

func TestOperationReferencesAppearOnlyAfterBinding(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output, "test")
	ctx, operation := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	operation.Record(StageAdmission, CodeSuccess, OutcomeSucceeded)
	BindDevice(ctx, DeviceRef("dev_00112233445566778899aabb"))
	operation.Record(StageConnect, CodeSuccess, OutcomeSucceeded)
	BindSession(ctx, SessionRef("ses_aabbccddeeff001122334455"))
	operation.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(&output)
	var before, deviceBound, sessionBound map[string]any
	for _, target := range []*map[string]any{&before, &deviceBound, &sessionBound} {
		if err := decoder.Decode(target); err != nil {
			t.Fatal(err)
		}
	}
	if _, exists := before["device_ref"]; exists {
		t.Fatalf("pre-resolution event has device_ref: %#v", before)
	}
	if _, exists := before["session_ref"]; exists {
		t.Fatalf("pre-generation event has session_ref: %#v", before)
	}
	if deviceBound["device_ref"] != "dev_00112233445566778899aabb" {
		t.Fatalf("device-bound event = %#v", deviceBound)
	}
	if _, exists := deviceBound["session_ref"]; exists {
		t.Fatalf("pre-generation event has session_ref: %#v", deviceBound)
	}
	if sessionBound["device_ref"] != "dev_00112233445566778899aabb" || sessionBound["session_ref"] != "ses_aabbccddeeff001122334455" {
		t.Fatalf("session-bound event = %#v", sessionBound)
	}
}

func TestRecorderEmitsClosedSessionSchema(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output, "test")
	deviceRef := DeviceRef("dev_00112233445566778899aabb")
	sessionRef := SessionRef("ses_aabbccddeeff001122334455")
	recorder.RecordSession(deviceRef, "", SessionConnectionAttemptCompleted, "device_unavailable", OutcomeFailed, 70*time.Second)
	recorder.RecordSession(deviceRef, sessionRef, SessionGenerationActive, CodeSuccess, OutcomeSucceeded, -time.Second)
	recorder.RecordSession(deviceRef, sessionRef, SessionEvent("not_closed"), CodeSuccess, OutcomeSucceeded, 0)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(&output)
	var failedAttempt, active map[string]any
	if err := decoder.Decode(&failedAttempt); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&active); err != nil {
		t.Fatal(err)
	}
	if len(failedAttempt) != 9 || failedAttempt["schema"] != "jetkvm.session.v1" || failedAttempt["event"] != "connection_attempt_completed" || failedAttempt["duration_ms"] != float64(60_000) || failedAttempt["outcome"] != OutcomeFailed {
		t.Fatalf("failed attempt = %#v", failedAttempt)
	}
	if _, exists := failedAttempt["session_ref"]; exists {
		t.Fatalf("failed attempt invented session_ref: %#v", failedAttempt)
	}
	if len(active) != 10 || active["event"] != "generation_active" || active["session_ref"] != string(sessionRef) || active["duration_ms"] != float64(0) {
		t.Fatalf("active event = %#v", active)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		t.Fatalf("invalid session event was retained: %v", err)
	}
}

func TestOpaqueReferencesAreRandomAndScoped(t *testing.T) {
	firstDevice, err := NewDeviceRef()
	if err != nil {
		t.Fatal(err)
	}
	secondDevice, err := NewDeviceRef()
	if err != nil {
		t.Fatal(err)
	}
	firstSession, err := NewSessionRef()
	if err != nil {
		t.Fatal(err)
	}
	secondSession, err := NewSessionRef()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^dev_[0-9a-f]{24}$`).MatchString(string(firstDevice)) || firstDevice == secondDevice {
		t.Fatalf("device refs = %q, %q", firstDevice, secondDevice)
	}
	if !regexp.MustCompile(`^ses_[0-9a-f]{24}$`).MatchString(string(firstSession)) || firstSession == secondSession {
		t.Fatalf("session refs = %q, %q", firstSession, secondSession)
	}
}

func TestStageSpanUsesOperationCorrelationAndOwnDuration(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output, "test")
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

func TestRecordersShareRandomProcessIdentity(t *testing.T) {
	processIDs := make([]string, 2)
	for index := range processIDs {
		var output bytes.Buffer
		recorder := New(&output, "test")
		_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
		span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
		if err := recorder.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		var got event
		if err := json.NewDecoder(&output).Decode(&got); err != nil {
			t.Fatal(err)
		}
		processIDs[index] = got.ProcessInstanceID
	}
	if !regexp.MustCompile(`^proc_[0-9a-f]{24}$`).MatchString(processIDs[0]) || processIDs[0] != processIDs[1] {
		t.Fatalf("process instance IDs = %q, want one random process identity", processIDs)
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
	recorder := New(writer, "test")
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

func TestRecorderSessionPressureIsNonblockingAndCounted(t *testing.T) {
	writer := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	recorder := New(writer, "test")
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageAdmission, CodeSuccess, OutcomeSucceeded)
	<-writer.started
	deviceRef := DeviceRef("dev_00112233445566778899aabb")
	sessionRef := SessionRef("ses_aabbccddeeff001122334455")
	recorded := make(chan struct{})
	go func() {
		for range cap(recorder.events) + 1 {
			recorder.RecordSession(deviceRef, sessionRef, SessionGenerationReused, CodeSuccess, OutcomeSucceeded, 0)
		}
		close(recorded)
	}()
	select {
	case <-recorded:
	case <-time.After(25 * time.Millisecond):
		close(writer.release)
		<-recorded
		t.Fatal("session telemetry pressure blocked its producer")
	}
	close(writer.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(writer.output.Bytes()))
	var summary shutdownEvent
	for decoder.More() {
		if err := decoder.Decode(&summary); err != nil {
			t.Fatal(err)
		}
	}
	if summary.Code != "telemetry_summary" || summary.DroppedEvents == 0 || summary.Outcome != OutcomeFailed {
		t.Fatalf("shutdown summary = %#v", summary)
	}
}

func TestRecorderCloseIsBoundedByContext(t *testing.T) {
	writer := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	recorder := New(writer, "test")
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	<-writer.started
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := recorder.Close(ctx)
	close(writer.release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("Close took %v after its deadline", elapsed)
	}
}

func TestRecorderStagePressureRetainsTerminalToolEvent(t *testing.T) {
	writer := &blockingWriter{started: make(chan struct{}), release: make(chan struct{})}
	recorder := New(writer, "test")
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
	recorder := New(&output, "test")
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
	var summary shutdownEvent
	if err := json.NewDecoder(&output).Decode(&summary); err != nil || summary.Code != "telemetry_summary" || summary.DroppedEvents != 0 || summary.WriterFailed || strings.Contains(output.String(), sentinel) {
		t.Fatalf("invalid telemetry was emitted: %q", output.String())
	}
}

func TestRecorderConcurrentLinesAreAtomic(t *testing.T) {
	var output bytes.Buffer
	recorder := New(&output, "test")
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
		if len(event) != 11 {
			t.Fatalf("line %d has fields %#v", index, event)
		}
	}
	var summary shutdownEvent
	if err := decoder.Decode(&summary); err != nil || summary.Code != "telemetry_summary" {
		t.Fatalf("shutdown summary = %#v, error = %v", summary, err)
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
	recorder := New(io.Discard, "test")
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
	recorder := New(failingWriter{}, "test")
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatalf("Close propagated writer failure: %v", err)
	}
}

type recoveringWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	output  bytes.Buffer
}

func (writer *recoveringWriter) Write(data []byte) (int, error) {
	failed := false
	writer.once.Do(func() {
		close(writer.started)
		<-writer.release
		failed = true
	})
	if failed {
		return 0, errors.New("private transient writer failure")
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.output.Write(data)
}

type summaryFailingWriter struct {
	writes int
	output bytes.Buffer
}

func (writer *summaryFailingWriter) Write(data []byte) (int, error) {
	writer.writes++
	written, _ := writer.output.Write(data)
	if writer.writes == 2 {
		return written, errors.New("private summary write failure")
	}
	return written, nil
}

func TestRecorderCorrectsFailureReportedBySummaryWrite(t *testing.T) {
	writer := new(summaryFailingWriter)
	recorder := New(writer, "test")
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(writer.output.Bytes()))
	var summary shutdownEvent
	var failure event
	for {
		var got event
		if err := decoder.Decode(&got); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if got.Code == "telemetry_summary" {
			summary.event = got
		} else if got.Code == "telemetry_writer_failure" {
			failure = got
		}
	}
	if writer.writes != 3 || summary.Outcome != OutcomeSucceeded || summary.WriterFailed || failure.Outcome != OutcomeFailed || failure.CorrelationID == "" || failure.CorrelationID != summary.CorrelationID {
		t.Fatalf("writes=%d summary=%#v failure=%#v", writer.writes, summary, failure)
	}
}

func TestRecorderCloseReportsDroppedEventsAndWriterFailure(t *testing.T) {
	writer := &recoveringWriter{started: make(chan struct{}), release: make(chan struct{})}
	recorder := New(writer, "test")
	_, span := recorder.Start(context.Background(), TransportStdio, OperationStatus)
	span.Record(StageRPC, CodeSuccess, OutcomeSucceeded)
	<-writer.started
	for index := 0; index < cap(recorder.events)+cap(recorder.terminal)+2; index++ {
		span.Record(StageTool, CodeSuccess, OutcomeSucceeded)
	}
	close(writer.release)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(writer.output.Bytes()))
	for {
		var got struct {
			Operation     string `json:"operation"`
			Stage         string `json:"stage"`
			Code          string `json:"code"`
			Outcome       string `json:"outcome"`
			DroppedEvents uint64 `json:"dropped_events"`
			WriterFailed  bool   `json:"writer_failed"`
		}
		if err := decoder.Decode(&got); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if got.Operation == OperationLifecycle && got.Stage == StageShutdown && got.Code == "telemetry_summary" {
			if got.Outcome != OutcomeFailed || got.DroppedEvents == 0 || !got.WriterFailed {
				t.Fatalf("shutdown summary = %#v", got)
			}
			return
		}
	}
	t.Fatalf("shutdown summary missing: %s", writer.output.String())
}
