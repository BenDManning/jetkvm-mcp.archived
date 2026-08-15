package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync/atomic"
	"time"
)

const schemaVersion = "jetkvm.operation.v1"

const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

const (
	OperationInventory = "inventory"
	OperationStatus    = "status"
	OperationPower     = "power"
	OperationHID       = "hid"
	OperationCapture   = "capture"
	OperationMedia     = "media"
	OperationDebugRPC  = "debug_rpc"
	OperationLifecycle = "lifecycle"
)

const (
	StageTool      = "tool"
	StageAdmission = "admission"
	StageConnect   = "connect"
	StageAuth      = "auth"
	StageSignaling = "signaling"
	StageRPC       = "rpc"
	StageCapture   = "capture"
	StageCleanup   = "cleanup"
	StageShutdown  = "shutdown"
)

const (
	CodeSuccess = "success"
)

const (
	OutcomeSucceeded = "succeeded"
	OutcomeFailed    = "failed"
	OutcomeNotSent   = "not_sent"
	OutcomeUnknown   = "unknown"
)

type correlationKey struct{}

type Recorder struct {
	writer   io.Writer
	events   chan queuedEvent
	terminal chan queuedEvent
	closed   atomic.Bool
}

type queuedEvent struct {
	line  []byte
	flush chan struct{}
}

type Span struct {
	recorder      *Recorder
	correlationID string
	transport     string
	operation     string
	started       time.Time
}

type StageSpan struct {
	operation *Span
	stage     string
	started   time.Time
}

type event struct {
	Schema        string `json:"schema"`
	CorrelationID string `json:"correlation_id"`
	Transport     string `json:"transport"`
	Operation     string `json:"operation"`
	Stage         string `json:"stage"`
	DurationMS    int64  `json:"duration_ms"`
	Code          string `json:"code"`
	Outcome       string `json:"outcome"`
}

var fallbackCorrelation atomic.Uint64

func New(writer io.Writer) *Recorder {
	recorder := &Recorder{
		writer: writer, events: make(chan queuedEvent, 256), terminal: make(chan queuedEvent, 256),
	}
	if writer != nil {
		go recorder.writeEvents()
	}
	return recorder
}

func (recorder *Recorder) Start(ctx context.Context, transport, operation string) (context.Context, *Span) {
	correlationID := newCorrelationID()
	span := &Span{recorder: recorder, correlationID: correlationID, transport: transport, operation: operation, started: time.Now()}
	return context.WithValue(ctx, correlationKey{}, span), span
}

func CorrelationID(ctx context.Context) string {
	span, _ := ctx.Value(correlationKey{}).(*Span)
	if span == nil {
		return ""
	}
	return span.correlationID
}

func BeginStage(ctx context.Context, stage string) *StageSpan {
	span, _ := ctx.Value(correlationKey{}).(*Span)
	return &StageSpan{operation: span, stage: stage, started: time.Now()}
}

func (stage *StageSpan) Record(code, outcome string) {
	if stage == nil || stage.operation == nil {
		return
	}
	stage.operation.record(stage.stage, code, outcome, time.Since(stage.started))
}

func (span *Span) Record(stage, code, outcome string) {
	if span == nil {
		return
	}
	span.record(stage, code, outcome, time.Since(span.started))
}

func (span *Span) record(stage, code, outcome string, elapsed time.Duration) {
	if span == nil || span.recorder == nil || span.recorder.writer == nil {
		return
	}
	if !validTransport(span.transport) || !validOperation(span.operation) || !validStage(stage) || !validCode(code) || !validOutcome(outcome) {
		return
	}
	duration := elapsed.Milliseconds()
	if duration < 0 {
		duration = 0
	}
	if duration > 60_000 {
		duration = 60_000
	}
	value := event{
		Schema: schemaVersion, CorrelationID: span.correlationID, Transport: span.transport,
		Operation: span.operation, Stage: stage, DurationMS: duration, Code: code, Outcome: outcome,
	}
	line, err := json.Marshal(value)
	if err != nil || span.recorder.closed.Load() {
		return
	}
	line = append(line, '\n')
	queued := queuedEvent{line: line}
	if stage == StageTool {
		select {
		case span.recorder.events <- queued:
			return
		default:
		}
		select {
		case span.recorder.terminal <- queued:
		default:
		}
		return
	}
	select {
	case span.recorder.events <- queued:
	default:
	}
}

func validTransport(value string) bool {
	return value == TransportStdio || value == TransportHTTP
}

func validOperation(value string) bool {
	switch value {
	case OperationInventory, OperationStatus, OperationPower, OperationHID, OperationCapture, OperationMedia, OperationDebugRPC, OperationLifecycle:
		return true
	default:
		return false
	}
}

func validStage(value string) bool {
	switch value {
	case StageTool, StageAdmission, StageConnect, StageAuth, StageSignaling, StageRPC, StageCapture, StageCleanup, StageShutdown:
		return true
	default:
		return false
	}
}

func validCode(value string) bool {
	switch value {
	case CodeSuccess, "operation_failed", "canceled", "timeout", "invalid_input", "busy", "authentication_failed", "device_unavailable", "video_unavailable", "no_signal", "protocol_error":
		return true
	default:
		return false
	}
}

func validOutcome(value string) bool {
	switch value {
	case OutcomeSucceeded, OutcomeFailed, OutcomeNotSent, OutcomeUnknown:
		return true
	default:
		return false
	}
}

func (recorder *Recorder) Close(ctx context.Context) error {
	if recorder == nil || recorder.writer == nil || !recorder.closed.CompareAndSwap(false, true) {
		return nil
	}
	flushed := make(chan struct{})
	select {
	case recorder.events <- queuedEvent{flush: flushed}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-flushed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (recorder *Recorder) writeEvents() {
	for {
		select {
		case queued := <-recorder.events:
			if queued.flush != nil {
				recorder.drain()
				close(queued.flush)
				return
			}
			recorder.writeQueued(queued)
		default:
			select {
			case queued := <-recorder.events:
				if queued.flush != nil {
					recorder.drain()
					close(queued.flush)
					return
				}
				recorder.writeQueued(queued)
			case queued := <-recorder.terminal:
				recorder.writeQueued(queued)
			}
		}
	}
}

func (recorder *Recorder) drain() {
	for {
		select {
		case queued := <-recorder.terminal:
			recorder.writeQueued(queued)
			continue
		default:
		}
		select {
		case queued := <-recorder.events:
			recorder.writeQueued(queued)
		default:
			return
		}
	}
}

func (recorder *Recorder) writeQueued(queued queuedEvent) {
	if queued.flush == nil {
		_, _ = recorder.writer.Write(queued.line)
	}
}

func newCorrelationID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "op_" + hex.EncodeToString(raw[:])
	}
	sequence := fallbackCorrelation.Add(1)
	var fallback [12]byte
	for index := range fallback {
		fallback[len(fallback)-1-index] = byte(sequence)
		sequence >>= 8
	}
	return "op_" + hex.EncodeToString(fallback[:])
}
