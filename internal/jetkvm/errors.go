package jetkvm

import (
	"context"
	"errors"
)

const (
	ToolOutcomeFailed  = "failed"
	ToolOutcomeNotSent = "not_sent"
	ToolOutcomeUnknown = "unknown"
)

type OperationError struct {
	Cause     error
	Code      string
	Outcome   string
	Retryable bool
}

func (err *OperationError) Error() string {
	if err == nil || err.Cause == nil {
		return "JetKVM operation failed"
	}
	return err.Cause.Error()
}

func (err *OperationError) Unwrap() error { return err.Cause }

func (err *OperationError) ToolErrorCode() string { return err.Code }

func (err *OperationError) ToolErrorOutcome() string { return err.Outcome }

func (err *OperationError) ToolErrorRetryable() bool { return err.Retryable }

func classifyOperationError(err error, outcome string) error {
	if err == nil {
		return nil
	}
	code, retryable := "operation_failed", false
	switch {
	case errors.Is(err, ErrUnsupportedInput), errors.Is(err, ErrUnknownDevice), errors.Is(err, ErrUnknownWakeTarget), errors.Is(err, ErrMediaPath), errors.Is(err, ErrMediaDirectoryNotConfigured):
		code = "invalid_input"
	case errors.Is(err, ErrVideoBusy), errors.Is(err, ErrBusy):
		code = "busy"
	case errors.Is(err, ErrSessionReleased):
		code = "session_released"
	case errors.Is(err, ErrSessionTakenOver):
		code = "session_taken_over"
	case errors.Is(err, ErrOwnershipUncertain):
		code = "ownership_uncertain"
	case errors.Is(err, ErrAuthentication):
		code = "authentication_failed"
	case errors.Is(err, ErrDeviceUnreachable), errors.Is(err, ErrSessionClosed):
		code = "device_unavailable"
		retryable = outcome == ToolOutcomeFailed
	case errors.Is(err, ErrDecoderUnavailable):
		code = "video_unavailable"
	case errors.Is(err, ErrNoSignal):
		code = "no_signal"
		retryable = outcome == ToolOutcomeFailed
	case errors.Is(err, ErrProtocol), errors.Is(err, ErrInvalidResponse), errors.Is(err, ErrRPCMethodUnavailable):
		code = "protocol_error"
	case errors.Is(err, context.Canceled):
		code = "canceled"
		retryable = outcome == ToolOutcomeFailed
	case errors.Is(err, context.DeadlineExceeded):
		code = "timeout"
		retryable = outcome == ToolOutcomeFailed
	}
	if outcome != ToolOutcomeFailed {
		retryable = false
	}
	return &OperationError{Cause: err, Code: code, Outcome: outcome, Retryable: retryable}
}

var (
	ErrAuthentication              = errors.New("JetKVM authentication failed")
	ErrDeviceUnreachable           = errors.New("JetKVM device unreachable")
	ErrInvalidResponse             = errors.New("invalid JetKVM response")
	ErrProtocol                    = errors.New("JetKVM protocol error")
	ErrUnknownDevice               = errors.New("unknown JetKVM device")
	ErrUnknownWakeTarget           = errors.New("unknown Wake-on-LAN target")
	ErrRPCMethodUnavailable        = errors.New("JetKVM RPC method unavailable")
	ErrUnsolicitedRPC              = errors.New("unsolicited JetKVM RPC event")
	ErrSessionClosed               = errors.New("JetKVM session closed")
	ErrSessionReleased             = errors.New("JetKVM session was explicitly released")
	ErrSessionTakenOver            = errors.New("JetKVM session was taken over")
	ErrOwnershipUncertain          = errors.New("JetKVM session ownership is uncertain")
	ErrUnsupportedInput            = errors.New("unsupported JetKVM input")
	ErrMediaPath                   = errors.New("invalid media path")
	ErrMediaDirectoryNotConfigured = errors.New("media directory is not configured")
	ErrVideoBusy                   = errors.New("JetKVM video capture already in progress")
	ErrBusy                        = errors.New("JetKVM operation is busy")
	ErrNoSignal                    = errors.New("JetKVM video signal is unavailable")
	ErrDecoderUnavailable          = errors.New("FFmpeg decoder is unavailable")
)
