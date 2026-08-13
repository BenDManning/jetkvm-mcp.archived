package jetkvm

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

const maxCapturePNGBytes = 32 << 20

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
	var annexB []byte
	var capturedAt time.Time
	err = manager.withSession(ctx, device, SessionProfileVideo, func(session Session) error {
		var state struct {
			Ready *bool `json:"ready"`
		}
		if err := session.Call(ctx, methodVideoState, nil, &state); err != nil {
			return err
		}
		if state.Ready != nil && !*state.Ready {
			return ErrNoSignal
		}
		var err error
		annexB, capturedAt, err = session.CaptureH264(ctx)
		return err
	})
	if err != nil {
		return mcpserver.CaptureResult{}, classifyReadFailure(err)
	}
	pngData, width, height, err := manager.decoder.Decode(ctx, annexB, maxWidth, maxHeight)
	if err != nil {
		return mcpserver.CaptureResult{}, classifyReadFailure(err)
	}
	if len(pngData) == 0 || len(pngData) > maxCapturePNGBytes || width < 1 || width > maxWidth || height < 1 || height > maxHeight || capturedAt.IsZero() {
		return mcpserver.CaptureResult{}, classifyReadFailure(ErrInvalidResponse)
	}
	config, err := png.DecodeConfig(bytes.NewReader(pngData))
	if err != nil || config.Width != width || config.Height != height {
		return mcpserver.CaptureResult{}, classifyReadFailure(ErrInvalidResponse)
	}
	return mcpserver.CaptureResult{
		Device: device.Name, CapturedAt: capturedAt.UTC(), MIMEType: "image/png",
		Width: width, Height: height, PNG: append([]byte(nil), pngData...),
	}, nil
}
