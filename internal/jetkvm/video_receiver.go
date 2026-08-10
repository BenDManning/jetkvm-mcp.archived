package jetkvm

import (
	"context"
	"sync"
	"time"

	"github.com/pion/rtp"
)

type capturedH264 struct {
	data []byte
	at   time.Time
	err  error
}

type videoWaiter struct {
	watermark uint64
	result    chan capturedH264
}

type videoReceiver struct {
	mu         sync.Mutex
	stream     *h264Stream
	generation uint64
	waiter     *videoWaiter
	requestPLI func() error
	closed     bool
}

func newVideoReceiver() *videoReceiver {
	return &videoReceiver{stream: newH264Stream()}
}

func (receiver *videoReceiver) Capture(ctx context.Context) ([]byte, time.Time, error) {
	if receiver == nil {
		return nil, time.Time{}, ErrSessionClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	receiver.mu.Lock()
	if receiver.closed {
		receiver.mu.Unlock()
		return nil, time.Time{}, ErrSessionClosed
	}
	if receiver.waiter != nil {
		receiver.mu.Unlock()
		return nil, time.Time{}, ErrVideoBusy
	}
	receiver.stream.beginWait()
	waiter := &videoWaiter{watermark: receiver.generation, result: make(chan capturedH264, 1)}
	receiver.waiter = waiter
	requestPLI := receiver.requestPLI
	receiver.mu.Unlock()
	if requestPLI != nil {
		_ = requestPLI()
	}

	select {
	case result := <-waiter.result:
		return result.data, result.at, result.err
	case <-ctx.Done():
		receiver.removeWaiter(waiter)
		return nil, time.Time{}, ctx.Err()
	}
}

func (receiver *videoReceiver) Observe(packet *rtp.Packet, observedAt time.Time) {
	if receiver == nil || packet == nil {
		return
	}
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if receiver.closed {
		return
	}
	receiver.generation++
	if receiver.waiter == nil {
		receiver.stream.drain(packet, receiver.generation, observedAt)
		return
	}
	data, ok := receiver.stream.push(packet, receiver.generation, receiver.waiter.watermark, observedAt)
	if !ok {
		return
	}
	waiter := receiver.waiter
	receiver.waiter = nil
	receiver.stream.endWait()
	capturedAt := receiver.stream.candidateAt.UTC()
	if capturedAt.IsZero() {
		capturedAt = observedAt.UTC()
	}
	waiter.result <- capturedH264{data: append([]byte(nil), data...), at: capturedAt}
}

func (receiver *videoReceiver) SetPLI(request func() error) {
	if receiver == nil {
		return
	}
	receiver.mu.Lock()
	receiver.requestPLI = request
	waiting := receiver.waiter != nil
	receiver.mu.Unlock()
	if waiting && request != nil {
		_ = request()
	}
}

func (receiver *videoReceiver) Waiting() bool {
	if receiver == nil {
		return false
	}
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	return receiver.waiter != nil
}

func (receiver *videoReceiver) Close() {
	if receiver == nil {
		return
	}
	receiver.mu.Lock()
	if receiver.closed {
		receiver.mu.Unlock()
		return
	}
	receiver.closed = true
	waiter := receiver.waiter
	receiver.waiter = nil
	receiver.stream.endWait()
	receiver.mu.Unlock()
	if waiter != nil {
		waiter.result <- capturedH264{err: ErrSessionClosed}
	}
}

func (receiver *videoReceiver) removeWaiter(waiter *videoWaiter) {
	receiver.mu.Lock()
	if receiver.waiter == waiter {
		receiver.waiter = nil
		receiver.stream.endWait()
	}
	receiver.mu.Unlock()
}
