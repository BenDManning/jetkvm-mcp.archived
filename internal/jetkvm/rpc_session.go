package jetkvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const maxPendingRPCRequests = 64

type textSender interface {
	SendText(string) error
}

type rpcOutcome struct {
	result json.RawMessage
	err    error
}

type rpcSession struct {
	ctx     context.Context
	cancel  context.CancelFunc
	sender  textSender
	timeout time.Duration
	nextID  atomic.Uint64

	closeOnce sync.Once
	sendGate  chan struct{}
	mu        sync.Mutex
	pending   map[uint64]chan rpcOutcome
}

func newRPCSession(parent context.Context, sender textSender, timeout time.Duration) *rpcSession {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithCancel(parent)
	session := &rpcSession{
		ctx: ctx, cancel: cancel, sender: sender, timeout: timeout,
		sendGate: make(chan struct{}, 1), pending: make(map[uint64]chan rpcOutcome),
	}
	session.sendGate <- struct{}{}
	return session
}

func (session *rpcSession) Call(ctx context.Context, method string, params any, result any) (returnErr error) {
	rpcStage := startStage(ctx, StageRPC)
	defer func() { rpcStage.Finish(returnErr) }()
	if session == nil || session.sender == nil {
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	}
	if strings.TrimSpace(method) == "" {
		return classifyOperationError(errors.New("RPC method is required"), ToolOutcomeNotSent)
	}
	requestCtx, cancel := context.WithTimeout(ctx, session.timeout)
	defer cancel()
	id := session.nextID.Add(1)
	waiter := make(chan rpcOutcome, 1)
	if err := session.addPending(id, waiter); err != nil {
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	remove := func() { session.removePending(id) }

	payload, err := marshalRPCRequest(id, method, params)
	if err != nil {
		remove()
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	select {
	case <-requestCtx.Done():
		remove()
		return classifyOperationError(requestCtx.Err(), ToolOutcomeNotSent)
	case <-session.ctx.Done():
		remove()
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	case <-session.sendGate:
	}
	if requestCtx.Err() != nil || session.ctx.Err() != nil {
		session.sendGate <- struct{}{}
		remove()
		if requestCtx.Err() != nil {
			return classifyOperationError(requestCtx.Err(), ToolOutcomeNotSent)
		}
		return classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	}
	err = session.sender.SendText(payload)
	session.sendGate <- struct{}{}
	if err != nil {
		remove()
		return classifyOperationError(fmt.Errorf("%w: RPC send", ErrDeviceUnreachable), ToolOutcomeUnknown)
	}

	select {
	case outcome := <-waiter:
		if outcome.err != nil {
			var rpcErr *rpcProtocolError
			if errors.Is(outcome.err, ErrRPCMethodUnavailable) || errors.As(outcome.err, &rpcErr) {
				return classifyOperationError(outcome.err, ToolOutcomeFailed)
			}
			return classifyOperationError(outcome.err, ToolOutcomeUnknown)
		}
		if result == nil {
			return nil
		}
		if len(outcome.result) == 0 {
			return classifyOperationError(ErrInvalidResponse, ToolOutcomeUnknown)
		}
		if err := json.Unmarshal(outcome.result, result); err != nil {
			return classifyOperationError(ErrInvalidResponse, ToolOutcomeUnknown)
		}
		return nil
	case <-requestCtx.Done():
		remove()
		return classifyOperationError(requestCtx.Err(), ToolOutcomeUnknown)
	case <-session.ctx.Done():
		remove()
		return classifyOperationError(ErrSessionClosed, ToolOutcomeUnknown)
	}
}

func (session *rpcSession) HandleMessage(data []byte) {
	if session == nil {
		return
	}
	response, err := decodeRPCResponse(data)
	if response.ID == 0 {
		return
	}
	session.mu.Lock()
	waiter := session.pending[response.ID]
	delete(session.pending, response.ID)
	session.mu.Unlock()
	if waiter == nil {
		return
	}
	select {
	case waiter <- rpcOutcome{result: response.Result, err: err}:
	default:
	}
}

func (session *rpcSession) Close() {
	if session == nil {
		return
	}
	session.closeOnce.Do(func() {
		session.cancel()
		session.mu.Lock()
		pending := session.pending
		session.pending = make(map[uint64]chan rpcOutcome)
		session.mu.Unlock()
		for _, waiter := range pending {
			select {
			case waiter <- rpcOutcome{err: ErrSessionClosed}:
			default:
			}
		}
	})
}

func (session *rpcSession) addPending(id uint64, waiter chan rpcOutcome) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.ctx.Err() != nil {
		return ErrSessionClosed
	}
	if len(session.pending) >= maxPendingRPCRequests {
		return ErrBusy
	}
	session.pending[id] = waiter
	return nil
}

func (session *rpcSession) removePending(id uint64) {
	session.mu.Lock()
	delete(session.pending, id)
	session.mu.Unlock()
}

func (session *rpcSession) pendingCount() int {
	session.mu.Lock()
	defer session.mu.Unlock()
	return len(session.pending)
}
