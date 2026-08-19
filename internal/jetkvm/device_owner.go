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
	ownerOwnershipTakenOver
	ownerOwnershipUncertain
	ownerOwnershipReleased
	ownerOwnershipStopped
)

type ownerTransition uint8

const (
	ownerTransitionNone ownerTransition = iota
	ownerTransitionConnecting
	ownerTransitionDraining
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
	class     ownerOperationClass
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
	class     ownerOperationClass
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
	sessionRef telemetry.SessionRef
	err        error
	cleanupErr error
	panicValue any
}

func (completeOwnerAttempt) ownerCommand() {}

type ownerGenerationEnded struct{ generation uint64 }

func (ownerGenerationEnded) ownerCommand() {}

type ownerTakeoverRecognized struct{ generation uint64 }

func (ownerTakeoverRecognized) ownerCommand() {}

type ownerIdleExpired struct{ generation uint64 }

func (ownerIdleExpired) ownerCommand() {}

type ownerDemandChanged struct{}

func (ownerDemandChanged) ownerCommand() {}

type completeOwnerCleanup struct {
	generation uint64
	err        error
}

func (completeOwnerCleanup) ownerCommand() {}

type stopDeviceOwner struct{ reply chan error }

func (stopDeviceOwner) ownerCommand() {}

type releaseOwnerSession struct {
	ctx   context.Context
	reply chan error
}

func (releaseOwnerSession) ownerCommand() {}

type takeOverOwnerSession struct {
	ctx   context.Context
	reply chan error
}

func (takeOverOwnerSession) ownerCommand() {}

type completeOwnerValidation struct {
	generation uint64
	err        error
}

func (completeOwnerValidation) ownerCommand() {}

type ownerCommand interface{ ownerCommand() }

type ownerTimer interface{ Stop() bool }

type ownerSettings struct {
	connectionAttempts chan struct{}
	idleTimeout        time.Duration
	connectTimeout     time.Duration
	afterFunc          func(time.Duration, func()) ownerTimer
	profile            capabilityProfile
	initialOwnership   ownerOwnership
	recorder           *telemetry.Recorder
	deviceRef          telemetry.DeviceRef
}

type deviceOwner struct {
	id        uint64
	device    DeviceConfig
	connector SessionConnector
	settings  ownerSettings
	scheduler *ownerScheduler
	commands  chan ownerCommand
	done      chan struct{}
	demand    atomic.Int64
	snapshot  atomic.Value
}

type ownerAttemptState struct {
	generation    uint64
	started       time.Time
	cancel        context.CancelFunc
	takeoverReply chan error
	takeoverCtx   context.Context
}

type ownerValidationState struct {
	generation uint64
	ctx        context.Context
	reply      chan error
}

type ownerGenerationState struct {
	id        uint64
	ref       telemetry.SessionRef
	leased    bool
	session   ConnectedSession
	scheduler *generationScheduler
	ctx       context.Context
	cancel    context.CancelFunc
	watchDone chan struct{}
}

type ownerCleanupState struct {
	generation    *ownerGenerationState
	started       time.Time
	target        ownerOwnership
	err           error
	reply         chan error
	takeoverReply chan error
	takeoverCtx   context.Context
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
	if settings.profile.revision == 0 {
		settings.profile = initialCapabilityProfile
	}
	owner := &deviceOwner{
		id: nextOwnerID.Add(1), device: device, connector: connector, settings: settings,
		scheduler: newOwnerScheduler(settings.profile),
		commands:  make(chan ownerCommand), done: make(chan struct{}),
	}
	owner.snapshot.Store(ownerSnapshot{
		Ownership: settings.initialOwnership, Transition: ownerTransitionNone,
		Health: ownerHealthUnavailable, Freshness: ownerFreshnessUnknown,
		Provenance: ownerProvenanceNone, CapabilityProfileRevision: settings.profile.revision,
	})
	go owner.loop()
	return owner
}

func (owner *deviceOwner) Snapshot() ownerSnapshot {
	return owner.snapshot.Load().(ownerSnapshot)
}

func (owner *deviceOwner) recordSession(sessionRef telemetry.SessionRef, event telemetry.SessionEvent, err error, started time.Time) {
	if owner.settings.recorder == nil {
		return
	}
	elapsed := time.Duration(0)
	if !started.IsZero() {
		elapsed = time.Since(started)
	}
	code, outcome := telemetryResult(err)
	owner.settings.recorder.RecordSession(owner.settings.deviceRef, sessionRef, event, code, outcome, elapsed)
}

func (owner *deviceOwner) recordCleanupFailure(sessionRef telemetry.SessionRef, err error, started time.Time, ownershipChanged bool) {
	if errors.Is(err, context.DeadlineExceeded) {
		owner.recordSession(sessionRef, telemetry.SessionCleanupTimeout, err, started)
	}
	if ownershipChanged {
		owner.recordSession(sessionRef, telemetry.SessionOwnershipUncertain, classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed), time.Time{})
	}
}

func (owner *deviceOwner) Run(ctx context.Context, operation func(context.Context, Session) error) error {
	return owner.RunScheduled(ctx, ownerOperationOrdinary, operation)
}

func (owner *deviceOwner) RunScheduled(ctx context.Context, class ownerOperationClass, operation func(context.Context, Session) error) error {
	return owner.RunPrepared(ctx, class, nil, operation)
}

func (owner *deviceOwner) RunPrepared(ctx context.Context, class ownerOperationClass, prepare func() error, operation func(context.Context, Session) error) error {
	if operation == nil {
		return errors.New("device operation is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	owner.demand.Add(1)
	defer func() {
		if owner.demand.Add(-1) == 0 {
			owner.send(ownerDemandChanged{})
		}
	}()
	if class == ownerOperationMutation || class == ownerOperationHID {
		if err := owner.scheduler.mutation.acquire(ctx); err != nil {
			return classifyOperationError(err, ToolOutcomeNotSent)
		}
		defer owner.scheduler.mutation.release()
	}
	if prepare != nil {
		if err := prepare(); err != nil {
			return err
		}
	}
	registration := registerOwnerWorker{ctx: ctx, class: class, operation: operation, reply: make(chan ownerRegistration, 1)}
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
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (owner *deviceOwner) Release() error { return owner.ReleaseContext(context.Background()) }

func (owner *deviceOwner) ReleaseContext(ctx context.Context) error {
	command := releaseOwnerSession{ctx: ctx, reply: make(chan error, 1)}
	select {
	case owner.commands <- command:
	case <-owner.done:
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	}
	return <-command.reply
}

func (owner *deviceOwner) TakeOver() error { return owner.TakeOverContext(context.Background()) }

func (owner *deviceOwner) TakeOverContext(ctx context.Context) error {
	command := takeOverOwnerSession{ctx: ctx, reply: make(chan error, 1)}
	select {
	case owner.commands <- command:
	case <-owner.done:
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	}
	return <-command.reply
}

func (owner *deviceOwner) loop() {
	defer close(owner.done)
	workers := make(map[uint64]*ownerWorkerState)
	queue := make([]uint64, 0)
	snapshot := owner.Snapshot()
	var nextGeneration, nextWorker uint64
	var attempt *ownerAttemptState
	var validation *ownerValidationState
	var generation *ownerGenerationState
	var lastSessionRef telemetry.SessionRef
	var cleanup *ownerCleanupState
	var idleTimer ownerTimer
	var stopping bool
	var stopWaiters []chan error
	ownership := snapshot.Ownership

	publish := func() {
		snapshot.Ownership = ownership
		if stopping {
			snapshot.Ownership = ownerOwnershipStopped
		}
		switch {
		case attempt != nil:
			snapshot.Transition = ownerTransitionConnecting
		case cleanup != nil && cleanup.err == nil:
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

	startAttempt := func(takeoverReply chan error, takeoverCtx context.Context) error {
		ordinary := takeoverReply == nil
		if attempt != nil || generation != nil || cleanup != nil || validation != nil || stopping || (ordinary && ownership != ownerOwnershipIdle) {
			if takeoverReply != nil {
				if stopping {
					return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
				}
				return busyNotSent()
			}
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
		attempt = &ownerAttemptState{generation: nextGeneration, started: time.Now(), cancel: cancel, takeoverReply: takeoverReply, takeoverCtx: takeoverCtx}
		publish()
		go owner.runAttempt(attemptCtx, nextGeneration)
		return nil
	}

	startNext := func() {
		if generation == nil || stopping {
			return
		}
		if recognizedTakeover(generation.session) {
			return
		}
		for len(queue) != 0 {
			if recognizedTakeover(generation.session) {
				return
			}
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
			worker.evidence.generation = generation.id
			worker.evidence.dispatch = ownerDispatchStarted
			telemetry.BindSession(worker.ctx, generation.ref)
			if generation.leased {
				owner.recordSession(generation.ref, telemetry.SessionGenerationReused, nil, time.Time{})
			} else {
				generation.leased = true
			}
			operationCtx, cancel := context.WithCancel(worker.ctx)
			worker.cancel = cancel
			stopGenerationCancel := context.AfterFunc(generation.ctx, cancel)
			go owner.runOperation(operationCtx, stopGenerationCancel, worker.evidence, generation.session, generation.scheduler, worker.class, worker.operation)
		}
	}

	scheduleIdle := func() {
		if stopping || attempt != nil || generation == nil || cleanup != nil || len(workers) != 0 || len(queue) != 0 || owner.demand.Load() != 0 || idleTimer != nil {
			return
		}
		generationID := generation.id
		idleTimer = owner.settings.afterFunc(owner.settings.idleTimeout, func() {
			owner.send(ownerIdleExpired{generation: generationID})
		})
	}

	startCleanup := func(reasonGeneration *ownerGenerationState, target ownerOwnership, reply chan error) {
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
		cleanup = &ownerCleanupState{generation: reasonGeneration, started: time.Now(), target: target, reply: reply}
		publish()
		go owner.runCleanup(reasonGeneration)
	}

	completeValidationLoss := func(generation *ownerGenerationState, reply chan error) {
		ownership = ownerOwnershipUncertain
		snapshot.Health = ownerHealthDegraded
		startCleanup(generation, ownerOwnershipUncertain, nil)
		owner.recordSession(generation.ref, telemetry.SessionOwnershipUncertain, classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed), time.Time{})
		reply <- classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
	}

	finishIfStopped := func() bool {
		cleanupPending := cleanup != nil && cleanup.err == nil
		if !stopping || attempt != nil || validation != nil || generation != nil || cleanupPending || len(workers) != 0 {
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
			if ownership != ownerOwnershipIdle && ownership != ownerOwnershipActive {
				command.reply <- ownerRegistration{err: ownerOwnershipError(ownership)}
				break
			}
			if cleanup != nil && cleanup.err != nil {
				command.reply <- ownerRegistration{err: classifyOperationError(ErrOwnershipUncertain, ToolOutcomeNotSent)}
				break
			}
			nextWorker++
			evidence := ownerWorkerEvidence{owner: owner.id, worker: nextWorker, dispatch: ownerDispatchNotSent}
			worker := &ownerWorkerState{evidence: evidence, ctx: command.ctx, class: command.class, operation: command.operation, reply: make(chan ownerResult, 1)}
			workers[nextWorker] = worker
			queue = append(queue, nextWorker)
			if generation == nil && attempt == nil && cleanup == nil {
				if err := startAttempt(nil, nil); err != nil {
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
			preDispatch := worker != nil && worker.evidence.generation == 0
			if preDispatch {
				removeQueued(command.evidence.worker)
				lastAttemptWaiter := attempt != nil && len(queue) == 0
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
			takeoverReply := attempt.takeoverReply
			takeoverCtx := attempt.takeoverCtx
			attemptStarted := attempt.started
			attempt.cancel()
			attempt = nil
			attemptErr := ownerResultError(command.err, command.panicValue)
			if command.cleanupErr != nil {
				attemptErr = classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
			}
			publish()
			owner.recordSession(command.sessionRef, telemetry.SessionConnectionAttemptCompleted, attemptErr, attemptStarted)
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
				if takeoverReply != nil {
					ownership = ownerOwnershipUncertain
					if command.cleanupErr != nil {
						takeoverReply <- classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
					} else {
						takeoverReply <- ownerResultError(command.err, command.panicValue)
					}
				}
				if command.cleanupErr != nil && command.session != nil {
					generationCtx, cancel := context.WithCancel(context.Background())
					failed := &ownerGenerationState{id: command.generation, ref: command.sessionRef, session: command.session, ctx: generationCtx, cancel: cancel}
					cleanup = &ownerCleanupState{generation: failed, started: time.Now(), target: ownerOwnershipIdle, err: command.cleanupErr}
					ownership = ownerOwnershipUncertain
					publish()
				}
			} else if command.session != nil && !stopping && (len(queue) != 0 || takeoverReply != nil) {
				generationCtx, cancel := context.WithCancel(context.Background())
				generation = &ownerGenerationState{id: command.generation, ref: command.sessionRef, leased: takeoverReply != nil, session: command.session, scheduler: newGenerationScheduler(owner.scheduler, command.session.Done()), ctx: generationCtx, cancel: cancel, watchDone: make(chan struct{})}
				lastSessionRef = generation.ref
				ownership = ownerOwnershipActive
				snapshot.Health = ownerHealthHealthy
				publish()
				owner.recordSession(generation.ref, telemetry.SessionGenerationActive, nil, time.Time{})
				go owner.watchGeneration(generation)
				if takeoverReply != nil {
					telemetry.BindSession(takeoverCtx, generation.ref)
					takeoverReply <- nil
				}
				startNext()
				scheduleIdle()
			} else if command.session != nil {
				if takeoverReply != nil {
					takeoverReply <- classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
				}
				generationCtx, cancel := context.WithCancel(context.Background())
				orphan := &ownerGenerationState{id: command.generation, ref: command.sessionRef, session: command.session, ctx: generationCtx, cancel: cancel}
				target := ownerOwnershipIdle
				if stopping {
					target = ownerOwnershipStopped
				}
				cleanup = &ownerCleanupState{generation: orphan, started: time.Now(), target: target}
				go owner.runCleanup(orphan)
			}
			publish()

		case completeOwnerWorker:
			completion := command.completion
			worker := workers[completion.evidence.worker]
			if worker == nil || worker.evidence.generation == 0 || worker.evidence.owner != completion.evidence.owner || worker.evidence.generation != completion.evidence.generation {
				break
			}
			if worker.cancel != nil {
				worker.cancel()
			}
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
				var validationReply chan error
				sessionRef := generation.ref
				terminalErr := ErrOwnershipUncertain
				event := telemetry.SessionOwnershipUncertain
				ownership = ownerOwnershipUncertain
				if recognizedTakeover(generation.session) {
					terminalErr = ErrSessionTakenOver
					event = telemetry.SessionTakenOver
					ownership = ownerOwnershipTakenOver
				}
				snapshot.Health = ownerHealthDegraded
				if validation != nil && validation.generation == command.generation {
					validationReply = validation.reply
					validation = nil
				}
				failQueued(ownerResult{err: classifyOperationError(terminalErr, ToolOutcomeNotSent)})
				startCleanup(generation, ownership, nil)
				eventErr := error(nil)
				if event == telemetry.SessionOwnershipUncertain {
					eventErr = classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
				}
				owner.recordSession(sessionRef, event, eventErr, time.Time{})
				if validationReply != nil {
					validationReply <- classifyOperationError(terminalErr, ToolOutcomeFailed)
				}
			}

		case ownerTakeoverRecognized:
			pendingTerminalCleanup := cleanup != nil && cleanup.generation.id == command.generation && cleanup.err == nil && cleanup.reply == nil && cleanup.takeoverReply == nil
			completedTerminalCleanup := cleanup == nil && generation == nil && attempt == nil && ownership == ownerOwnershipUncertain && nextGeneration == command.generation
			if !stopping && (pendingTerminalCleanup || completedTerminalCleanup) {
				var sessionRef telemetry.SessionRef
				if pendingTerminalCleanup {
					sessionRef = cleanup.generation.ref
				} else {
					sessionRef = lastSessionRef
				}
				ownership = ownerOwnershipTakenOver
				if pendingTerminalCleanup {
					cleanup.target = ownerOwnershipTakenOver
				}
				snapshot.Health = ownerHealthDegraded
				failQueued(ownerResult{err: classifyOperationError(ErrSessionTakenOver, ToolOutcomeNotSent)})
				publish()
				owner.recordSession(sessionRef, telemetry.SessionTakenOver, nil, time.Time{})
			}

		case ownerIdleExpired:
			idleTimer = nil
			if generation != nil && generation.id == command.generation && len(workers) == 0 && len(queue) == 0 && owner.demand.Load() == 0 {
				startCleanup(generation, ownerOwnershipIdle, nil)
			}

		case ownerDemandChanged:
			scheduleIdle()

		case completeOwnerCleanup:
			if cleanup == nil || cleanup.generation.id != command.generation {
				break
			}
			if command.err != nil {
				wasUncertain := ownership == ownerOwnershipUncertain
				cleanup.err = command.err
				ownership = ownerOwnershipUncertain
				snapshot.Health = ownerHealthDegraded
				failQueued(ownerResult{err: busyNotSent()})
				if cleanup.reply != nil {
					cleanup.reply <- classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
					cleanup.reply = nil
				}
				if cleanup.takeoverReply != nil {
					cleanup.takeoverReply <- classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
					cleanup.takeoverReply = nil
				}
				publish()
				owner.recordCleanupFailure(cleanup.generation.ref, command.err, cleanup.started, !wasUncertain)
			} else {
				target := cleanup.target
				sessionRef := cleanup.generation.ref
				cleanupStarted := cleanup.started
				reply := cleanup.reply
				takeoverReply := cleanup.takeoverReply
				takeoverCtx := cleanup.takeoverCtx
				lateTakeover := false
				if target == ownerOwnershipUncertain && reply == nil && takeoverReply == nil && recognizedTakeover(cleanup.generation.session) {
					target = ownerOwnershipTakenOver
					lateTakeover = true
				}
				keepTakeoverWatch := target == ownerOwnershipUncertain && reply == nil && takeoverReply == nil
				if cleanup.generation.watchDone != nil && !keepTakeoverWatch {
					close(cleanup.generation.watchDone)
				}
				cleanup = nil
				ownership = target
				snapshot.Health = ownerHealthUnavailable
				publish()
				switch target {
				case ownerOwnershipIdle:
					owner.recordSession(sessionRef, telemetry.SessionIdleReleased, nil, cleanupStarted)
				case ownerOwnershipReleased:
					owner.recordSession(sessionRef, telemetry.SessionExplicitlyReleased, nil, cleanupStarted)
				case ownerOwnershipStopped:
					owner.recordSession(sessionRef, telemetry.SessionShutdownClosed, nil, cleanupStarted)
				}
				if lateTakeover {
					owner.recordSession(sessionRef, telemetry.SessionTakenOver, nil, time.Time{})
				}
				if reply != nil {
					reply <- nil
				}
				if takeoverReply != nil {
					if err := startAttempt(takeoverReply, takeoverCtx); err != nil {
						takeoverReply <- err
					}
				}
				if !stopping && len(queue) != 0 {
					if err := startAttempt(nil, nil); err != nil {
						failQueued(ownerResult{err: err})
					}
				}
			}
			publish()

		case releaseOwnerSession:
			switch {
			case stopping:
				command.reply <- classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
			case len(workers) != 0 || len(queue) != 0 || owner.demand.Load() != 0 || attempt != nil:
				command.reply <- busyNotSent()
			case cleanup != nil && cleanup.err == nil:
				command.reply <- busyNotSent()
			case ownership == ownerOwnershipReleased:
				command.reply <- nil
			case generation != nil:
				telemetry.BindSession(command.ctx, generation.ref)
				startCleanup(generation, ownerOwnershipReleased, command.reply)
			case cleanup != nil:
				telemetry.BindSession(command.ctx, cleanup.generation.ref)
				cleanup.target = ownerOwnershipReleased
				cleanup.reply = command.reply
				cleanup.err = nil
				cleanup.started = time.Now()
				go owner.runCleanup(cleanup.generation)
			case ownership == ownerOwnershipIdle || ownership == ownerOwnershipTakenOver || ownership == ownerOwnershipUncertain:
				ownership = ownerOwnershipReleased
				snapshot.Health = ownerHealthUnavailable
				command.reply <- nil
			default:
				command.reply <- busyNotSent()
			}
			publish()

		case takeOverOwnerSession:
			switch {
			case stopping:
				command.reply <- classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
			case len(workers) != 0 || len(queue) != 0 || owner.demand.Load() != 0 || attempt != nil || validation != nil:
				command.reply <- busyNotSent()
			case cleanup != nil && cleanup.err == nil:
				if cleanup.reply == nil && cleanup.takeoverReply == nil {
					cleanup.takeoverReply = command.reply
					cleanup.takeoverCtx = command.ctx
				} else {
					command.reply <- busyNotSent()
				}
			case generation != nil && ownership == ownerOwnershipActive:
				if idleTimer != nil {
					idleTimer.Stop()
					idleTimer = nil
				}
				telemetry.BindSession(command.ctx, generation.ref)
				validation = &ownerValidationState{generation: generation.id, ctx: command.ctx, reply: command.reply}
				go owner.runValidation(generation)
			case cleanup != nil:
				cleanup.takeoverReply = command.reply
				cleanup.takeoverCtx = command.ctx
				cleanup.reply = nil
				cleanup.err = nil
				cleanup.started = time.Now()
				go owner.runCleanup(cleanup.generation)
			case generation == nil:
				if err := startAttempt(command.reply, command.ctx); err != nil {
					command.reply <- err
				}
			default:
				command.reply <- busyNotSent()
			}
			publish()

		case completeOwnerValidation:
			if validation == nil || validation.generation != command.generation {
				break
			}
			reply := validation.reply
			validationCtx := validation.ctx
			validation = nil
			if generation == nil || generation.id != command.generation {
				reply <- classifyOperationError(ErrOwnershipUncertain, ToolOutcomeFailed)
			} else if command.err != nil {
				select {
				case <-generation.session.Done():
					completeValidationLoss(generation, reply)
				default:
					snapshot.Health = ownerHealthDegraded
					reply <- classifyReadFailure(command.err)
				}
			} else {
				select {
				case <-generation.session.Done():
					completeValidationLoss(generation, reply)
				default:
					telemetry.BindSession(validationCtx, generation.ref)
					snapshot.Health = ownerHealthHealthy
					reply <- nil
					scheduleIdle()
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
				for _, active := range workers {
					if active.cancel != nil {
						active.cancel()
					}
				}
				if generation != nil {
					startCleanup(generation, ownerOwnershipStopped, nil)
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
	if err == nil {
		select {
		case <-session.Done():
			err = ErrSessionClosed
		default:
		}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}
	if err == nil && owner.settings.recorder != nil {
		completion.sessionRef, _ = telemetry.NewSessionRef()
	}
	if err != nil && session != nil {
		completion.cleanupErr = owner.closeDetached(session)
		if completion.cleanupErr == nil {
			session = nil
		}
	}
}

func ownerResultError(err error, panicValue any) error {
	if err != nil {
		return err
	}
	if panicValue != nil {
		return classifyOperationError(ErrProtocol, ToolOutcomeNotSent)
	}
	return nil
}

func (owner *deviceOwner) runValidation(generation *ownerGenerationState) {
	ctx, cancel := context.WithTimeout(generation.ctx, owner.settings.connectTimeout)
	defer cancel()
	completion := completeOwnerValidation{generation: generation.id}
	defer func() {
		if recover() != nil {
			completion.err = ErrProtocol
		}
		owner.send(completion)
	}()
	var pong string
	completion.err = generation.session.Call(ctx, methodPing, nil, &pong)
	if completion.err == nil && pong != "pong" {
		completion.err = ErrInvalidResponse
	}
}

func (owner *deviceOwner) runOperation(ctx context.Context, stopGenerationCancel func() bool, evidence ownerWorkerEvidence, session Session, scheduler *generationScheduler, class ownerOperationClass, operation func(context.Context, Session) error) {
	defer stopGenerationCancel()
	completion := ownerCompletion{evidence: evidence}
	defer func() {
		completion.evidence.dispatch = ownerDispatchCompleted
		if panicValue := recover(); panicValue != nil {
			completion.panicValue = panicValue
		}
		owner.send(completeOwnerWorker{completion: completion})
	}()
	completion.err = scheduler.run(ctx, class, session, operation)
	completion.err = classifyTerminalCompletion(session, class, completion.err)
}

func classifyTerminalCompletion(session Session, class ownerOperationClass, err error) error {
	if err == nil {
		return nil
	}
	connected, ok := session.(ConnectedSession)
	if !ok || connected.Done() == nil {
		return err
	}
	select {
	case <-connected.Done():
	default:
		return err
	}
	var classified *OperationError
	if errors.As(err, &classified) && classified.Outcome == ToolOutcomeFailed &&
		!errors.Is(err, ErrSessionClosed) && !errors.Is(err, context.Canceled) {
		return err
	}
	outcome := ToolOutcomeFailed
	if errors.As(err, &classified) && classified.Outcome == ToolOutcomeNotSent {
		outcome = ToolOutcomeNotSent
	} else if class == ownerOperationMutation || class == ownerOperationHID {
		outcome = ToolOutcomeUnknown
	}
	terminalErr := ErrOwnershipUncertain
	if recognizedTakeover(connected) {
		terminalErr = ErrSessionTakenOver
	}
	return classifyOperationError(terminalErr, outcome)
}

func recognizedTakeover(session ConnectedSession) bool {
	takeover, ok := session.(interface{ RecognizedTakeover() bool })
	return ok && takeover.RecognizedTakeover()
}

func takeoverSignal(session ConnectedSession) <-chan struct{} {
	takeover, ok := session.(interface{ TakeoverDetected() <-chan struct{} })
	if !ok {
		return nil
	}
	return takeover.TakeoverDetected()
}

func (owner *deviceOwner) watchGeneration(generation *ownerGenerationState) {
	if generation == nil || generation.session.Done() == nil {
		return
	}
	takeover := takeoverSignal(generation.session)
	select {
	case <-takeover:
		owner.send(ownerGenerationEnded{generation: generation.id})
	case <-generation.session.Done():
		owner.send(ownerGenerationEnded{generation: generation.id})
		if takeover == nil {
			return
		}
		select {
		case <-takeover:
			owner.send(ownerTakeoverRecognized{generation: generation.id})
		case <-generation.watchDone:
		case <-owner.done:
		}
	case <-owner.done:
	}
}

func (owner *deviceOwner) runCleanup(generation *ownerGenerationState) {
	generation.cancel()
	err := owner.closeDetached(generation.session)
	owner.send(completeOwnerCleanup{generation: generation.id, err: err})
}

func ownerOwnershipError(ownership ownerOwnership) error {
	switch ownership {
	case ownerOwnershipReleased:
		return classifyOperationError(ErrSessionReleased, ToolOutcomeNotSent)
	case ownerOwnershipTakenOver:
		return classifyOperationError(ErrSessionTakenOver, ToolOutcomeNotSent)
	case ownerOwnershipUncertain:
		return classifyOperationError(ErrOwnershipUncertain, ToolOutcomeNotSent)
	default:
		return busyNotSent()
	}
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
