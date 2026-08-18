package jetkvm

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

const (
	ownerCleanupTimeout = 5 * time.Second
	ownerConnectTimeout = 15 * time.Second
)

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
	panicValue any
}

type ownerWorkerState struct {
	evidence  ownerWorkerEvidence
	ctx       context.Context
	cancel    context.CancelFunc
	operation func(context.Context, Session) error
	reply     chan ownerResult
	canceled  bool
}

type ownerRegistration struct {
	evidence ownerWorkerEvidence
	reply    chan ownerResult
	err      error
}

type registerOwnerWorker struct {
	ctx       context.Context
	operation func(context.Context, Session) error
	reply     chan ownerRegistration
}

func (registerOwnerWorker) ownerCommand() {}

type cancelOwnerWorker struct {
	evidence ownerWorkerEvidence
	err      error
	reply    chan bool
}

func (cancelOwnerWorker) ownerCommand() {}

type completeOwnerWorker struct{ completion ownerCompletion }

func (completeOwnerWorker) ownerCommand() {}

type completeOwnerAttempt struct {
	generation uint64
	session    ConnectedSession
	err        error
	cleanupErr error
	panicValue any
}

func (completeOwnerAttempt) ownerCommand() {}

type ownerGenerationEnded struct{ generation uint64 }

func (ownerGenerationEnded) ownerCommand() {}

type ownerIdleExpired struct{ generation uint64 }

func (ownerIdleExpired) ownerCommand() {}

type completeOwnerCleanup struct {
	generation uint64
	err        error
}

func (completeOwnerCleanup) ownerCommand() {}

type stopDeviceOwner struct{ reply chan error }

func (stopDeviceOwner) ownerCommand() {}

type ownerCommand interface{ ownerCommand() }

type ownerTimer interface{ Stop() bool }

type ownerSettings struct {
	connectionAttempts chan struct{}
	idleTimeout        time.Duration
	connectTimeout     time.Duration
	afterFunc          func(time.Duration, func()) ownerTimer
}

type deviceOwner struct {
	id        uint64
	device    DeviceConfig
	connector SessionConnector
	settings  ownerSettings
	commands  chan ownerCommand
	done      chan struct{}
	snapshot  atomic.Value
}

type ownerAttemptState struct {
	generation uint64
	cancel     context.CancelFunc
}

type ownerGenerationState struct {
	id      uint64
	session ConnectedSession
	ctx     context.Context
	cancel  context.CancelFunc
}

type ownerCleanupState struct {
	generation uint64
	err        error
}

var nextOwnerID atomic.Uint64

func newDeviceOwner(device DeviceConfig, connector SessionConnector) *deviceOwner {
	return newDeviceOwnerWithSettings(device, connector, ownerSettings{})
}

func newDeviceOwnerWithSettings(device DeviceConfig, connector SessionConnector, settings ownerSettings) *deviceOwner {
	if settings.connectionAttempts == nil {
		settings.connectionAttempts = make(chan struct{}, defaultMaxConnectionAttempts)
	}
	if settings.idleTimeout <= 0 {
		settings.idleTimeout = defaultSessionIdleTimeout
	}
	if settings.connectTimeout <= 0 {
		settings.connectTimeout = ownerConnectTimeout
	}
	if settings.afterFunc == nil {
		settings.afterFunc = func(delay time.Duration, fire func()) ownerTimer {
			return time.AfterFunc(delay, fire)
		}
	}
	owner := &deviceOwner{
		id: nextOwnerID.Add(1), device: device, connector: connector, settings: settings,
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
	registration := registerOwnerWorker{ctx: ctx, operation: operation, reply: make(chan ownerRegistration, 1)}
	select {
	case owner.commands <- registration:
	case <-owner.done:
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	case <-ctx.Done():
		return classifyOperationError(ctx.Err(), ToolOutcomeNotSent)
	}
	registered := <-registration.reply
	if registered.err != nil {
		return registered.err
	}
	select {
	case result := <-registered.reply:
		return returnOwnerResult(result)
	case <-ctx.Done():
		cancellation := cancelOwnerWorker{
			evidence: registered.evidence, err: ctx.Err(), reply: make(chan bool, 1),
		}
		select {
		case owner.commands <- cancellation:
			if <-cancellation.reply {
				return classifyOperationError(ctx.Err(), ToolOutcomeNotSent)
			}
			return returnOwnerResult(<-registered.reply)
		case <-owner.done:
			return classifyOperationError(ctx.Err(), ToolOutcomeNotSent)
		}
	}
}

func returnOwnerResult(result ownerResult) error {
	if result.panicValue != nil {
		panic(result.panicValue)
	}
	return result.err
}

func (owner *deviceOwner) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	command := stopDeviceOwner{reply: make(chan error, 1)}
	select {
	case owner.commands <- command:
	case <-owner.done:
		return nil
	}
	select {
	case err := <-command.reply:
		return err
	case <-owner.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (owner *deviceOwner) loop() {
	defer close(owner.done)
	workers := make(map[uint64]*ownerWorkerState)
	queue := make([]uint64, 0)
	snapshot := owner.Snapshot()
	var nextGeneration, nextWorker, activeWorker uint64
	var attempt *ownerAttemptState
	var generation *ownerGenerationState
	var cleanup *ownerCleanupState
	var idleTimer ownerTimer
	var stopping bool
	var stopWaiters []chan error

	publish := func() {
		switch {
		case stopping:
			snapshot.Ownership = ownerOwnershipStopped
		case generation != nil:
			snapshot.Ownership = ownerOwnershipActive
		case cleanup != nil:
			snapshot.Ownership = ownerOwnershipActive
		default:
			snapshot.Ownership = ownerOwnershipIdle
		}
		switch {
		case attempt != nil:
			snapshot.Transition = ownerTransitionConnecting
		case cleanup != nil:
			snapshot.Transition = ownerTransitionClosing
		default:
			snapshot.Transition = ownerTransitionNone
		}
		owner.snapshot.Store(snapshot)
	}

	removeQueued := func(workerID uint64) {
		for index, queued := range queue {
			if queued == workerID {
				queue = append(queue[:index], queue[index+1:]...)
				return
			}
		}
	}
	failQueued := func(result ownerResult) {
		for _, workerID := range queue {
			if worker := workers[workerID]; worker != nil {
				delete(workers, workerID)
				worker.reply <- result
			}
		}
		queue = queue[:0]
	}

	startAttempt := func() error {
		if attempt != nil || generation != nil || cleanup != nil || stopping {
			return nil
		}
		if !tryAcquire(owner.settings.connectionAttempts) {
			return busyNotSent()
		}
		nextGeneration++
		attemptBase := context.Background()
		if len(queue) != 0 {
			if worker := workers[queue[0]]; worker != nil {
				attemptBase = context.WithoutCancel(worker.ctx)
			}
		}
		attemptCtx, cancel := context.WithTimeout(attemptBase, owner.settings.connectTimeout)
		attempt = &ownerAttemptState{generation: nextGeneration, cancel: cancel}
		publish()
		go owner.runAttempt(attemptCtx, nextGeneration)
		return nil
	}

	startNext := func() {
		if activeWorker != 0 || generation == nil || stopping {
			return
		}
		for len(queue) != 0 {
			workerID := queue[0]
			queue = queue[1:]
			worker := workers[workerID]
			if worker == nil {
				continue
			}
			if err := worker.ctx.Err(); err != nil {
				delete(workers, workerID)
				worker.reply <- ownerResult{err: classifyOperationError(err, ToolOutcomeNotSent)}
				continue
			}
			if idleTimer != nil {
				idleTimer.Stop()
				idleTimer = nil
			}
			activeWorker = workerID
			worker.evidence.generation = generation.id
			worker.evidence.dispatch = ownerDispatchStarted
			operationCtx, cancel := context.WithCancel(worker.ctx)
			worker.cancel = cancel
			stopGenerationCancel := context.AfterFunc(generation.ctx, cancel)
			go owner.runOperation(operationCtx, stopGenerationCancel, worker.evidence, generation.session, worker.operation)
			return
		}
	}

	scheduleIdle := func() {
		if stopping || attempt != nil || generation == nil || cleanup != nil || activeWorker != 0 || len(queue) != 0 || idleTimer != nil {
			return
		}
		generationID := generation.id
		idleTimer = owner.settings.afterFunc(owner.settings.idleTimeout, func() {
			owner.send(ownerIdleExpired{generation: generationID})
		})
	}

	startCleanup := func(reasonGeneration *ownerGenerationState) {
		if reasonGeneration == nil || cleanup != nil {
			return
		}
		if idleTimer != nil {
			idleTimer.Stop()
			idleTimer = nil
		}
		if generation != nil && generation.id == reasonGeneration.id {
			generation = nil
		}
		reasonGeneration.cancel()
		cleanup = &ownerCleanupState{generation: reasonGeneration.id}
		publish()
		go owner.runCleanup(reasonGeneration)
	}

	finishIfStopped := func() bool {
		cleanupPending := cleanup != nil && cleanup.err == nil
		if !stopping || attempt != nil || generation != nil || cleanupPending || len(workers) != 0 {
			return false
		}
		publish()
		var stopErr error
		if cleanup != nil {
			stopErr = cleanup.err
		}
		for _, waiter := range stopWaiters {
			waiter <- stopErr
		}
		return true
	}

	for command := range owner.commands {
		switch command := command.(type) {
		case registerOwnerWorker:
			if stopping {
				command.reply <- ownerRegistration{err: classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)}
				break
			}
			if cleanup != nil && cleanup.err != nil {
				command.reply <- ownerRegistration{err: busyNotSent()}
				break
			}
			nextWorker++
			evidence := ownerWorkerEvidence{owner: owner.id, worker: nextWorker, dispatch: ownerDispatchNotSent}
			worker := &ownerWorkerState{evidence: evidence, ctx: command.ctx, operation: command.operation, reply: make(chan ownerResult, 1)}
			workers[nextWorker] = worker
			queue = append(queue, nextWorker)
			if generation == nil && attempt == nil && cleanup == nil {
				if err := startAttempt(); err != nil {
					removeQueued(nextWorker)
					delete(workers, nextWorker)
					command.reply <- ownerRegistration{err: err}
					break
				}
			}
			command.reply <- ownerRegistration{evidence: evidence, reply: worker.reply}
			startNext()
			publish()

		case cancelOwnerWorker:
			worker := workers[command.evidence.worker]
			preDispatch := worker != nil && activeWorker != command.evidence.worker
			if preDispatch {
				removeQueued(command.evidence.worker)
				lastAttemptWaiter := attempt != nil && len(queue) == 0 && activeWorker == 0
				if lastAttemptWaiter {
					worker.canceled = true
					attempt.cancel()
					preDispatch = false
				} else {
					delete(workers, command.evidence.worker)
					worker.reply <- ownerResult{err: classifyOperationError(command.err, ToolOutcomeNotSent)}
				}
			} else if worker != nil && worker.cancel != nil {
				worker.cancel()
			}
			command.reply <- preDispatch
			publish()

		case completeOwnerAttempt:
			if attempt == nil || attempt.generation != command.generation {
				if command.session != nil {
					go owner.closeDetached(command.session)
				}
				break
			}
			attempt.cancel()
			attempt = nil
			for workerID, worker := range workers {
				if worker.canceled {
					delete(workers, workerID)
					err := worker.ctx.Err()
					if err == nil {
						err = context.Canceled
					}
					worker.reply <- ownerResult{err: classifyOperationError(err, ToolOutcomeNotSent)}
				}
			}
			liveQueue := queue[:0]
			for _, workerID := range queue {
				worker := workers[workerID]
				if worker == nil {
					continue
				}
				if err := worker.ctx.Err(); err != nil {
					delete(workers, workerID)
					worker.reply <- ownerResult{err: classifyOperationError(err, ToolOutcomeNotSent)}
					continue
				}
				liveQueue = append(liveQueue, workerID)
			}
			queue = liveQueue
			if command.err != nil || command.panicValue != nil {
				snapshot.Health = ownerHealthDegraded
				failQueued(ownerResult{err: command.err, panicValue: command.panicValue})
				if command.cleanupErr != nil {
					cleanup = &ownerCleanupState{generation: command.generation, err: command.cleanupErr}
				}
			} else if command.session != nil && !stopping && len(queue) != 0 {
				generationCtx, cancel := context.WithCancel(context.Background())
				generation = &ownerGenerationState{id: command.generation, session: command.session, ctx: generationCtx, cancel: cancel}
				snapshot.Health = ownerHealthHealthy
				go owner.watchGeneration(generation.id, command.session.Done())
				startNext()
			} else if command.session != nil {
				generationCtx, cancel := context.WithCancel(context.Background())
				orphan := &ownerGenerationState{id: command.generation, session: command.session, ctx: generationCtx, cancel: cancel}
				cleanup = &ownerCleanupState{generation: orphan.id}
				go owner.runCleanup(orphan)
			}
			publish()

		case completeOwnerWorker:
			completion := command.completion
			worker := workers[completion.evidence.worker]
			if worker == nil || activeWorker != completion.evidence.worker || worker.evidence.owner != completion.evidence.owner || worker.evidence.generation != completion.evidence.generation {
				break
			}
			activeWorker = 0
			delete(workers, completion.evidence.worker)
			if completion.err == nil && completion.panicValue == nil {
				snapshot.Observations.Completed++
			} else {
				snapshot.Observations.Failed++
			}
			snapshot.Freshness = ownerFreshnessCurrent
			snapshot.Provenance = ownerProvenanceOperation
			worker.reply <- ownerResult{err: completion.err, panicValue: completion.panicValue}
			startNext()
			scheduleIdle()
			publish()

		case ownerGenerationEnded:
			if generation != nil && generation.id == command.generation {
				startCleanup(generation)
			}

		case ownerIdleExpired:
			idleTimer = nil
			if generation != nil && generation.id == command.generation && activeWorker == 0 && len(queue) == 0 {
				startCleanup(generation)
			}

		case completeOwnerCleanup:
			if cleanup == nil || cleanup.generation != command.generation {
				break
			}
			if command.err != nil {
				cleanup.err = command.err
				snapshot.Health = ownerHealthDegraded
				failQueued(ownerResult{err: busyNotSent()})
			} else {
				cleanup = nil
				snapshot.Health = ownerHealthUnavailable
				if !stopping && len(queue) != 0 {
					if err := startAttempt(); err != nil {
						failQueued(ownerResult{err: err})
					}
				}
			}
			publish()

		case stopDeviceOwner:
			if !stopping {
				stopping = true
				if idleTimer != nil {
					idleTimer.Stop()
					idleTimer = nil
				}
				if attempt != nil {
					attempt.cancel()
				}
				if attempt != nil {
					for _, workerID := range queue {
						if worker := workers[workerID]; worker != nil {
							worker.canceled = true
						}
					}
					queue = queue[:0]
				} else {
					failQueued(ownerResult{err: classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)})
				}
				if active := workers[activeWorker]; active != nil && active.cancel != nil {
					active.cancel()
				}
				if generation != nil {
					startCleanup(generation)
				}
			}
			stopWaiters = append(stopWaiters, command.reply)
			publish()
		}
		if finishIfStopped() {
			return
		}
	}
}

func (owner *deviceOwner) runAttempt(ctx context.Context, generation uint64) {
	var session ConnectedSession
	var err error
	completion := completeOwnerAttempt{generation: generation}
	defer func() {
		release(owner.settings.connectionAttempts)
		if panicValue := recover(); panicValue != nil {
			completion.panicValue = panicValue
			if session != nil {
				completion.cleanupErr = owner.closeDetached(session)
			}
		} else {
			completion.session = session
			completion.err = classifyConnectFailure(ctx, err)
		}
		owner.send(completion)
	}()
	session, err = owner.connector.Connect(ctx, owner.device)
	if err == nil && session == nil {
		err = ErrInvalidResponse
	}
	if err == nil {
		var pong string
		err = session.Call(ctx, methodPing, nil, &pong)
		if err == nil && pong != "pong" {
			err = ErrInvalidResponse
		}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}
	if err != nil && session != nil {
		completion.cleanupErr = owner.closeDetached(session)
		session = nil
	}
}

func (owner *deviceOwner) runOperation(ctx context.Context, stopGenerationCancel func() bool, evidence ownerWorkerEvidence, session Session, operation func(context.Context, Session) error) {
	defer stopGenerationCancel()
	completion := ownerCompletion{evidence: evidence}
	defer func() {
		completion.evidence.dispatch = ownerDispatchCompleted
		if panicValue := recover(); panicValue != nil {
			completion.panicValue = panicValue
		}
		owner.send(completeOwnerWorker{completion: completion})
	}()
	completion.err = operation(ctx, session)
}

func (owner *deviceOwner) watchGeneration(generation uint64, done <-chan struct{}) {
	if done == nil {
		return
	}
	select {
	case <-done:
		owner.send(ownerGenerationEnded{generation: generation})
	case <-owner.done:
	}
}

func (owner *deviceOwner) runCleanup(generation *ownerGenerationState) {
	generation.cancel()
	err := owner.closeDetached(generation.session)
	owner.send(completeOwnerCleanup{generation: generation.id, err: err})
}

func (owner *deviceOwner) closeDetached(session ConnectedSession) error {
	cleanup := telemetry.BeginStage(context.Background(), telemetry.StageCleanup)
	ctx, cancel := context.WithTimeout(context.Background(), ownerCleanupTimeout)
	err := session.Close(ctx)
	cancel()
	finishTelemetryStage(cleanup, err)
	return err
}

func (owner *deviceOwner) send(command ownerCommand) {
	select {
	case owner.commands <- command:
	case <-owner.done:
	}
}

func classifyConnectFailure(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return classifyOperationError(ctxErr, ToolOutcomeNotSent)
	}
	var classified interface{ ToolErrorOutcome() string }
	if errors.As(err, &classified) {
		return err
	}
	return classifyOperationError(err, ToolOutcomeNotSent)
}
