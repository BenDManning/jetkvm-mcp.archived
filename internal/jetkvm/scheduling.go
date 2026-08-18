package jetkvm

import (
	"context"
	"io"
	"sync"
	"time"
)

type ownerOperationClass uint8

const (
	ownerOperationOrdinary ownerOperationClass = iota
	ownerOperationMutation
	ownerOperationHID
)

type capabilitySchedule uint8

const (
	capabilitySchedulePrecise capabilitySchedule = iota
	capabilityScheduleSessionWide
)

type capabilityProfile struct {
	revision uint64
	schedule capabilitySchedule
}

var initialCapabilityProfile = capabilityProfile{revision: 1, schedule: capabilitySchedulePrecise}

type gateWaiter struct{ ready chan struct{} }

// fairGate is a cancellable FIFO gate. The queue itself is bounded by the
// Manager's global and per-device operation admission.
type fairGate struct {
	mu      sync.Mutex
	held    bool
	waiters []*gateWaiter
}

func (gate *fairGate) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	gate.mu.Lock()
	if !gate.held && len(gate.waiters) == 0 {
		gate.held = true
		gate.mu.Unlock()
		return nil
	}
	waiter := &gateWaiter{ready: make(chan struct{})}
	gate.waiters = append(gate.waiters, waiter)
	gate.mu.Unlock()

	select {
	case <-waiter.ready:
		if err := ctx.Err(); err != nil {
			gate.release()
			return err
		}
		return nil
	case <-ctx.Done():
		gate.mu.Lock()
		for index, candidate := range gate.waiters {
			if candidate == waiter {
				gate.waiters = append(gate.waiters[:index], gate.waiters[index+1:]...)
				gate.mu.Unlock()
				return ctx.Err()
			}
		}
		gate.mu.Unlock()
		// Ownership was handed to this waiter concurrently with cancellation.
		<-waiter.ready
		gate.release()
		return ctx.Err()
	}
}

func (gate *fairGate) release() {
	gate.mu.Lock()
	if len(gate.waiters) == 0 {
		gate.held = false
		gate.mu.Unlock()
		return
	}
	waiter := gate.waiters[0]
	gate.waiters = gate.waiters[1:]
	gate.mu.Unlock()
	close(waiter.ready)
}

type ownerScheduler struct {
	profile  capabilityProfile
	mutation fairGate
}

type generationScheduler struct {
	owner   *ownerScheduler
	done    <-chan struct{}
	rpc     fairGate
	capture fairGate
	session fairGate
}

type hidPressed uint8

const (
	hidPressedNone hidPressed = iota
	hidPressedKeyboard
	hidPressedMouse
)

type scheduledSession struct {
	Session
	scheduler  *generationScheduler
	rpcHeld    bool
	wholeHeld  bool
	hidPressed hidPressed
}

func newOwnerScheduler(profile capabilityProfile) *ownerScheduler {
	return &ownerScheduler{profile: profile}
}

func newGenerationScheduler(owner *ownerScheduler, done <-chan struct{}) *generationScheduler {
	return &generationScheduler{owner: owner, done: done}
}

func (session *scheduledSession) GenerationEnded() bool {
	if session.scheduler.done == nil {
		return false
	}
	select {
	case <-session.scheduler.done:
		return true
	default:
		return false
	}
}

func (scheduler *generationScheduler) run(ctx context.Context, class ownerOperationClass, session Session, operation func(context.Context, Session) error) error {
	scheduled := &scheduledSession{Session: session, scheduler: scheduler}
	if scheduler.owner.profile.schedule == capabilityScheduleSessionWide {
		if err := scheduler.session.acquire(ctx); err != nil {
			return classifyOperationError(err, ToolOutcomeNotSent)
		}
		defer scheduler.session.release()
		scheduled.wholeHeld = true
	}
	if class == ownerOperationHID {
		if err := scheduler.rpc.acquire(ctx); err != nil {
			return classifyOperationError(err, ToolOutcomeNotSent)
		}
		defer scheduler.rpc.release()
		scheduled.rpcHeld = true
		defer scheduled.finishHID()
	}
	return operation(ctx, scheduled)
}

func (session *scheduledSession) Call(ctx context.Context, method string, params any, result any) error {
	if session.rpcHeld || session.wholeHeld {
		return session.Session.Call(ctx, method, params, result)
	}
	if err := session.scheduler.rpc.acquire(ctx); err != nil {
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	defer session.scheduler.rpc.release()
	return session.Session.Call(ctx, method, params, result)
}

func (session *scheduledSession) Upload(ctx context.Context, uploadID string, reader io.Reader, size int64) error {
	return session.Session.Upload(ctx, uploadID, reader, size)
}

func (session *scheduledSession) CaptureH264(ctx context.Context) ([]byte, time.Time, error) {
	if err := session.scheduler.capture.acquire(ctx); err != nil {
		return nil, time.Time{}, classifyOperationError(err, ToolOutcomeNotSent)
	}
	defer session.scheduler.capture.release()
	frame, capturedAt, err := session.Session.CaptureH264(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(frame) == 0 || len(frame) > maxAnnexBBytes || capturedAt.IsZero() {
		return nil, time.Time{}, ErrInvalidResponse
	}
	return append([]byte(nil), frame...), capturedAt, nil
}

type hidSafetyController interface {
	armHIDNeutralization(hidPressed)
	acknowledgeHIDNeutralization()
}

type hidCleanupSuppressor interface {
	SuppressHIDCleanup() bool
}

func armHIDNeutralization(session Session, pressed hidPressed) {
	if controller, ok := session.(hidSafetyController); ok {
		controller.armHIDNeutralization(pressed)
	}
}

func acknowledgeHIDNeutralization(session Session) {
	if controller, ok := session.(hidSafetyController); ok {
		controller.acknowledgeHIDNeutralization()
	}
}

func (session *scheduledSession) armHIDNeutralization(pressed hidPressed) {
	session.hidPressed = pressed
}

func (session *scheduledSession) acknowledgeHIDNeutralization() {
	session.hidPressed = hidPressedNone
}

func (session *scheduledSession) finishHID() {
	if suppressor, ok := session.Session.(hidCleanupSuppressor); ok && suppressor.SuppressHIDCleanup() {
		session.hidPressed = hidPressedNone
		return
	}
	switch session.hidPressed {
	case hidPressedKeyboard:
		bestEffortKeyboardRelease(session.Session)
	case hidPressedMouse:
		bestEffortMouseRelease(session.Session)
	}
	session.hidPressed = hidPressedNone
}
