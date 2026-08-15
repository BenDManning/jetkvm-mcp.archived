package jetkvm

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

func TestWebRTCLifecycleStagesShareBoundedRecorder(t *testing.T) {
	fixture := newWebRTCProviderFixture(t)
	recorder := NewStageRecorder(32)
	baseCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ctx := WithStageRecorder(baseCtx, recorder)
	err := fixture.provider.WithSession(ctx, fixture.device, SessionProfileData, func(session Session) error {
		var pong string
		if err := session.Call(ctx, "ping", nil, &pong); err != nil {
			return err
		}
		if pong != "pong" {
			t.Fatalf("pong=%q", pong)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := recorder.Snapshot()
	counts := make(map[Stage]int)
	for _, sample := range snapshot.Samples {
		counts[sample.Stage]++
		if sample.Outcome != StageOutcomeOK || sample.DurationUS < 0 {
			t.Fatalf("sample=%+v", sample)
		}
	}
	for _, stage := range []Stage{StageSessionSetup, StageAuth, StageSignaling, StageDataReady, StageRPC, StageSessionCleanup} {
		if counts[stage] != 1 {
			t.Errorf("stage %q count=%d snapshot=%+v", stage, counts[stage], snapshot)
		}
	}
}

func TestSessionCleanupStageClassifiesCanceledContext(t *testing.T) {
	recorder := NewStageRecorder(4)
	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := WithStageRecorder(baseCtx, recorder)
	cancel()
	session := &connectedSession{ctx: context.Background(), cancel: func() {}}
	session.CloseContext(ctx)
	snapshot := recorder.Snapshot()
	if len(snapshot.Samples) != 1 || snapshot.Samples[0].Stage != StageSessionCleanup || snapshot.Samples[0].Outcome != StageOutcomeCanceled {
		t.Fatalf("cleanup samples=%+v", snapshot.Samples)
	}
}

func TestStageRecorderDoesNotChangeManagerResultsOrClassifiedErrors(t *testing.T) {
	newFixture := func(callErr error) *Manager {
		session := &performanceSession{}
		if callErr != nil {
			session.callFn = func(context.Context, string) error { return callErr }
		}
		return performanceManager(t, &performanceProvider{session: session}, new(performanceDecoder))
	}
	plain := newFixture(nil)
	recorded := newFixture(nil)
	recordedCtx := WithStageRecorder(context.Background(), NewStageRecorder(64))
	plainStatus, plainErr := plain.Status(context.Background(), "fixture")
	recordedStatus, recordedErr := recorded.Status(recordedCtx, "fixture")
	if plainErr != nil || recordedErr != nil || !reflect.DeepEqual(plainStatus, recordedStatus) {
		t.Fatalf("status plain=%+v/%v recorded=%+v/%v", plainStatus, plainErr, recordedStatus, recordedErr)
	}
	plainCapture, plainErr := plain.CaptureScreen(context.Background(), "fixture", mcpserver.CaptureRequest{MaxWidth: 1, MaxHeight: 1})
	recordedCapture, recordedErr := recorded.CaptureScreen(recordedCtx, "fixture", mcpserver.CaptureRequest{MaxWidth: 1, MaxHeight: 1})
	if plainErr != nil || recordedErr != nil || !reflect.DeepEqual(plainCapture, recordedCapture) {
		t.Fatalf("capture plain=%+v/%v recorded=%+v/%v", plainCapture, plainErr, recordedCapture, recordedErr)
	}

	plainFailure := newFixture(context.Canceled)
	recordedFailure := newFixture(context.Canceled)
	_, plainErr = plainFailure.Status(context.Background(), "fixture")
	_, recordedErr = recordedFailure.Status(recordedCtx, "fixture")
	if !errors.Is(plainErr, context.Canceled) || !errors.Is(recordedErr, context.Canceled) {
		t.Fatalf("plain error=%v recorded error=%v", plainErr, recordedErr)
	}
	type classified interface {
		ToolErrorCode() string
		ToolErrorOutcome() string
	}
	var plainClass, recordedClass classified
	if !errors.As(plainErr, &plainClass) || !errors.As(recordedErr, &recordedClass) || plainClass.ToolErrorCode() != recordedClass.ToolErrorCode() || plainClass.ToolErrorOutcome() != recordedClass.ToolErrorOutcome() {
		t.Fatalf("classification plain=%v recorded=%v", plainErr, recordedErr)
	}
}
