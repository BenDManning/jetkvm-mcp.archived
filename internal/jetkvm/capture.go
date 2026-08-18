package jetkvm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

const maxCapturePNGBytes = 32 << 20

var errCaptureServerDeadline = errors.New("capture server deadline exceeded")

type Decoder interface {
	Decode(ctx context.Context, annexB []byte, maxWidth, maxHeight int) (pngData []byte, width, height int, err error)
}

type ManagerOption func(*Manager) error

func WithDecoder(decoder Decoder) ManagerOption {
	return func(manager *Manager) error {
		if decoder == nil {
			return ErrDecoderUnavailable
		}
		manager.decoder = decoder
		return nil
	}
}

func (manager *Manager) CaptureScreen(ctx context.Context, name string, request mcpserver.CaptureRequest) (mcpserver.CaptureResult, error) {
	return manager.captureScreen(ctx, name, request, mcpserver.CaptureScreenTimeout)
}

func (manager *Manager) captureScreen(ctx context.Context, name string, request mcpserver.CaptureRequest, timeout time.Duration) (_ mcpserver.CaptureResult, returnErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	captureStage := telemetry.BeginStage(ctx, telemetry.StageCapture)
	defer func() { finishTelemetryStage(captureStage, returnErr) }()
	captureCtx, cancel := context.WithTimeoutCause(ctx, timeout, errCaptureServerDeadline)
	defer cancel()
	stopShutdownCancel := context.AfterFunc(manager.shutdownCtx, cancel)
	defer stopShutdownCancel()
	device, err := manager.device(name)
	if err != nil {
		return mcpserver.CaptureResult{}, err
	}
	if manager.decoder == nil {
		return mcpserver.CaptureResult{}, classifyReadFailure(ErrDecoderUnavailable)
	}
	maxWidth, maxHeight := request.MaxWidth, request.MaxHeight
	if maxWidth == 0 {
		maxWidth = 1920
	}
	if maxHeight == 0 {
		maxHeight = 1080
	}
	if maxWidth < 1 || maxWidth > 3840 || maxHeight < 1 || maxHeight > 2160 {
		return mcpserver.CaptureResult{}, classifyOperationError(fmt.Errorf("%w: capture bounds", ErrUnsupportedInput), ToolOutcomeNotSent)
	}
	if err := capturePreDispatchContextError(captureCtx); err != nil {
		return mcpserver.CaptureResult{}, err
	}
	if !manager.beginDecoder() {
		return mcpserver.CaptureResult{}, classifyOperationError(ErrSessionClosed, ToolOutcomeNotSent)
	}
	defer manager.decoderWorkers.Done()
	if !tryAcquire(manager.decoders) {
		return mcpserver.CaptureResult{}, busyNotSent()
	}
	defer release(manager.decoders)
	var annexB []byte
	var capturedAt time.Time
	err = manager.withOperation(captureCtx, device, false, true, func(operationCtx context.Context, session Session) error {
		var state struct {
			Ready *bool `json:"ready"`
		}
		if err := session.Call(operationCtx, methodVideoState, nil, &state); err != nil {
			return err
		}
		if state.Ready != nil && !*state.Ready {
			return ErrNoSignal
		}
		var err error
		annexB, capturedAt, err = session.CaptureH264(operationCtx)
		return err
	})
	if err != nil {
		return mcpserver.CaptureResult{}, classifyCaptureReadFailure(captureCtx, err)
	}
	if err := captureContextError(captureCtx); err != nil {
		return mcpserver.CaptureResult{}, err
	}
	pngData, width, height, err := manager.decoder.Decode(captureCtx, annexB, maxWidth, maxHeight)
	if err != nil {
		return mcpserver.CaptureResult{}, classifyCaptureReadFailure(captureCtx, err)
	}
	if err := captureContextError(captureCtx); err != nil {
		return mcpserver.CaptureResult{}, err
	}
	if len(pngData) == 0 || len(pngData) > maxCapturePNGBytes || width < 1 || width > maxWidth || height < 1 || height > maxHeight || capturedAt.IsZero() {
		return mcpserver.CaptureResult{}, classifyReadFailure(ErrInvalidResponse)
	}
	config, err := png.DecodeConfig(&contextReader{ctx: captureCtx, reader: bytes.NewReader(pngData)})
	if err := captureContextError(captureCtx); err != nil {
		return mcpserver.CaptureResult{}, err
	}
	if err != nil || config.Width != width || config.Height != height {
		return mcpserver.CaptureResult{}, classifyReadFailure(ErrInvalidResponse)
	}
	result := mcpserver.CaptureResult{
		Device: device.Name, CapturedAt: capturedAt.UTC(), MIMEType: "image/png",
		Width: width, Height: height,
	}
	return finalizeCaptureResult(captureCtx, result, pngData)
}

func finalizeCaptureResult(ctx context.Context, result mcpserver.CaptureResult, pngData []byte) (mcpserver.CaptureResult, error) {
	result.PNG = append([]byte(nil), pngData...)
	if err := captureContextError(ctx); err != nil {
		return mcpserver.CaptureResult{}, err
	}
	return result, nil
}

func classifyCaptureReadFailure(captureCtx context.Context, err error) error {
	var operationErr *OperationError
	if errors.As(err, &operationErr) {
		if errors.Is(context.Cause(captureCtx), errCaptureServerDeadline) && errors.Is(err, context.DeadlineExceeded) {
			return classifyOperationError(operationErr.Cause, ToolOutcomeFailed)
		}
		if captureCtx.Err() != nil && operationErr.Outcome == ToolOutcomeNotSent && !errors.Is(err, captureCtx.Err()) {
			if errors.Is(context.Cause(captureCtx), errCaptureServerDeadline) {
				return classifyReadFailure(context.DeadlineExceeded)
			}
			return classifyOperationError(captureCtx.Err(), ToolOutcomeNotSent)
		}
		if captureCtx.Err() != nil && !errors.Is(err, captureCtx.Err()) {
			return captureContextError(captureCtx)
		}
		return err
	}
	if contextErr := captureContextError(captureCtx); contextErr != nil {
		return contextErr
	}
	return classifyReadFailure(err)
}

func captureContextError(captureCtx context.Context) error {
	if captureCtx.Err() == nil {
		return nil
	}
	if errors.Is(context.Cause(captureCtx), errCaptureServerDeadline) {
		return classifyReadFailure(context.DeadlineExceeded)
	}
	return classifyReadFailure(captureCtx.Err())
}

func capturePreDispatchContextError(captureCtx context.Context) error {
	if captureCtx.Err() == nil {
		return nil
	}
	if errors.Is(context.Cause(captureCtx), errCaptureServerDeadline) {
		return classifyReadFailure(context.DeadlineExceeded)
	}
	return classifyOperationError(captureCtx.Err(), ToolOutcomeNotSent)
}
