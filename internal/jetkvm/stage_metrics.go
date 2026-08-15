package jetkvm

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Stage string

const (
	StageSessionSetup   Stage = "session_setup"
	StageAuth           Stage = "auth"
	StageSignaling      Stage = "signaling_setup"
	StageDataReady      Stage = "data_ready"
	StageRPC            Stage = "rpc"
	StageVideoWait      Stage = "video_wait"
	StageFFmpeg         Stage = "ffmpeg"
	StageSessionCleanup Stage = "session_cleanup"
)

type StageOutcome string

const (
	StageOutcomeOK       StageOutcome = "ok"
	StageOutcomeFailed   StageOutcome = "failed"
	StageOutcomeCanceled StageOutcome = "canceled"
	StageOutcomeDeadline StageOutcome = "deadline"
)

type StageSample struct {
	Stage      Stage        `json:"stage"`
	DurationUS int64        `json:"duration_us"`
	Outcome    StageOutcome `json:"outcome"`
}

type StageSnapshot struct {
	Samples []StageSample `json:"samples"`
	Dropped int64         `json:"dropped"`
	Active  map[Stage]int `json:"active"`
}

type StageRecorder struct {
	mu       sync.Mutex
	capacity int
	samples  []StageSample
	active   map[Stage]int
	dropped  atomic.Int64
}

type stageRecorderKey struct{}

type stageSpan struct {
	recorder *StageRecorder
	stage    Stage
	started  time.Time
	finished atomic.Bool
}

func NewStageRecorder(capacity int) *StageRecorder {
	if capacity < 1 || capacity > 4096 {
		capacity = 4096
	}
	return &StageRecorder{capacity: capacity, samples: make([]StageSample, 0, capacity), active: make(map[Stage]int)}
}

func WithStageRecorder(ctx context.Context, recorder *StageRecorder) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if recorder == nil {
		return ctx
	}
	return context.WithValue(ctx, stageRecorderKey{}, recorder)
}

func startStage(ctx context.Context, stage Stage) *stageSpan {
	if ctx == nil || !validStage(stage) {
		return &stageSpan{}
	}
	recorder, _ := ctx.Value(stageRecorderKey{}).(*StageRecorder)
	if recorder != nil {
		recorder.start(stage)
	}
	return &stageSpan{recorder: recorder, stage: stage, started: time.Now()}
}

func (span *stageSpan) Finish(err error) {
	if span == nil || span.recorder == nil || !span.finished.CompareAndSwap(false, true) {
		return
	}
	outcome := StageOutcomeOK
	if errors.Is(err, context.Canceled) {
		outcome = StageOutcomeCanceled
	} else if errors.Is(err, context.DeadlineExceeded) {
		outcome = StageOutcomeDeadline
	} else if err != nil {
		outcome = StageOutcomeFailed
	}
	duration := time.Since(span.started).Microseconds()
	if duration < 0 {
		duration = 0
	}
	if duration > 60_000_000 {
		duration = 60_000_000
	}
	span.recorder.finish(StageSample{Stage: span.stage, DurationUS: duration, Outcome: outcome})
}

func (recorder *StageRecorder) start(stage Stage) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.active == nil {
		recorder.active = make(map[Stage]int)
	}
	recorder.active[stage]++
}

func (recorder *StageRecorder) finish(sample StageSample) {
	if recorder == nil || !validStage(sample.Stage) || !validOutcome(sample.Outcome) {
		return
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.active[sample.Stage] > 0 {
		recorder.active[sample.Stage]--
	}
	if len(recorder.samples) >= recorder.capacity {
		recorder.dropped.Add(1)
		return
	}
	recorder.samples = append(recorder.samples, sample)
}

func (recorder *StageRecorder) Snapshot() StageSnapshot {
	if recorder == nil {
		return StageSnapshot{}
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	active := make(map[Stage]int, len(recorder.active))
	for stage, count := range recorder.active {
		active[stage] = count
	}
	return StageSnapshot{Samples: append([]StageSample(nil), recorder.samples...), Dropped: recorder.dropped.Load(), Active: active}
}

func validStage(stage Stage) bool {
	switch stage {
	case StageSessionSetup, StageAuth, StageSignaling, StageDataReady, StageRPC, StageVideoWait, StageFFmpeg, StageSessionCleanup:
		return true
	default:
		return false
	}
}

func validOutcome(outcome StageOutcome) bool {
	switch outcome {
	case StageOutcomeOK, StageOutcomeFailed, StageOutcomeCanceled, StageOutcomeDeadline:
		return true
	default:
		return false
	}
}
