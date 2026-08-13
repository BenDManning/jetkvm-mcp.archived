package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BenDManning/jetkvm-mcp/internal/httporigin"
	"github.com/BenDManning/jetkvm-mcp/internal/jetkvm"
	"gopkg.in/yaml.v3"
)

const maxConfigBytes = 1 << 20

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Runtime struct {
	Devices            []jetkvm.DeviceConfig
	Limits             jetkvm.Limits
	HTTPBearerToken    string
	HTTPAllowedOrigins []string
}

type fileConfig struct {
	Devices map[string]fileDevice `yaml:"devices"`
	Limits  fileLimits            `yaml:"limits,omitempty"`
	HTTP    fileHTTP              `yaml:"http,omitempty"`
}

type fileLimits struct {
	MaxOperations          *int `yaml:"max_operations,omitempty"`
	MaxOperationsPerDevice *int `yaml:"max_operations_per_device,omitempty"`
	MaxSessions            *int `yaml:"max_sessions,omitempty"`
	MaxCaptures            *int `yaml:"max_captures,omitempty"`
	MaxDecoders            *int `yaml:"max_decoders,omitempty"`
}

type fileDevice struct {
	URL                    string                         `yaml:"url"`
	PasswordEnvironment    string                         `yaml:"password_env,omitempty"`
	InsecureSkipVerify     bool                           `yaml:"insecure_skip_verify,omitempty"`
	MediaDirectory         string                         `yaml:"media_directory,omitempty"`
	MediaURLAllowedOrigins []string                       `yaml:"media_url_allowed_origins,omitempty"`
	WakeOnLAN              map[string]fileWakeOnLANTarget `yaml:"wake_on_lan,omitempty"`
}

type fileWakeOnLANTarget struct {
	MACAddress  string `yaml:"mac_address"`
	BroadcastIP string `yaml:"broadcast_ip,omitempty"`
}

type fileHTTP struct {
	BearerTokenEnvironment string   `yaml:"bearer_token_env,omitempty"`
	AllowedOrigins         []string `yaml:"allowed_origins,omitempty"`
}

type LookupEnvironment func(string) (string, bool)

func Load(path string, lookup LookupEnvironment) (Runtime, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	file, err := os.Open(path)
	if err != nil {
		return Runtime{}, errors.New("open config")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil {
		return Runtime{}, errors.New("read config")
	}
	if len(data) == 0 || len(data) > maxConfigBytes {
		return Runtime{}, errors.New("config must be non-empty and at most 1 MiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var source fileConfig
	if err := decoder.Decode(&source); err != nil {
		return Runtime{}, errors.New("decode config")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Runtime{}, errors.New("config must contain exactly one YAML document")
	}
	if len(source.Devices) == 0 {
		return Runtime{}, errors.New("at least one device is required")
	}
	limits := jetkvm.DefaultLimits()
	if source.Limits.MaxOperations != nil {
		limits.MaxOperations = *source.Limits.MaxOperations
	}
	if source.Limits.MaxOperationsPerDevice != nil {
		limits.MaxOperationsPerDevice = *source.Limits.MaxOperationsPerDevice
	}
	if source.Limits.MaxSessions != nil {
		limits.MaxSessions = *source.Limits.MaxSessions
	}
	if source.Limits.MaxCaptures != nil {
		limits.MaxCaptures = *source.Limits.MaxCaptures
	}
	if source.Limits.MaxDecoders != nil {
		limits.MaxDecoders = *source.Limits.MaxDecoders
	}
	if _, err := jetkvm.ValidateLimits(limits); err != nil {
		return Runtime{}, errors.New("invalid admission limits")
	}

	names := make([]string, 0, len(source.Devices))
	for name := range source.Devices {
		names = append(names, name)
	}
	sort.Strings(names)
	runtime := Runtime{Devices: make([]jetkvm.DeviceConfig, 0, len(names)), Limits: limits}
	for _, name := range names {
		configured := source.Devices[name]
		rawURL := strings.TrimSpace(configured.URL)
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
			return Runtime{}, fmt.Errorf("device %q requires an HTTP(S) URL", name)
		}
		if parsed.User != nil {
			return Runtime{}, fmt.Errorf("device %q URL must not include credentials", name)
		}
		if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(rawURL, "#") {
			return Runtime{}, fmt.Errorf("device %q URL must not include a query or fragment", name)
		}
		password, err := resolveEnvironment(configured.PasswordEnvironment, lookup)
		if err != nil {
			return Runtime{}, fmt.Errorf("device %q: %w", name, err)
		}
		mediaDirectory := strings.TrimSpace(configured.MediaDirectory)
		if mediaDirectory != "" {
			if !filepath.IsAbs(mediaDirectory) {
				return Runtime{}, fmt.Errorf("device %q media_directory must be absolute", name)
			}
			mediaDirectory = filepath.Clean(mediaDirectory)
		}
		mediaOrigins := make([]string, 0, len(configured.MediaURLAllowedOrigins))
		seenMediaOrigins := make(map[string]struct{}, len(configured.MediaURLAllowedOrigins))
		for _, value := range configured.MediaURLAllowedOrigins {
			origin, err := normalizeMediaURLOrigin(value)
			if err != nil || strings.Contains(origin, "*") {
				return Runtime{}, fmt.Errorf("device %q media_url_allowed_origins must contain exact HTTP(S) origins without credentials, wildcard hosts, paths, queries, fragments, or invalid ports", name)
			}
			if _, duplicate := seenMediaOrigins[origin]; duplicate {
				continue
			}
			seenMediaOrigins[origin] = struct{}{}
			mediaOrigins = append(mediaOrigins, origin)
		}
		wakeTargets := make(map[string]jetkvm.WakeOnLANTarget, len(configured.WakeOnLAN))
		for targetName, target := range configured.WakeOnLAN {
			wakeTargets[targetName] = jetkvm.WakeOnLANTarget{
				MACAddress: strings.TrimSpace(target.MACAddress), BroadcastIP: strings.TrimSpace(target.BroadcastIP),
			}
		}
		runtime.Devices = append(runtime.Devices, jetkvm.DeviceConfig{
			Name: strings.TrimSpace(name), BaseURL: *parsed, Password: password,
			InsecureSkipVerify: configured.InsecureSkipVerify, MediaDirectory: mediaDirectory,
			MediaURLAllowedOrigins: mediaOrigins, WakeOnLAN: wakeTargets,
		})
	}
	token, err := resolveEnvironment(source.HTTP.BearerTokenEnvironment, lookup)
	if err != nil {
		return Runtime{}, fmt.Errorf("HTTP bearer token: %w", err)
	}
	runtime.HTTPBearerToken = token
	seenOrigins := make(map[string]struct{}, len(source.HTTP.AllowedOrigins))
	for _, configured := range source.HTTP.AllowedOrigins {
		origin, err := normalizeHTTPOrigin(configured)
		if err != nil {
			return Runtime{}, err
		}
		if _, duplicate := seenOrigins[origin]; duplicate {
			continue
		}
		seenOrigins[origin] = struct{}{}
		runtime.HTTPAllowedOrigins = append(runtime.HTTPAllowedOrigins, origin)
	}
	return runtime, nil
}

func normalizeHTTPOrigin(value string) (string, error) {
	origin, err := httporigin.ParseEffective(value)
	if errors.Is(err, httporigin.ErrCredentials) {
		return "", errors.New("HTTP allowed origin must not include credentials")
	}
	if err != nil {
		return "", errors.New("HTTP allowed origin must be an HTTP(S) origin without a path, query, or fragment and with a valid port")
	}
	if strings.Contains(origin.Host, "*") {
		return "", errors.New("HTTP allowed origin must use an exact authority without wildcard hosts")
	}
	return origin.Value, nil
}

func normalizeMediaURLOrigin(value string) (string, error) {
	origin, err := httporigin.ParseEffective(value)
	if err != nil {
		return "", err
	}
	return origin.Value, nil
}

func resolveEnvironment(name string, lookup LookupEnvironment) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if !environmentNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid environment variable name %q", name)
	}
	value, exists := lookup(name)
	if !exists || value == "" {
		return "", fmt.Errorf("environment variable %s is not set", name)
	}
	return value, nil
}
