package jetkvm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

type managedTelemetryEvent struct {
	Schema     string `json:"schema"`
	Operation  string `json:"operation"`
	Stage      string `json:"stage"`
	DeviceRef  string `json:"device_ref"`
	SessionRef string `json:"session_ref"`
	Event      string `json:"event"`
	Code       string `json:"code"`
	Outcome    string `json:"outcome"`
}

func TestManagedSessionTelemetryCorrelatesReuseReleaseAndShutdown(t *testing.T) {
	base, _ := url.Parse("https://PRIVATE-device.invalid/PRIVATE-path")
	connector := new(routingConnector)
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	manager, err := NewManager(
		[]DeviceConfig{
			{Name: "PRIVATE-lab", BaseURL: *base, Password: "PRIVATE-password"},
			{Name: "PRIVATE-rack", BaseURL: *base, Password: "PRIVATE-password"},
		},
		connector,
		WithTelemetry(recorder),
	)
	if err != nil {
		t.Fatal(err)
	}

	recordOperation := func(device string) {
		t.Helper()
		ctx, span := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
		_, operationErr := manager.DebugRPC(ctx, device, methodPing, nil, false)
		if operationErr != nil {
			t.Fatal(operationErr)
		}
		span.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	}
	recordOperation("PRIVATE-lab")
	if snapshot := manager.owners["PRIVATE-lab"].Snapshot(); snapshot.Ownership != ownerOwnershipActive || snapshot.Transition != ownerTransitionNone {
		t.Fatalf("generation became observable before active publication: %+v", snapshot)
	}
	recordOperation("PRIVATE-lab")
	recordOperation("PRIVATE-rack")

	ctx, releaseSpan := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationLifecycle)
	if _, err := manager.ReleaseSession(ctx, "PRIVATE-lab"); err != nil {
		t.Fatal(err)
	}
	releaseSpan.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	events := make([]managedTelemetryEvent, 0)
	decoder := json.NewDecoder(&output)
	for decoder.More() {
		var event managedTelemetryEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	refPattern := regexp.MustCompile(`^(dev|ses)_[0-9a-f]{24}$`)
	var labDeviceRef, labSessionRef, rackDeviceRef, rackSessionRef string
	toolEvents := 0
	lifecycle := make(map[string]int)
	for _, event := range events {
		if event.Schema == "jetkvm.operation.v3" && event.Stage == telemetry.StageTool && event.Operation == telemetry.OperationDebugRPC {
			toolEvents++
			if !refPattern.MatchString(event.DeviceRef) || !refPattern.MatchString(event.SessionRef) {
				t.Fatalf("operation refs = %#v", event)
			}
			if labDeviceRef == "" {
				labDeviceRef, labSessionRef = event.DeviceRef, event.SessionRef
			} else if event.DeviceRef == labDeviceRef && event.SessionRef != labSessionRef {
				t.Fatalf("reused operation changed session ref: %#v", event)
			} else if event.DeviceRef != labDeviceRef {
				rackDeviceRef, rackSessionRef = event.DeviceRef, event.SessionRef
			}
		}
		if event.Schema == "jetkvm.session.v1" {
			lifecycle[event.Event]++
		}
	}
	if toolEvents != 3 || labDeviceRef == "" || rackDeviceRef == "" || labDeviceRef == rackDeviceRef || labSessionRef == rackSessionRef {
		t.Fatalf("operation correlation lab=%q/%q rack=%q/%q events=%#v", labDeviceRef, labSessionRef, rackDeviceRef, rackSessionRef, events)
	}
	if lifecycle[string(telemetry.SessionConnectionAttemptCompleted)] != 2 ||
		lifecycle[string(telemetry.SessionGenerationActive)] != 2 ||
		lifecycle[string(telemetry.SessionGenerationReused)] != 1 ||
		lifecycle[string(telemetry.SessionExplicitlyReleased)] != 1 ||
		lifecycle[string(telemetry.SessionShutdownClosed)] != 1 {
		t.Fatalf("lifecycle counts = %#v", lifecycle)
	}
	if bytes.Contains(output.Bytes(), []byte("PRIVATE")) {
		t.Fatalf("managed telemetry retained a sensitive sentinel: %s", output.String())
	}
}

func decodeManagedTelemetry(t *testing.T, output []byte) []managedTelemetryEvent {
	t.Helper()
	var events []managedTelemetryEvent
	decoder := json.NewDecoder(bytes.NewReader(output))
	for decoder.More() {
		var event managedTelemetryEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func countSessionEvents(events []managedTelemetryEvent) map[string]int {
	counts := make(map[string]int)
	for _, event := range events {
		if event.Schema == "jetkvm.session.v1" {
			counts[event.Event]++
		}
	}
	return counts
}

func TestManagedSessionTelemetryFailedAttemptAndIdempotentReleaseHaveNoSessionRef(t *testing.T) {
	base, _ := url.Parse("https://PRIVATE-device.invalid")
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	manager, err := NewManager(
		[]DeviceConfig{{Name: "PRIVATE-lab", BaseURL: *base}},
		&deterministicConnector{err: ErrDeviceUnreachable},
		WithTelemetry(recorder),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	_, operationErr := manager.DebugRPC(ctx, "PRIVATE-lab", methodPing, nil, false)
	if operationErr == nil {
		t.Fatal("failed connector unexpectedly succeeded")
	}
	code, outcome := telemetryResult(operationErr)
	span.Record(telemetry.StageTool, code, outcome)
	unknownCtx, unknownSpan := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	_, unknownErr := manager.DebugRPC(unknownCtx, "PRIVATE-unknown", methodPing, nil, false)
	if unknownErr == nil {
		t.Fatal("unknown device unexpectedly resolved")
	}
	code, outcome = telemetryResult(unknownErr)
	unknownSpan.Record(telemetry.StageTool, code, outcome)
	if _, err := manager.ReleaseSession(ctx, "PRIVATE-lab"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ReleaseSession(ctx, "PRIVATE-lab"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	events := decodeManagedTelemetry(t, output.Bytes())
	counts := countSessionEvents(events)
	if counts[string(telemetry.SessionConnectionAttemptCompleted)] != 1 || counts[string(telemetry.SessionGenerationActive)] != 0 || counts[string(telemetry.SessionExplicitlyReleased)] != 0 {
		t.Fatalf("lifecycle counts = %#v", counts)
	}
	for _, event := range events {
		if event.Schema == "jetkvm.session.v1" && event.SessionRef != "" {
			t.Fatalf("failed/idempotent lifecycle invented session ref: %#v", event)
		}
	}
	var resolved, unresolved int
	for _, event := range events {
		if event.Schema != "jetkvm.operation.v3" || event.Stage != telemetry.StageTool || event.Operation != telemetry.OperationDebugRPC {
			continue
		}
		if event.DeviceRef == "" {
			unresolved++
		} else if event.SessionRef == "" {
			resolved++
		}
	}
	if resolved != 1 || unresolved != 1 {
		t.Fatalf("resolved-without-generation=%d unresolved=%d events=%#v", resolved, unresolved, events)
	}
}

func TestManagedSessionTelemetryFailedSetupCleanupUsesAttemptEvent(t *testing.T) {
	base, _ := url.Parse("https://PRIVATE-device.invalid")
	session := &residentSession{done: make(chan struct{}), closeErr: context.DeadlineExceeded}
	connector := &sessionAndErrorConnector{session: session, err: ErrDeviceUnreachable}
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	manager, err := NewManager([]DeviceConfig{{Name: "PRIVATE-lab", BaseURL: *base}}, connector, WithTelemetry(recorder))
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	_, operationErr := manager.DebugRPC(ctx, "PRIVATE-lab", methodPing, nil, false)
	if operationErr == nil {
		t.Fatal("failed setup cleanup unexpectedly succeeded")
	}
	code, outcome := telemetryResult(operationErr)
	span.Record(telemetry.StageTool, code, outcome)
	_ = manager.Close(context.Background())
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	counts := countSessionEvents(decodeManagedTelemetry(t, output.Bytes()))
	if counts[string(telemetry.SessionConnectionAttemptCompleted)] != 1 ||
		counts[string(telemetry.SessionCleanupTimeout)] != 0 ||
		counts[string(telemetry.SessionOwnershipUncertain)] != 0 {
		t.Fatalf("failed setup lifecycle counts = %#v", counts)
	}
	for _, event := range decodeManagedTelemetry(t, output.Bytes()) {
		if event.Event == string(telemetry.SessionConnectionAttemptCompleted) && (event.SessionRef != "" || event.Code != "ownership_uncertain" || event.Outcome != telemetry.OutcomeFailed) {
			t.Fatalf("failed setup attempt event = %#v", event)
		}
	}
}

func TestManagedSessionTelemetryDistinguishesTakeoverUncertaintyAndCleanupTimeout(t *testing.T) {
	for _, test := range []struct {
		name           string
		takenOver      bool
		closeErr       error
		wantOwnership  ownerOwnership
		wantEvent      telemetry.SessionEvent
		wantNoEvent    telemetry.SessionEvent
		triggerRelease bool
	}{
		{name: "recognized takeover", takenOver: true, wantOwnership: ownerOwnershipTakenOver, wantEvent: telemetry.SessionTakenOver, wantNoEvent: telemetry.SessionOwnershipUncertain},
		{name: "uncertain loss", wantOwnership: ownerOwnershipUncertain, wantEvent: telemetry.SessionOwnershipUncertain, wantNoEvent: telemetry.SessionTakenOver},
		{name: "cleanup timeout", closeErr: context.DeadlineExceeded, wantOwnership: ownerOwnershipUncertain, wantEvent: telemetry.SessionCleanupTimeout, triggerRelease: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			base, _ := url.Parse("https://PRIVATE-device.invalid")
			terminalSession := &deterministicConnectedSession{
				fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
				lost:        make(chan struct{}),
			}
			var session ConnectedSession = terminalSession
			if test.closeErr != nil {
				session = &residentSession{done: make(chan struct{}), closeErr: test.closeErr}
			}
			connector := &releaseConnector{session: session}
			var output bytes.Buffer
			recorder := telemetry.New(&output, "test")
			manager, err := NewManager([]DeviceConfig{{Name: "PRIVATE-lab", BaseURL: *base}}, connector, WithTelemetry(recorder))
			if err != nil {
				t.Fatal(err)
			}
			ctx, span := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
			if _, err := manager.DebugRPC(ctx, "PRIVATE-lab", methodPing, nil, false); err != nil {
				t.Fatal(err)
			}
			span.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
			if test.triggerRelease {
				_, _ = manager.ReleaseSession(context.Background(), "PRIVATE-lab")
			} else {
				terminalSession.takenOver = test.takenOver
				close(terminalSession.lost)
			}
			deadline := time.After(time.Second)
			for manager.owners["PRIVATE-lab"].Snapshot().Ownership != test.wantOwnership {
				select {
				case <-deadline:
					t.Fatalf("owner snapshot = %+v", manager.owners["PRIVATE-lab"].Snapshot())
				default:
					time.Sleep(time.Millisecond)
				}
			}
			_ = manager.Close(context.Background())
			if err := recorder.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
			counts := countSessionEvents(decodeManagedTelemetry(t, output.Bytes()))
			if counts[string(test.wantEvent)] == 0 {
				t.Fatalf("missing %q from %#v", test.wantEvent, counts)
			}
			if test.wantNoEvent != "" && counts[string(test.wantNoEvent)] != 0 {
				t.Fatalf("unexpected %q in %#v", test.wantNoEvent, counts)
			}
		})
	}
}

func TestManagedSessionTelemetryRecordsTakeoverRecognizedAtCleanupCompletion(t *testing.T) {
	base, _ := url.Parse("https://PRIVATE-device.invalid")
	session := &lateTakeoverSession{
		deterministicConnectedSession: &deterministicConnectedSession{
			fakeSession: &fakeSession{results: map[string]any{methodPing: "pong"}},
			lost:        make(chan struct{}),
			closeStart:  make(chan struct{}),
			closeDone:   make(chan struct{}),
		},
		takeover: make(chan struct{}),
	}
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	manager, err := NewManager([]DeviceConfig{{Name: "PRIVATE-lab", BaseURL: *base}}, &releaseConnector{session: session}, WithTelemetry(recorder))
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	if _, err := manager.DebugRPC(ctx, "PRIVATE-lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	span.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	close(session.lost)
	<-session.closeStart
	session.recognized.Store(true)
	close(session.closeDone)
	deadline := time.After(time.Second)
	for manager.owners["PRIVATE-lab"].Snapshot().Ownership != ownerOwnershipTakenOver {
		select {
		case <-deadline:
			t.Fatalf("late takeover snapshot = %+v", manager.owners["PRIVATE-lab"].Snapshot())
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	counts := countSessionEvents(decodeManagedTelemetry(t, output.Bytes()))
	if counts[string(telemetry.SessionOwnershipUncertain)] != 1 || counts[string(telemetry.SessionTakenOver)] != 1 {
		t.Fatalf("late takeover lifecycle counts = %#v", counts)
	}
}

func TestManagedSessionTelemetryUsesNewReferenceForEachGeneration(t *testing.T) {
	base, _ := url.Parse("https://PRIVATE-device.invalid")
	connector := &residentSequenceConnector{sessions: []ConnectedSession{
		&residentSession{done: make(chan struct{})},
		&residentSession{done: make(chan struct{})},
	}}
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	manager, err := NewManager([]DeviceConfig{{Name: "PRIVATE-lab", BaseURL: *base}}, connector, WithTelemetry(recorder))
	if err != nil {
		t.Fatal(err)
	}
	ctx, first := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	if _, err := manager.DebugRPC(ctx, "PRIVATE-lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	first.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	if _, err := manager.ReleaseSession(ctx, "PRIVATE-lab"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.TakeOverSession(ctx, "PRIVATE-lab"); err != nil {
		t.Fatal(err)
	}
	secondCtx, second := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	if _, err := manager.DebugRPC(secondCtx, "PRIVATE-lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	second.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	activeRefs := make(map[string]struct{})
	deviceRefs := make(map[string]struct{})
	reuse := 0
	for _, event := range decodeManagedTelemetry(t, output.Bytes()) {
		if event.Schema != "jetkvm.session.v1" {
			continue
		}
		deviceRefs[event.DeviceRef] = struct{}{}
		if event.Event == string(telemetry.SessionGenerationActive) {
			activeRefs[event.SessionRef] = struct{}{}
		}
		if event.Event == string(telemetry.SessionGenerationReused) {
			reuse++
		}
	}
	if len(deviceRefs) != 1 || len(activeRefs) != 2 || reuse != 1 {
		t.Fatalf("device refs=%#v active session refs=%#v reuse=%d", deviceRefs, activeRefs, reuse)
	}
}

func TestManagedSessionTelemetryRecordsIdleReleaseAfterCleanup(t *testing.T) {
	base, _ := url.Parse("https://PRIVATE-device.invalid")
	session := &residentSession{done: make(chan struct{})}
	timers := make(chan *manualOwnerTimer, 1)
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	manager, err := NewManager(
		[]DeviceConfig{{Name: "PRIVATE-lab", BaseURL: *base}},
		&residentConnector{session: session},
		WithTelemetry(recorder),
		withOwnerAfterFunc(func(_ time.Duration, fire func()) ownerTimer {
			timer := &manualOwnerTimer{fire: fire}
			timers <- timer
			return timer
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationDebugRPC)
	if _, err := manager.DebugRPC(ctx, "PRIVATE-lab", methodPing, nil, false); err != nil {
		t.Fatal(err)
	}
	span.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	(<-timers).fire()
	deadline := time.After(time.Second)
	for manager.owners["PRIVATE-lab"].Snapshot().Ownership != ownerOwnershipIdle {
		select {
		case <-deadline:
			t.Fatal("idle cleanup did not complete")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	counts := countSessionEvents(decodeManagedTelemetry(t, output.Bytes()))
	if counts[string(telemetry.SessionIdleReleased)] != 1 {
		t.Fatalf("lifecycle counts = %#v", counts)
	}
}

type telemetryCleanupSession struct{}

type telemetryDetachedCleanupSession struct{}

func (telemetryCleanupSession) Call(_ context.Context, method string, _ any, result any) error {
	if method == "listStorageFiles" {
		if storage, ok := result.(*firmwareStorageFiles); ok {
			*storage = firmwareStorageFiles{}
		}
	}
	return nil
}

func (telemetryCleanupSession) Upload(context.Context, string, io.Reader, int64) error {
	return nil
}

func (telemetryCleanupSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}

func (telemetryDetachedCleanupSession) Call(ctx context.Context, _ string, _ any, _ any) error {
	stage := telemetry.BeginStage(ctx, telemetry.StageCleanup)
	stage.Record(telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	return nil
}

func (telemetryDetachedCleanupSession) Upload(ctx context.Context, _ string, _ io.Reader, _ int64) error {
	stage := telemetry.BeginStage(ctx, telemetry.StageCleanup)
	stage.Record(telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	return nil
}

func (telemetryDetachedCleanupSession) CaptureH264(context.Context) ([]byte, time.Time, error) {
	return nil, time.Time{}, nil
}

func TestDetachedUploadCleanupPreservesOperationCorrelation(t *testing.T) {
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	operationCtx, operation := recorder.Start(context.Background(), telemetry.TransportStdio, telemetry.OperationMedia)
	canceledCtx, cancel := context.WithCancel(operationCtx)
	cancel()
	abortStartedUpload(canceledCtx, telemetryDetachedCleanupSession{}, "upload_12345678-1234-1234-1234-123456789abc", "fixture.iso")
	operation.Record(telemetry.StageTool, "canceled", telemetry.OutcomeUnknown)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	cleanupEvents := 0
	correlation := ""
	decoder := json.NewDecoder(&output)
	for decoder.More() {
		var event struct {
			CorrelationID string `json:"correlation_id"`
			Stage         string `json:"stage"`
		}
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		if event.Stage == telemetry.StageShutdown {
			continue
		}
		if correlation == "" {
			correlation = event.CorrelationID
		}
		if event.CorrelationID != correlation {
			t.Fatalf("correlation changed from %q to %q", correlation, event.CorrelationID)
		}
		if event.Stage == telemetry.StageCleanup {
			cleanupEvents++
		}
	}
	if cleanupEvents != 2 {
		t.Fatalf("cleanup events = %d, want upload abort and artifact deletion", cleanupEvents)
	}
}

func TestOperationTelemetryStageMatrix(t *testing.T) {
	fixture := newWebRTCConnectorFixture(t)
	manager, err := NewManager([]DeviceConfig{fixture.device}, fixture.connector)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	recorder := telemetry.New(&output, "test")
	baseCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ctx, operation := recorder.Start(baseCtx, telemetry.TransportStdio, telemetry.OperationStatus)
	err = manager.withSession(ctx, fixture.device, func(operationCtx context.Context, session Session) error {
		var pong string
		if err := session.Call(operationCtx, "ping", nil, &pong); err != nil {
			return err
		}
		if pong != "pong" {
			t.Fatalf("pong = %q", pong)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.captureScreen(ctx, "missing", mcpserver.CaptureRequest{}, time.Second); err == nil {
		t.Fatal("fixture capture unexpectedly succeeded")
	}
	if _, err := discardIncompleteUpload(ctx, telemetryCleanupSession{}, "PRIVATE-filename"); err != nil {
		t.Fatal(err)
	}
	operation.Record(telemetry.StageTool, telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	stages := make(map[string]int)
	correlation := ""
	decoder := json.NewDecoder(&output)
	for decoder.More() {
		var event struct {
			CorrelationID string `json:"correlation_id"`
			Stage         string `json:"stage"`
		}
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		if event.Stage == telemetry.StageShutdown {
			continue
		}
		if correlation == "" {
			correlation = event.CorrelationID
		}
		if event.CorrelationID != correlation {
			t.Fatalf("correlation changed from %q to %q", correlation, event.CorrelationID)
		}
		stages[event.Stage]++
	}
	for _, stage := range []string{
		telemetry.StageAdmission,
		telemetry.StageConnect,
		telemetry.StageAuth,
		telemetry.StageSignaling,
		telemetry.StageRPC,
		telemetry.StageCapture,
		telemetry.StageCleanup,
		telemetry.StageTool,
	} {
		if stages[stage] == 0 {
			t.Errorf("stage %q missing from %#v", stage, stages)
		}
	}
}
