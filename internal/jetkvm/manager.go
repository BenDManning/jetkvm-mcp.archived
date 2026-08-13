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
	"sync"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
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
	mediaLocks map[string]*sync.Mutex
	provider   SessionProvider
	decoder    Decoder
}

func NewManager(devices []DeviceConfig, provider SessionProvider, options ...ManagerOption) (*Manager, error) {
	if provider == nil {
		return nil, errors.New("session provider is required")
	}
	manager := &Manager{
		devices:    make(map[string]DeviceConfig, len(devices)),
		mediaLocks: make(map[string]*sync.Mutex, len(devices)),
		provider:   provider,
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
		manager.mediaLocks[name] = &sync.Mutex{}
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
	return manager, nil
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
	err = manager.withSession(ctx, device, SessionProfileData, func(session Session) error {
		var pong string
		if err := session.Call(ctx, methodPing, nil, &pong); err != nil {
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
		if err := session.Call(ctx, methodLocalVersion, nil, &version); err != nil {
			status.Warnings = append(status.Warnings, "version unavailable")
		} else {
			status.Application, status.System = version.Application, version.System
		}

		if err := session.Call(ctx, methodActiveExtension, nil, &status.Extension); err != nil {
			status.Warnings = append(status.Warnings, "active extension unavailable")
		}

		var media *firmwareVirtualMediaState
		if err := session.Call(ctx, methodVirtualMediaState, nil, &media); err != nil {
			status.Warnings = append(status.Warnings, "virtual media unavailable")
		} else if projected, projectErr := publicVirtualMediaState(media); projectErr != nil {
			status.Warnings = append(status.Warnings, "virtual media unavailable")
		} else {
			status.VirtualMedia = projected
		}

		var video struct {
			Ready  *bool `json:"ready"`
			Width  int   `json:"width"`
			Height int   `json:"height"`
			FPS    int   `json:"fps"`
		}
		if err := session.Call(ctx, methodVideoState, nil, &video); err != nil {
			status.Warnings = append(status.Warnings, "video unavailable")
		} else {
			status.VideoReady, status.VideoWidth, status.VideoHeight, status.VideoFPS = video.Ready, video.Width, video.Height, video.FPS
		}

		if err := session.Call(ctx, methodUSBState, nil, &status.USBState); err != nil {
			status.Warnings = append(status.Warnings, "USB unavailable")
		} else {
			attached := status.USBState != "not attached" && status.USBState != ""
			status.USBWakeAttached = &attached
		}

		switch status.Extension {
		case "atx-power":
			var atx struct {
				Power *bool `json:"power"`
			}
			if err := session.Call(ctx, methodATXState, nil, &atx); err != nil {
				status.Warnings = append(status.Warnings, "ATX state unavailable")
			} else {
				status.ATXPowerOn = atx.Power
			}
		case "dc-power":
			var dc struct {
				On      *bool   `json:"isOn"`
				Voltage float64 `json:"voltage"`
			}
			if err := session.Call(ctx, methodDCPowerState, nil, &dc); err != nil {
				status.Warnings = append(status.Warnings, "DC state unavailable")
			} else {
				status.DCPowerOn, status.DCVoltage = dc.On, dc.Voltage
			}
		}
		return nil
	})
	if err != nil {
		return mcpserver.Status{}, classifyReadFailure(err)
	}
	return status, nil
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
	if err := manager.withSession(ctx, device, SessionProfileData, func(session Session) error {
		return session.Call(ctx, method, params, nil)
	}); err != nil {
		return mcpserver.PowerResult{}, err
	}
	return mcpserver.PowerResult{Device: device.Name, Action: action, Target: targetName, Status: "completed"}, nil
}

func (manager *Manager) withSession(ctx context.Context, device DeviceConfig, profile SessionProfile, operation func(Session) error) error {
	invoked := false
	err := manager.provider.WithSession(ctx, device, profile, func(session Session) error {
		invoked = true
		return operation(session)
	})
	if err != nil && !invoked {
		var classified interface{ ToolErrorOutcome() string }
		if errors.As(err, &classified) {
			return err
		}
		return classifyOperationError(err, ToolOutcomeNotSent)
	}
	return err
}

func classifyReadFailure(err error) error {
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
