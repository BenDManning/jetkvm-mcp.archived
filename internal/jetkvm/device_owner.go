package jetkvm

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

const ownerCleanupTimeout = 5 * time.Second

type ownerOwnership uint8

const (
	ownerOwnershipIdle ownerOwnership = iota
	ownerOwnershipActive
	ownerOwnershipStopped
)

type ownerTransition uint8

const (
	ownerTransitionNone ownerTransition = iota
	ownerTransitionConnecting
	ownerTransitionClosing
)

type ownerHealth uint8

const (
	ownerHealthUnavailable ownerHealth = iota
	ownerHealthHealthy
	ownerHealthDegraded
)

type ownerFreshness uint8

const (
	ownerFreshnessUnknown ownerFreshness = iota
	ownerFreshnessCurrent
)

type ownerProvenance uint8

const (
	ownerProvenanceNone ownerProvenance = iota
	ownerProvenanceOperation
)

type ownerObservations struct {
	Completed uint64
	Failed    uint64
}

// ownerSnapshot is an immutable value view of a device owner. Transport,
// waiter, cancellation, and worker state deliberately remain outside it.
type ownerSnapshot struct {
	Ownership                 ownerOwnership
	Transition                ownerTransition
	Health                    ownerHealth
	Freshness                 ownerFreshness
	Provenance                ownerProvenance
	Observations              ownerObservations
	CapabilityProfileRevision uint64
}

type ownerDispatchEvidence uint8

const (
	ownerDispatchNotSent ownerDispatchEvidence = iota
	ownerDispatchStarted
	ownerDispatchCompleted
)

type ownerWorkerEvidence struct {
	owner      uint64
	generation uint64
	worker     uint64
	dispatch   ownerDispatchEvidence
}

type ownerResult struct {
	err        error
	panicValue any
}

type ownerCompletion struct {
	evidence   ownerWorkerEvidence
	err        error
	connected  bool
	panicValue any
}

type ownerWorkerState struct {
	evidence ownerWorkerEvidence
	cancel   context.CancelFunc
	// reply is this worker's single reserved terminal path. It cannot fill
	// before its one completion and is never shared with another waiter.
	reply    chan ownerResult
	canceled bool
}

type ownerRegistration struct {
	evidence ownerWorkerEvidence
	reply    chan ownerResult
	err      error
}

type registerOwnerWorker struct {
	cancel context.CancelFunc
	reply  chan ownerRegistration
}

func (registerOwnerWorker) ownerCommand() {}

type beginOwnerDispatch struct {
	evidence ownerWorkerEvidence
	ctx      context.Context
	reply    chan bool
}

func (beginOwnerDispatch) ownerCommand() {}

type cancelOwnerWorker struct {
	evidence ownerWorkerEvidence
	reply    chan bool
}

func (cancelOwnerWorker) ownerCommand() {}

type completeOwnerWorker struct{ completion ownerCompletion }
type stopDeviceOwner struct{ reply chan struct{} }

func (completeOwnerWorker) ownerCommand() {}
func (stopDeviceOwner) ownerCommand()     {}

type ownerCommand interface{ ownerCommand() }

type deviceOwner struct {
	id        uint64
	device    DeviceConfig
	connector SessionConnector
	commands  chan ownerCommand
	done      chan struct{}
	snapshot  atomic.Value
}

var nextOwnerID atomic.Uint64

func newDeviceOwner(device DeviceConfig, connector SessionConnector) *deviceOwner {
	owner := &deviceOwner{
		id: nextOwnerID.Add(1), device: device, connector: connector,
		commands: make(chan ownerCommand), done: make(chan struct{}),
	}
	owner.snapshot.Store(ownerSnapshot{
		Ownership: ownerOwnershipIdle, Transition: ownerTransitionNone,
		Health: ownerHealthUnavailable, Freshness: ownerFreshnessUnknown,
		Provenance: ownerProvenanceNone, CapabilityProfileRevision: 1,
	})
	go owner.loop()
	return owner
}

func (owner *deviceOwner) Snapshot() ownerSnapshot {
	return owner.snapshot.Load().(ownerSnapshot)
}

func (owner *deviceOwner) Run(ctx context.Context, operation func(context.Context, Session) error) error {
	if operation == nil {
		return errors.New("device operation is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	registration := registerOwnerWorker{cancel: cancel, reply: make(chan ownerRegistration, 1)}
	select {
	case owner.commands <- registration:
	case <-owner.done:
		cancel()
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	case <-ctx.Done():
		cancel()
		return classifyOperationError(ctx.Err(), ToolOutcomeNotSent)
	}
	registered := <-registration.reply
	if registered.err != nil {
		cancel()
		return registered.err
	}
	go owner.runWorker(workerCtx, registered.evidence, operation)

	select {
	case result := <-registered.reply:
		return returnOwnerResult(result)
	case <-ctx.Done():
		if owner.cancelBeforeDispatch(registered.evidence) {
			cancel()
			return classifyOperationError(ctx.Err(), ToolOutcomeNotSent)
		}
		return returnOwnerResult(<-registered.reply)
	}
}

func returnOwnerResult(result ownerResult) error {
	if result.panicValue != nil {
		panic(result.panicValue)
	}
	return result.err
}

func (owner *deviceOwner) runWorker(ctx context.Context, evidence ownerWorkerEvidence, operation func(context.Context, Session) error) {
	connected := false
	defer func() {
		if panicValue := recover(); panicValue != nil {
			owner.complete(ownerCompletion{evidence: evidence, connected: connected, panicValue: panicValue})
		}
	}()
	session, err := owner.connector.Connect(ctx, owner.device)
	if err != nil {
		owner.complete(ownerCompletion{evidence: evidence, err: classifyConnectFailure(ctx, err)})
		return
	}
	if session == nil {
		owner.complete(ownerCompletion{
			evidence: evidence, err: classifyOperationError(ErrInvalidResponse, ToolOutcomeNotSent),
		})
		return
	}
	connected = true
	if !owner.beginDispatch(ctx, evidence) {
		cleanup := telemetry.BeginStage(ctx, telemetry.StageCleanup)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), ownerCleanupTimeout)
		_ = session.Close(cleanupCtx)
		cancel()
		finishTelemetryStage(cleanup, nil)
		dispatchErr := ctx.Err()
		if dispatchErr == nil {
			dispatchErr = ErrSessionClosed
		}
		owner.complete(ownerCompletion{evidence: evidence, err: classifyOperationError(dispatchErr, ToolOutcomeNotSent), connected: true})
		return
	}
	evidence.dispatch = ownerDispatchStarted
	var result error
	func() {
		defer func() {
			cleanup := telemetry.BeginStage(ctx, telemetry.StageCleanup)
			_ = session.Close(ctx)
			finishTelemetryStage(cleanup, nil)
		}()
		result = operation(ctx, session)
	}()
	evidence.dispatch = ownerDispatchCompleted
	owner.complete(ownerCompletion{evidence: evidence, err: result, connected: true})
}

func classifyConnectFailure(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return classifyOperationError(ctxErr, ToolOutcomeNotSent)
	}
	var classified interface{ ToolErrorOutcome() string }
	if errors.As(err, &classified) {
		return err
	}
	return classifyOperationError(err, ToolOutcomeNotSent)
}

func (owner *deviceOwner) beginDispatch(ctx context.Context, evidence ownerWorkerEvidence) bool {
	command := beginOwnerDispatch{evidence: evidence, ctx: ctx, reply: make(chan bool, 1)}
	select {
	case owner.commands <- command:
		return <-command.reply
	case <-owner.done:
		return false
	}
}

func (owner *deviceOwner) cancelBeforeDispatch(evidence ownerWorkerEvidence) bool {
	command := cancelOwnerWorker{evidence: evidence, reply: make(chan bool, 1)}
	select {
	case owner.commands <- command:
		return <-command.reply
	case <-owner.done:
		return false
	}
}

func (owner *deviceOwner) complete(completion ownerCompletion) {
	select {
	case owner.commands <- completeOwnerWorker{completion: completion}:
	case <-owner.done:
	}
}

func (owner *deviceOwner) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	command := stopDeviceOwner{reply: make(chan struct{}, 1)}
	select {
	case owner.commands <- command:
	case <-owner.done:
		return nil
	}
	select {
	case <-command.reply:
		return nil
	case <-owner.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (owner *deviceOwner) loop() {
	defer close(owner.done)
	workers := make(map[uint64]*ownerWorkerState)
	snapshot := owner.Snapshot()
	var nextGeneration, nextWorker uint64
	var stopping bool
	var stopWaiters []chan struct{}
	publish := func() { owner.publishSnapshot(workers, stopping, snapshot) }
	for command := range owner.commands {
		switch command := command.(type) {
		case registerOwnerWorker:
			if stopping {
				command.reply <- ownerRegistration{err: classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)}
				continue
			}
			nextGeneration++
			nextWorker++
			evidence := ownerWorkerEvidence{
				owner: owner.id, generation: nextGeneration, worker: nextWorker, dispatch: ownerDispatchNotSent,
			}
			worker := &ownerWorkerState{evidence: evidence, cancel: command.cancel, reply: make(chan ownerResult, 1)}
			workers[evidence.worker] = worker
			publish()
			command.reply <- ownerRegistration{evidence: evidence, reply: worker.reply}
		case beginOwnerDispatch:
			worker := matchingOwnerWorker(workers, command.evidence)
			allowed := worker != nil && !worker.canceled && !stopping && command.ctx.Err() == nil
			if allowed {
				worker.evidence.dispatch = ownerDispatchStarted
				publish()
			}
			command.reply <- allowed
		case cancelOwnerWorker:
			worker := matchingOwnerWorker(workers, command.evidence)
			canceled := worker != nil && !worker.canceled && worker.evidence.dispatch == ownerDispatchNotSent
			if canceled {
				worker.canceled = true
				worker.cancel()
				publish()
			}
			command.reply <- canceled
		case completeOwnerWorker:
			completion := command.completion
			worker := matchingOwnerWorker(workers, completion.evidence)
			if worker == nil {
				continue
			}
			delete(workers, completion.evidence.worker)
			if !worker.canceled {
				if completion.err == nil && completion.panicValue == nil {
					snapshot.Observations.Completed++
				} else {
					snapshot.Observations.Failed++
				}
				snapshot.Freshness = ownerFreshnessCurrent
				snapshot.Provenance = ownerProvenanceOperation
				if completion.connected {
					snapshot.Health = ownerHealthHealthy
				} else if completion.err != nil {
					snapshot.Health = ownerHealthDegraded
				}
				worker.reply <- ownerResult{err: completion.err, panicValue: completion.panicValue}
			}
			publish()
		case stopDeviceOwner:
			if !stopping {
				stopping = true
				for _, worker := range workers {
					worker.cancel()
				}
				publish()
			}
			stopWaiters = append(stopWaiters, command.reply)
		}
		if stopping && len(workers) == 0 {
			publish()
			for _, waiter := range stopWaiters {
				waiter <- struct{}{}
			}
			return
		}
	}
}

func matchingOwnerWorker(workers map[uint64]*ownerWorkerState, evidence ownerWorkerEvidence) *ownerWorkerState {
	worker := workers[evidence.worker]
	if worker == nil || worker.evidence.owner != evidence.owner || worker.evidence.generation != evidence.generation {
		return nil
	}
	return worker
}

func (owner *deviceOwner) publishSnapshot(workers map[uint64]*ownerWorkerState, stopping bool, snapshot ownerSnapshot) {
	active, connecting := 0, false
	for _, worker := range workers {
		if worker.canceled {
			continue
		}
		active++
		connecting = connecting || worker.evidence.dispatch == ownerDispatchNotSent
	}
	if stopping {
		snapshot.Ownership = ownerOwnershipStopped
		if len(workers) == 0 {
			snapshot.Transition = ownerTransitionNone
		} else {
			snapshot.Transition = ownerTransitionClosing
		}
	} else if active == 0 {
		snapshot.Ownership = ownerOwnershipIdle
		snapshot.Transition = ownerTransitionNone
	} else {
		snapshot.Ownership = ownerOwnershipActive
		snapshot.Transition = ownerTransitionNone
		if connecting {
			snapshot.Transition = ownerTransitionConnecting
		}
	}
	owner.snapshot.Store(snapshot)
}
