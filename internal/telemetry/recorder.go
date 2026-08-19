package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	operationSchemaVersion = "jetkvm.operation.v3"
	sessionSchemaVersion   = "jetkvm.session.v1"
)

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

type DeviceRef string

type SessionRef string

type SessionEvent string

const (
	SessionConnectionAttemptCompleted SessionEvent = "connection_attempt_completed"
	SessionGenerationActive           SessionEvent = "generation_active"
	SessionGenerationReused           SessionEvent = "generation_reused"
	SessionIdleReleased               SessionEvent = "idle_released"
	SessionExplicitlyReleased         SessionEvent = "explicitly_released"
	SessionTakenOver                  SessionEvent = "session_taken_over"
	SessionOwnershipUncertain         SessionEvent = "ownership_uncertain"
	SessionCleanupTimeout             SessionEvent = "cleanup_timeout"
	SessionShutdownClosed             SessionEvent = "shutdown_closed"
)

type Recorder struct {
	writer            io.Writer
	serverVersion     string
	processInstanceID string
	events            chan queuedEvent
	terminal          chan queuedEvent
	closed            atomic.Bool
	transportOnce     sync.Once
	transport         string
	droppedEvents     atomic.Uint64
	writerFailed      atomic.Bool
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
	mu            sync.RWMutex
	deviceRef     DeviceRef
	sessionRef    SessionRef
}

type StageSpan struct {
	operation *Span
	stage     string
	started   time.Time
}

type event struct {
	Schema            string `json:"schema"`
	Time              string `json:"time"`
	ProcessInstanceID string `json:"process_instance_id"`
	ServerVersion     string `json:"server_version"`
	CorrelationID     string `json:"correlation_id"`
	Transport         string `json:"transport"`
	Operation         string `json:"operation"`
	Stage             string `json:"stage"`
	DurationMS        int64  `json:"duration_ms"`
	Code              string `json:"code"`
	Outcome           string `json:"outcome"`
	DeviceRef         string `json:"device_ref,omitempty"`
	SessionRef        string `json:"session_ref,omitempty"`
}

type sessionEvent struct {
	Schema            string `json:"schema"`
	Time              string `json:"time"`
	ProcessInstanceID string `json:"process_instance_id"`
	ServerVersion     string `json:"server_version"`
	DeviceRef         string `json:"device_ref"`
	SessionRef        string `json:"session_ref,omitempty"`
	Event             string `json:"event"`
	DurationMS        int64  `json:"duration_ms"`
	Code              string `json:"code"`
	Outcome           string `json:"outcome"`
}

type shutdownEvent struct {
	event
	DroppedEvents uint64 `json:"dropped_events"`
	WriterFailed  bool   `json:"writer_failed"`
}

var fallbackCorrelation atomic.Uint64
var processIdentityOnce sync.Once
var processInstanceID string

func New(writer io.Writer, serverVersion string) *Recorder {
	processIdentityOnce.Do(func() {
		processInstanceID, _ = newRandomID("proc_")
	})
	if processInstanceID == "" {
		writer = nil
	}
	recorder := &Recorder{
		writer: writer, serverVersion: serverVersion, processInstanceID: processInstanceID,
		events: make(chan queuedEvent, 256), terminal: make(chan queuedEvent, 256),
	}
	if writer != nil {
		go recorder.writeEvents()
	}
	return recorder
}

func (recorder *Recorder) Start(ctx context.Context, transport, operation string) (context.Context, *Span) {
	if validTransport(transport) {
		recorder.transportOnce.Do(func() { recorder.transport = transport })
	}
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

func BindDevice(ctx context.Context, ref DeviceRef) {
	if ctx == nil {
		return
	}
	span, _ := ctx.Value(correlationKey{}).(*Span)
	if span == nil || !validReference(string(ref), "dev_") {
		return
	}
	span.mu.Lock()
	span.deviceRef = ref
	span.mu.Unlock()
}

func BindSession(ctx context.Context, ref SessionRef) {
	if ctx == nil {
		return
	}
	span, _ := ctx.Value(correlationKey{}).(*Span)
	if span == nil || !validReference(string(ref), "ses_") {
		return
	}
	span.mu.Lock()
	span.sessionRef = ref
	span.mu.Unlock()
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
	span.mu.RLock()
	deviceRef, sessionRef := span.deviceRef, span.sessionRef
	span.mu.RUnlock()
	value := event{
		Schema: operationSchemaVersion, Time: time.Now().UTC().Format(time.RFC3339Nano),
		ProcessInstanceID: span.recorder.processInstanceID, ServerVersion: span.recorder.serverVersion,
		CorrelationID: span.correlationID, Transport: span.transport,
		Operation: span.operation, Stage: stage, DurationMS: duration, Code: code, Outcome: outcome,
		DeviceRef: string(deviceRef), SessionRef: string(sessionRef),
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
			span.recorder.droppedEvents.Add(1)
		}
		return
	}
	select {
	case span.recorder.events <- queued:
	default:
		span.recorder.droppedEvents.Add(1)
	}
}

func (recorder *Recorder) RecordSession(deviceRef DeviceRef, sessionRef SessionRef, eventName SessionEvent, code, outcome string, elapsed time.Duration) {
	missingSessionRefAllowed := eventName == SessionConnectionAttemptCompleted && outcome != OutcomeSucceeded
	if recorder == nil || recorder.writer == nil || recorder.closed.Load() ||
		!validReference(string(deviceRef), "dev_") ||
		(sessionRef == "" && !missingSessionRefAllowed) ||
		(sessionRef != "" && !validReference(string(sessionRef), "ses_")) ||
		!validSessionEvent(eventName) || !validCode(code) || !validOutcome(outcome) {
		return
	}
	duration := elapsed.Milliseconds()
	if duration < 0 {
		duration = 0
	}
	if duration > 60_000 {
		duration = 60_000
	}
	value := sessionEvent{
		Schema: sessionSchemaVersion, Time: time.Now().UTC().Format(time.RFC3339Nano),
		ProcessInstanceID: recorder.processInstanceID, ServerVersion: recorder.serverVersion,
		DeviceRef: string(deviceRef), SessionRef: string(sessionRef), Event: string(eventName),
		DurationMS: duration, Code: code, Outcome: outcome,
	}
	line, err := json.Marshal(value)
	if err != nil {
		return
	}
	select {
	case recorder.events <- queuedEvent{line: append(line, '\n')}:
	default:
		recorder.droppedEvents.Add(1)
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
	case CodeSuccess, "operation_failed", "canceled", "timeout", "invalid_input", "busy", "authentication_failed", "device_unavailable", "video_unavailable", "no_signal", "protocol_error", "session_released", "session_taken_over", "ownership_uncertain", "telemetry_summary", "telemetry_writer_failure":
		return true
	default:
		return false
	}
}

func validSessionEvent(value SessionEvent) bool {
	switch value {
	case SessionConnectionAttemptCompleted, SessionGenerationActive, SessionGenerationReused,
		SessionIdleReleased, SessionExplicitlyReleased, SessionTakenOver,
		SessionOwnershipUncertain, SessionCleanupTimeout, SessionShutdownClosed:
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
			if recorder.handleQueued(queued) {
				return
			}
		default:
			select {
			case queued := <-recorder.events:
				if recorder.handleQueued(queued) {
					return
				}
			case queued := <-recorder.terminal:
				recorder.writeQueued(queued)
			}
		}
	}
}

func (recorder *Recorder) handleQueued(queued queuedEvent) bool {
	if queued.flush == nil {
		recorder.writeQueued(queued)
		return false
	}
	recorder.drain()
	recorder.writeShutdownSummary()
	close(queued.flush)
	return true
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
		if written, err := recorder.writer.Write(queued.line); err != nil || written != len(queued.line) {
			recorder.writerFailed.Store(true)
		}
	}
}

func (recorder *Recorder) writeShutdownSummary() {
	if !validTransport(recorder.transport) {
		return
	}
	correlationID := newCorrelationID()
	line := recorder.shutdownSummaryLine(correlationID)
	if written, err := recorder.writer.Write(line); err == nil && written == len(line) {
		return
	}

	// A sink can accept the summary bytes and still report an error. Make one
	// bounded attempt to record that later failure as a distinct event so the
	// already-retained summary is not contradicted.
	recorder.writerFailed.Store(true)
	line = recorder.writerFailureLine(correlationID)
	_, _ = recorder.writer.Write(line)
}

func (recorder *Recorder) shutdownSummaryLine(correlationID string) []byte {
	outcome := OutcomeSucceeded
	if recorder.droppedEvents.Load() > 0 || recorder.writerFailed.Load() {
		outcome = OutcomeFailed
	}
	value := shutdownEvent{
		event:         recorder.shutdownEvent(correlationID, "telemetry_summary", outcome),
		DroppedEvents: recorder.droppedEvents.Load(),
		WriterFailed:  recorder.writerFailed.Load(),
	}
	line, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return append(line, '\n')
}

func (recorder *Recorder) writerFailureLine(correlationID string) []byte {
	line, err := json.Marshal(recorder.shutdownEvent(correlationID, "telemetry_writer_failure", OutcomeFailed))
	if err != nil {
		return nil
	}
	return append(line, '\n')
}

func (recorder *Recorder) shutdownEvent(correlationID, code, outcome string) event {
	return event{
		Schema: operationSchemaVersion, Time: time.Now().UTC().Format(time.RFC3339Nano),
		ProcessInstanceID: recorder.processInstanceID, ServerVersion: recorder.serverVersion,
		CorrelationID: correlationID, Transport: recorder.transport,
		Operation: OperationLifecycle, Stage: StageShutdown,
		Code: code, Outcome: outcome,
	}
}

func newCorrelationID() string {
	if id, ok := newRandomID("op_"); ok {
		return id
	}

	// Correlation remains unique within the process if the platform random source
	// is temporarily unavailable. Process identity has no such fallback because
	// its contract is explicitly random.
	sequence := fallbackCorrelation.Add(1)
	var fallback [12]byte
	for index := range fallback {
		fallback[len(fallback)-1-index] = byte(sequence)
		sequence >>= 8
	}
	return "op_" + hex.EncodeToString(fallback[:])
}

func newRandomID(prefix string) (string, bool) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return prefix + hex.EncodeToString(raw[:]), true
	}
	return "", false
}

func NewDeviceRef() (DeviceRef, error) {
	value, ok := newRandomID("dev_")
	if !ok {
		return "", io.ErrUnexpectedEOF
	}
	return DeviceRef(value), nil
}

func NewSessionRef() (SessionRef, error) {
	value, ok := newRandomID("ses_")
	if !ok {
		return "", io.ErrUnexpectedEOF
	}
	return SessionRef(value), nil
}

func validReference(value, prefix string) bool {
	if len(value) != len(prefix)+24 || value[:len(prefix)] != prefix {
		return false
	}
	_, err := hex.DecodeString(value[len(prefix):])
	return err == nil
}
