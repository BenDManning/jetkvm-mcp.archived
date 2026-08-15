package jetkvm

import (
	"context"
	"errors"

	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

type telemetryClassifiedError interface {
	error
	ToolErrorCode() string
	ToolErrorOutcome() string
}

func finishTelemetryStage(stage *telemetry.StageSpan, err error) {
	if err == nil {
		stage.Record(telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
		return
	}
	code, outcome := "operation_failed", telemetry.OutcomeFailed
	var classified telemetryClassifiedError
	if errors.As(err, &classified) {
		code = classified.ToolErrorCode()
		outcome = classified.ToolErrorOutcome()
	} else if errors.Is(err, context.Canceled) {
		code = "canceled"
	} else if errors.Is(err, context.DeadlineExceeded) {
		code = "timeout"
	}
	stage.Record(code, outcome)
}
