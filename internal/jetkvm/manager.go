package jetkvm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
)

const (
	methodPing              = "ping"
	methodLocalVersion      = "getLocalVersion"
	methodActiveExtension   = "getActiveExtension"
	methodVirtualMediaState = "getVirtualMediaState"
	methodVideoState        = "getVideoState"
	methodUSBState          = "getUSBState"
	methodATXState          = "getATXState"
	methodDCPowerState      = "getDCPowerState"
)

const (
	defaultMaxOperations          = 16
	defaultMaxOperationsPerDevice = 4
	defaultMaxSessions            = 8
	defaultMaxCaptures            = 2
	defaultMaxDecoders            = 2
	maxAdmissionLimit             = 1024
)

// Limits bounds in-flight work. Every value must be positive and no greater
// than 1024. Omitted values use conservative defaults.
type Limits struct {
	MaxOperations          int
	MaxOperationsPerDevice int
	MaxSessions            int
	MaxCaptures            int
	MaxDecoders            int
}

func DefaultLimits() Limits {
	return Limits{
		MaxOperations: defaultMaxOperations, MaxOperationsPerDevice: defaultMaxOperationsPerDevice,
		MaxSessions: defaultMaxSessions, MaxCaptures: defaultMaxCaptures, MaxDecoders: defaultMaxDecoders,
	}
}

func (limits Limits) normalized() (Limits, error) {
	defaults := DefaultLimits()
	if limits.MaxOperations == 0 {
		limits.MaxOperations = defaults.MaxOperations
	}
	if limits.MaxOperationsPerDevice == 0 {
		limits.MaxOperationsPerDevice = defaults.MaxOperationsPerDevice
	}
	if limits.MaxSessions == 0 {
		limits.MaxSessions = defaults.MaxSessions
	}
	if limits.MaxCaptures == 0 {
		limits.MaxCaptures = defaults.MaxCaptures
	}
	if limits.MaxDecoders == 0 {
		limits.MaxDecoders = defaults.MaxDecoders
	}
	for _, value := range []int{limits.MaxOperations, limits.MaxOperationsPerDevice, limits.MaxSessions, limits.MaxCaptures, limits.MaxDecoders} {
		if value < 1 || value > maxAdmissionLimit {
			return Limits{}, errors.New("admission limits must be between 1 and 1024")
		}
	}
	if limits.MaxOperationsPerDevice > limits.MaxOperations || limits.MaxSessions > limits.MaxOperations || limits.MaxCaptures > limits.MaxSessions {
		return Limits{}, errors.New("admission limits exceed their parent capacity")
	}
	return limits, nil
}

// ValidateLimits rejects unsafe capacity combinations without changing values.
func ValidateLimits(limits Limits) (Limits, error) {
	for _, value := range []int{limits.MaxOperations, limits.MaxOperationsPerDevice, limits.MaxSessions, limits.MaxCaptures, limits.MaxDecoders} {
		if value < 1 || value > maxAdmissionLimit {
			return Limits{}, errors.New("admission limits must be between 1 and 1024")
		}
	}
	if limits.MaxOperationsPerDevice > limits.MaxOperations || limits.MaxSessions > limits.MaxOperations || limits.MaxCaptures > limits.MaxSessions {
		return Limits{}, errors.New("admission limits exceed their parent capacity")
	}
	return limits, nil
}

type WakeOnLANTarget struct {
	MACAddress  string
	BroadcastIP string
}

type DeviceConfig struct {
	Name                   string
	BaseURL                url.URL
	Password               string
	InsecureSkipVerify     bool
	MediaDirectory         string
	MediaURLAllowedOrigins []string
	WakeOnLAN              map[string]WakeOnLANTarget
}

type Session interface {
	Call(ctx context.Context, method string, params any, result any) error
	Upload(ctx context.Context, uploadID string, reader io.Reader, size int64) error
	CaptureH264(ctx context.Context) ([]byte, time.Time, error)
}

type SessionProfile uint8

const (
	SessionProfileData SessionProfile = iota
	SessionProfileVideo
)

type SessionProvider interface {
	WithSession(ctx context.Context, device DeviceConfig, profile SessionProfile, operation func(Session) error) error
}

type Manager struct {
	devices    map[string]DeviceConfig
	provider   SessionProvider
	decoder    Decoder
	operations chan struct{}
	deviceOps  map[string]chan struct{}
	sessions   chan struct{}
	captures   chan struct{}
	decoders   chan struct{}
	mutations  map[string]chan struct{}
}

func NewManager(devices []DeviceConfig, provider SessionProvider, options ...ManagerOption) (*Manager, error) {
	if provider == nil {
		return nil, errors.New("session provider is required")
	}
	manager := &Manager{
		devices:   make(map[string]DeviceConfig, len(devices)),
		provider:  provider,
		deviceOps: make(map[string]chan struct{}, len(devices)),
		mutations: make(map[string]chan struct{}, len(devices)),
	}
	for _, candidate := range devices {
		name := strings.TrimSpace(candidate.Name)
		if name == "" || candidate.BaseURL.Host == "" || candidate.BaseURL.Scheme != "http" && candidate.BaseURL.Scheme != "https" {
			return nil, errors.New("device name and HTTP(S) URL are required")
		}
		if _, exists := manager.devices[name]; exists {
			return nil, fmt.Errorf("duplicate device %q", name)
		}
		candidate.Name = name
		if candidate.BaseURL.RawQuery != "" || candidate.BaseURL.ForceQuery || candidate.BaseURL.Fragment != "" {
			return nil, fmt.Errorf("device %q URL must not include a query or fragment", name)
		}
		mediaOrigins, err := normalizeMediaURLAllowedOrigins(candidate.MediaURLAllowedOrigins)
		if err != nil {
			return nil, fmt.Errorf("device %q media URL allowed origins are invalid", name)
		}
		candidate.MediaURLAllowedOrigins = mediaOrigins
		for targetName, target := range candidate.WakeOnLAN {
			if strings.TrimSpace(targetName) == "" {
				return nil, errors.New("Wake-on-LAN target name is required")
			}
			if _, err := net.ParseMAC(target.MACAddress); err != nil {
				return nil, fmt.Errorf("Wake-on-LAN target %q has invalid MAC address", targetName)
			}
			if target.BroadcastIP != "" {
				broadcast := net.ParseIP(target.BroadcastIP)
				if broadcast == nil || broadcast.To4() == nil {
					return nil, fmt.Errorf("Wake-on-LAN target %q has invalid IPv4 broadcast IP", targetName)
				}
			}
		}
		manager.devices[name] = candidate
	}
	if len(manager.devices) == 0 {
		return nil, errors.New("at least one device is required")
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("manager option is required")
		}
		if err := option(manager); err != nil {
			return nil, err
		}
	}
	if manager.operations == nil {
		if err := WithLimits(DefaultLimits())(manager); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

// WithLimits configures bounded, process-wide admission. Capacity exhaustion
// rejects immediately; mutation sequencing waits only while the caller context
// remains active.
func WithLimits(limits Limits) ManagerOption {
	return func(manager *Manager) error {
		validated, err := limits.normalized()
		if err != nil {
			return err
		}
		manager.operations = make(chan struct{}, validated.MaxOperations)
		manager.sessions = make(chan struct{}, validated.MaxSessions)
		manager.captures = make(chan struct{}, validated.MaxCaptures)
		manager.decoders = make(chan struct{}, validated.MaxDecoders)
		for name := range manager.devices {
			manager.deviceOps[name] = make(chan struct{}, validated.MaxOperationsPerDevice)
			manager.mutations[name] = make(chan struct{}, 1)
		}
		return nil
	}
}

func (manager *Manager) ListDevices(context.Context) (mcpserver.DeviceList, error) {
	names := make([]string, 0, len(manager.devices))
	for name := range manager.devices {
		names = append(names, name)
	}
	sort.Strings(names)
	devices := make([]mcpserver.ConfiguredDevice, 0, len(names))
	for _, name := range names {
		device := manager.devices[name]
		hasMediaDirectory := device.MediaDirectory != ""
		devices = append(devices, mcpserver.ConfiguredDevice{
			Device: name,
			Capabilities: mcpserver.DeviceCapabilities{
				MountVirtualMediaURL:   len(device.MediaURLAllowedOrigins) != 0,
				MountVirtualMediaFile:  hasMediaDirectory,
				UploadVirtualMediaFile: hasMediaDirectory,
				WakeHostLAN:            len(device.WakeOnLAN) != 0,
			},
		})
	}
	return mcpserver.DeviceList{Devices: devices}, nil
}

func (manager *Manager) Status(ctx context.Context, name string) (mcpserver.Status, error) {
	device, err := manager.device(name)
	if err != nil {
		return mcpserver.Status{}, err
	}
	status := mcpserver.Status{Device: device.Name}
	err = manager.withOperation(ctx, device, false, false, func() error {
		return manager.withSession(ctx, device, SessionProfileData, func(session Session) error {
			var pong string
			if err := session.Call(ctx, methodPing, nil, &pong); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if pong != "pong" {
				return fmt.Errorf("%w: ping result", ErrInvalidResponse)
			}
			status.Connected = true

			var version struct {
				Application string `json:"appVersion"`
				System      string `json:"systemVersion"`
			}
			probeErr := session.Call(ctx, methodLocalVersion, nil, &version)
			if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningVersionUnavailable); err != nil {
				return err
			}
			if probeErr == nil {
				status.Application, status.System = version.Application, version.System
			}

			probeErr = session.Call(ctx, methodActiveExtension, nil, &status.Extension)
			if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningActiveExtensionUnavailable); err != nil {
				return err
			}

			var media *firmwareVirtualMediaState
			probeErr = session.Call(ctx, methodVirtualMediaState, nil, &media)
			if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningVirtualMediaUnavailable); err != nil {
				return err
			}
			if probeErr == nil {
				if projected, projectErr := publicVirtualMediaState(media); projectErr != nil {
					status.Warnings = append(status.Warnings, mcpserver.StatusWarningVirtualMediaUnavailable)
				} else {
					status.VirtualMedia = projected
				}
			}

			var video struct {
				Ready  *bool `json:"ready"`
				Width  int   `json:"width"`
				Height int   `json:"height"`
				FPS    int   `json:"fps"`
			}
			probeErr = session.Call(ctx, methodVideoState, nil, &video)
			if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningVideoUnavailable); err != nil {
				return err
			}
			if probeErr == nil {
				status.VideoReady, status.VideoWidth, status.VideoHeight, status.VideoFPS = video.Ready, video.Width, video.Height, video.FPS
			}

			probeErr = session.Call(ctx, methodUSBState, nil, &status.USBState)
			if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningUSBUnavailable); err != nil {
				return err
			}
			if probeErr == nil {
				attached := status.USBState != "not attached" && status.USBState != ""
				status.USBWakeAttached = &attached
			}

			switch status.Extension {
			case "atx-power":
				var atx struct {
					Power *bool `json:"power"`
				}
				probeErr = session.Call(ctx, methodATXState, nil, &atx)
				if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningATXUnavailable); err != nil {
					return err
				}
				if probeErr == nil {
					status.ATXPowerOn = atx.Power
				}
			case "dc-power":
				var dc struct {
					On      *bool   `json:"isOn"`
					Voltage float64 `json:"voltage"`
				}
				probeErr = session.Call(ctx, methodDCPowerState, nil, &dc)
				if err := statusProbeResult(ctx, &status, probeErr, mcpserver.StatusWarningDCUnavailable); err != nil {
					return err
				}
				if probeErr == nil {
					status.DCPowerOn, status.DCVoltage = dc.On, dc.Voltage
				}
			}
			return nil
		})
	})
	if err != nil {
		return mcpserver.Status{}, classifyReadFailure(err)
	}
	return status, nil
}

func statusProbeResult(ctx context.Context, status *mcpserver.Status, probeErr error, warning mcpserver.StatusWarning) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		if probeErr != nil && errors.Is(probeErr, ctxErr) {
			return probeErr
		}
		return ctxErr
	}
	if probeErr != nil {
		status.Warnings = append(status.Warnings, warning)
	}
	return nil
}

func (manager *Manager) Power(ctx context.Context, name string, action mcpserver.PowerAction, targetName string) (mcpserver.PowerResult, error) {
	device, err := manager.device(name)
	if err != nil {
		return mcpserver.PowerResult{}, err
	}
	method, params, err := powerRequest(device, action, targetName)
	if err != nil {
		return mcpserver.PowerResult{}, err
	}
	if err := manager.withOperation(ctx, device, true, false, func() error {
		return manager.withSession(ctx, device, SessionProfileData, func(session Session) error {
			return session.Call(ctx, method, params, nil)
		})
	}); err != nil {
		return mcpserver.PowerResult{}, err
	}
	return mcpserver.PowerResult{Device: device.Name, Action: action, Target: targetName, Status: mcpserver.ResultStatusCompleted}, nil
}

func (manager *Manager) withSession(ctx context.Context, device DeviceConfig, profile SessionProfile, operation func(Session) error) error {
	admission := telemetry.BeginStage(ctx, telemetry.StageAdmission)
	if !tryAcquire(manager.sessions) {
		err := busyNotSent()
		finishTelemetryStage(admission, err)
		return err
	}
	finishTelemetryStage(admission, nil)
	defer release(manager.sessions)
	invoked := false
	err := manager.provider.WithSession(ctx, device, profile, func(session Session) error {
		invoked = true
		return operation(session)
	})
	if err != nil && !invoked {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return classifyOperationError(ctxErr, ToolOutcomeNotSent)
		}
		var classified interface{ ToolErrorOutcome() string }
		if errors.As(err, &classified) {
			return err
		}
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	return err
}

func (manager *Manager) withOperation(ctx context.Context, device DeviceConfig, mutation, capture bool, operation func() error) error {
	if !tryAcquire(manager.operations) {
		return busyNotSent()
	}
	if !tryAcquire(manager.deviceOps[device.Name]) {
		release(manager.operations)
		return busyNotSent()
	}
	return manager.runOperation(ctx, device, mutation, capture, operation)
}

func (manager *Manager) runOperation(ctx context.Context, device DeviceConfig, mutation, capture bool, operation func() error) error {
	defer release(manager.operations)
	defer release(manager.deviceOps[device.Name])
	if capture {
		if !tryAcquire(manager.captures) {
			return busyNotSent()
		}
		defer release(manager.captures)
	}
	if mutation {
		if err := acquireContext(ctx, manager.mutations[device.Name]); err != nil {
			return classifyOperationError(err, ToolOutcomeNotSent)
		}
		defer release(manager.mutations[device.Name])
	}
	return operation()
}

func tryAcquire(permits chan struct{}) bool {
	select {
	case permits <- struct{}{}:
		return true
	default:
		return false
	}
}

func acquireContext(ctx context.Context, permits chan struct{}) error {
	select {
	case permits <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func release(permits chan struct{}) { <-permits }

func busyNotSent() error { return classifyOperationError(ErrBusy, ToolOutcomeNotSent) }

func classifyReadFailure(err error) error {
	var classified *OperationError
	if errors.As(err, &classified) {
		return err
	}
	return classifyOperationError(err, ToolOutcomeFailed)
}

func (manager *Manager) device(name string) (DeviceConfig, error) {
	device, exists := manager.devices[strings.TrimSpace(name)]
	if !exists {
		return DeviceConfig{}, classifyOperationError(fmt.Errorf("%w: %s", ErrUnknownDevice, strings.TrimSpace(name)), ToolOutcomeNotSent)
	}
	return device, nil
}

func powerRequest(device DeviceConfig, action mcpserver.PowerAction, targetName string) (string, any, error) {
	switch action {
	case mcpserver.PowerActionPressHostPowerButton:
		return "setATXPowerAction", map[string]any{"action": "power-short"}, nil
	case mcpserver.PowerActionForceHostPowerOff:
		return "setATXPowerAction", map[string]any{"action": "power-long"}, nil
	case mcpserver.PowerActionPressHostResetButton:
		return "setATXPowerAction", map[string]any{"action": "reset"}, nil
	case mcpserver.PowerActionTurnHostDCPowerOn:
		return "setDCPowerState", map[string]any{"enabled": true}, nil
	case mcpserver.PowerActionTurnHostDCPowerOff:
		return "setDCPowerState", map[string]any{"enabled": false}, nil
	case mcpserver.PowerActionWakeHostUSB:
		return "wakeHost", nil, nil
	case mcpserver.PowerActionWakeHostLAN:
		target, exists := device.WakeOnLAN[strings.TrimSpace(targetName)]
		if !exists {
			return "", nil, classifyOperationError(fmt.Errorf("%w: %s", ErrUnknownWakeTarget, strings.TrimSpace(targetName)), ToolOutcomeNotSent)
		}
		params := map[string]any{"macAddress": target.MACAddress}
		if target.BroadcastIP != "" {
			params["broadcastIP"] = target.BroadcastIP
		}
		return "sendWOLMagicPacket", params, nil
	default:
		return "", nil, classifyOperationError(errors.New("unsupported power action"), ToolOutcomeNotSent)
	}
}
