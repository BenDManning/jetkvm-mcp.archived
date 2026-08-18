package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const toolErrorVersion = "1"

type toolErrorCode string

type toolOutcome string

const (
	toolErrorOperationFailed toolErrorCode = "operation_failed"
	toolErrorCanceled        toolErrorCode = "canceled"
	toolErrorTimeout         toolErrorCode = "timeout"

	toolOutcomeFailed  toolOutcome = "failed"
	toolOutcomeUnknown toolOutcome = "unknown"
)

type toolError struct {
	Version   string        `json:"version"`
	Code      toolErrorCode `json:"code"`
	Message   string        `json:"message"`
	Outcome   toolOutcome   `json:"outcome"`
	Retryable bool          `json:"retryable"`
}

type classifiedToolError interface {
	error
	ToolErrorCode() string
	ToolErrorOutcome() string
	ToolErrorRetryable() bool
}

type localToolError struct {
	error
	code    string
	outcome string
}

func (err localToolError) ToolErrorCode() string { return err.code }

func (err localToolError) ToolErrorOutcome() string { return err.outcome }

func (localToolError) ToolErrorRetryable() bool { return false }

func invalidInput(err error) error {
	if err == nil {
		err = errors.New("invalid input")
	}
	return localToolError{error: fmt.Errorf("invalid input: %w", err), code: "invalid_input", outcome: "not_sent"}
}

func toolFailure(err error, mutation bool) error {
	failure := toolError{
		Version:   toolErrorVersion,
		Code:      toolErrorOperationFailed,
		Message:   "JetKVM operation failed",
		Outcome:   toolOutcomeFailed,
		Retryable: false,
	}
	var classified classifiedToolError
	if errors.As(err, &classified) {
		code := toolErrorCode(classified.ToolErrorCode())
		outcome := toolOutcome(classified.ToolErrorOutcome())
		if validToolErrorCode(code) && validToolOutcome(outcome) {
			failure.Code = code
			failure.Outcome = outcome
			failure.Retryable = classified.ToolErrorRetryable()
		} else if mutation {
			failure.Outcome = toolOutcomeUnknown
		}
	} else if errors.Is(err, context.Canceled) {
		failure.Code = toolErrorCanceled
		failure.Message = "JetKVM operation was canceled"
		failure.Retryable = !mutation
	} else if errors.Is(err, context.DeadlineExceeded) {
		failure.Code = toolErrorTimeout
		failure.Message = "JetKVM operation timed out"
		failure.Retryable = !mutation
	}
	if mutation {
		if !errors.As(err, &classified) {
			failure.Outcome = toolOutcomeUnknown
		}
		failure.Retryable = false
	} else if failure.Outcome != "not_sent" && (failure.Code == toolErrorTimeout || failure.Code == toolErrorCanceled || failure.Code == "device_unavailable" || failure.Code == "no_signal") {
		failure.Outcome = toolOutcomeFailed
		failure.Retryable = true
	}
	failure.Message = toolErrorMessage(failure.Code)
	return failure
}

func validToolErrorCode(code toolErrorCode) bool {
	switch code {
	case toolErrorOperationFailed, toolErrorCanceled, toolErrorTimeout, "invalid_input", "busy", "authentication_failed", "device_unavailable", "video_unavailable", "no_signal", "protocol_error", "session_released", "session_taken_over", "ownership_uncertain":
		return true
	default:
		return false
	}
}

func validToolOutcome(outcome toolOutcome) bool {
	return outcome == toolOutcomeFailed || outcome == toolOutcomeUnknown || outcome == "not_sent"
}

func toolErrorMessage(code toolErrorCode) string {
	switch code {
	case toolErrorCanceled:
		return "JetKVM operation was canceled"
	case toolErrorTimeout:
		return "JetKVM operation timed out"
	case "invalid_input":
		return "JetKVM input is invalid"
	case "busy":
		return "JetKVM operation is busy"
	case "authentication_failed":
		return "JetKVM authentication failed"
	case "device_unavailable":
		return "JetKVM device is unavailable"
	case "video_unavailable":
		return "JetKVM video is unavailable"
	case "no_signal":
		return "JetKVM video signal is unavailable"
	case "protocol_error":
		return "JetKVM protocol operation failed"
	case "session_released":
		return "JetKVM session was explicitly released"
	case "session_taken_over":
		return "JetKVM session was taken over"
	case "ownership_uncertain":
		return "JetKVM session ownership is uncertain"
	default:
		return "JetKVM operation failed"
	}
}

func (failure toolError) Error() string {
	encoded, err := json.Marshal(failure)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
