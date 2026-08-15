package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestStageRecorderIsBoundedConcurrentAndSanitized(t *testing.T) {
	const privateSentinel = "PRIVATE-device-url-method-path-error-image-firmware"
	recorder := NewStageRecorder(64)
	ctx := WithStageRecorder(context.Background(), recorder)
	var workers sync.WaitGroup
	for index := 0; index < 1_000; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			span := startStage(ctx, StageRPC)
			if index%3 == 0 {
				span.Finish(context.Canceled)
			} else {
				span.Finish(nil)
			}
			span.Finish(errors.New(privateSentinel))
		}(index)
	}
	workers.Wait()
	snapshot := recorder.Snapshot()
	if len(snapshot.Samples) != 64 || snapshot.Dropped != 936 || snapshot.Active[StageRPC] != 0 {
		t.Fatalf("samples=%d dropped=%d active=%v", len(snapshot.Samples), snapshot.Dropped, snapshot.Active)
	}
	for _, sample := range snapshot.Samples {
		if sample.Stage != StageRPC || sample.DurationUS < 0 || sample.Outcome != StageOutcomeOK && sample.Outcome != StageOutcomeCanceled {
			t.Fatalf("sample=%+v", sample)
		}
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), privateSentinel) {
		t.Fatalf("snapshot retained private sentinel: %s", raw)
	}
}

func TestStageRecorderReportsOnlyCurrentlyActiveSpans(t *testing.T) {
	recorder := NewStageRecorder(1)
	span := startStage(WithStageRecorder(context.Background(), recorder), StageVideoWait)
	if active := recorder.Snapshot().Active[StageVideoWait]; active != 1 {
		t.Fatalf("active video waiters=%d", active)
	}
	span.Finish(nil)
	if active := recorder.Snapshot().Active[StageVideoWait]; active != 0 {
		t.Fatalf("active video waiters after finish=%d", active)
	}
}

func TestStageRecorderAbsentIsNoOpAndDeadlineIsStable(t *testing.T) {
	startStage(context.Background(), StageAuth).Finish(errors.New("PRIVATE-error"))
	recorder := NewStageRecorder(1)
	ctx := WithStageRecorder(context.Background(), recorder)
	span := startStage(ctx, StageAuth)
	span.Finish(context.DeadlineExceeded)
	snapshot := recorder.Snapshot()
	if len(snapshot.Samples) != 1 || snapshot.Samples[0].Outcome != StageOutcomeDeadline {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}
