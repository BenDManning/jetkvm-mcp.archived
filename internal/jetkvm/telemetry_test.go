package jetkvm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

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
	err = manager.withSession(ctx, fixture.device, func(session Session) error {
		var pong string
		if err := session.Call(ctx, "ping", nil, &pong); err != nil {
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
