package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func TestParseArgsSelectsStdioHTTPAndDebugRPC(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		kind    commandKind
		address string
		method  string
		unsafe  bool
	}{
		{name: "stdio", args: []string{"--config", "config.yaml"}, kind: commandServe},
		{name: "http", args: []string{"--config", "config.yaml", "--http", "127.0.0.1:9000"}, kind: commandServe, address: "127.0.0.1:9000"},
		{name: "debug", args: []string{"debug", "rpc", "--config", "config.yaml", "--device", "lab", "--method", "ping", "--params", "{}"}, kind: commandDebugRPC, method: "ping"},
		{name: "unsafe debug", args: []string{"debug", "rpc", "--config", "config.yaml", "--device", "lab", "--method", "customMethod", "--unsafe-acknowledge-risk"}, kind: commandDebugRPC, method: "customMethod", unsafe: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			options, err := parseArgs(test.args)
			if err != nil {
				t.Fatal(err)
			}
			if options.kind != test.kind || options.httpAddress != test.address || options.debugMethod != test.method || options.configPath != "config.yaml" || options.debugUnsafeAcknowledged != test.unsafe {
				t.Fatalf("options = %+v", options)
			}
		})
	}
}

func TestParseArgsRejectsDeprecatedOrIncompleteCommands(t *testing.T) {
	for _, args := range [][]string{
		{"--sse", "127.0.0.1:9000"},
		{"debug", "rpc", "--device", "lab", "--method", "ping"},
		{"debug", "rpc", "--config", "config.yaml", "--method", "ping"},
		{"debug", "rpc", "--config", "config.yaml", "--device", "lab"},
		{"debug", "other"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("parseArgs(%v) error = nil", args)
		}
	}
}

func TestRunVersionDoesNotRequireConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"--version"}, strings.NewReader(""), &stdout, &stderr, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) == "" || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestShutdownBudgetIsSharedAcrossShutdownPhases(t *testing.T) {
	budget := newShutdownBudget(50 * time.Millisecond)
	defer budget.Close()
	first := budget.Context()
	time.Sleep(30 * time.Millisecond)
	second := budget.Context()
	if first != second {
		t.Fatal("shutdown phases received different budget contexts")
	}
	select {
	case <-second.Done():
	case <-time.After(40 * time.Millisecond):
		t.Fatal("second shutdown phase received a fresh timeout")
	}
}

func TestHTTPAndManagerShutdownStartConcurrently(t *testing.T) {
	managerStarted := make(chan struct{})
	releaseManager := make(chan struct{})
	httpShutdown := func(context.Context) error {
		select {
		case <-managerStarted:
			return nil
		case <-time.After(time.Second):
			t.Fatal("HTTP shutdown started before manager shutdown")
			return nil
		}
	}
	managerShutdown := func(context.Context) error {
		close(managerStarted)
		<-releaseManager
		return nil
	}
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(releaseManager)
	}()
	httpErr, managerErr := shutdownTogether(context.Background(), httpShutdown, managerShutdown)
	if httpErr != nil || managerErr != nil {
		t.Fatalf("shutdown errors = (%v, %v)", httpErr, managerErr)
	}
}

func TestResolveVersionUsesReleaseModuleAndDevelopmentProvenance(t *testing.T) {
	installed := &debug.BuildInfo{Main: debug.Module{
		Path:    "github.com/BenDManning/jetkvm-mcp",
		Version: "v1.2.3",
	}}
	development := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/BenDManning/jetkvm-mcp", Version: "v0.1.1-0.20260813152351-0123456789ab"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	for _, test := range []struct {
		name     string
		explicit string
		info     *debug.BuildInfo
		ok       bool
		want     string
	}{
		{name: "release ldflags win", explicit: "v2.0.0", info: installed, ok: true, want: "v2.0.0"},
		{name: "explicit development label wins", explicit: "dev", info: installed, ok: true, want: "dev"},
		{name: "installed module", info: installed, ok: true, want: "v1.2.3"},
		{name: "development revision", info: development, ok: true, want: "devel+0123456789ab.dirty"},
		{name: "metadata-poor development", info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, ok: true, want: "devel"},
		{name: "unavailable metadata", want: "dev"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveVersion(test.explicit, test.info, test.ok); got != test.want {
				t.Fatalf("resolveVersion() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseArgsPreservesActionableFlagErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--unknown-root"},
		{"config", "validate", "--unknown-config"},
		{"debug", "rpc", "--unknown-debug"},
	} {
		_, err := parseArgs(args)
		if err == nil || !strings.Contains(err.Error(), strings.TrimLeft(args[len(args)-1], "-")) {
			t.Fatalf("parseArgs(%v) error = %v", args, err)
		}
	}
}

func TestParseArgsFlagErrorsDoNotEchoValues(t *testing.T) {
	const sentinel = "PRIVATE-FLAG-VALUE-SENTINEL-59a2"
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown", args: []string{"--unknown-root"}, want: "unknown flag --unknown-root"},
		{name: "missing argument", args: []string{"config", "validate", "--config"}, want: "flag --config needs an argument"},
		{name: "invalid value", args: []string{"debug", "rpc", "--unsafe-acknowledge-risk=" + sentinel}, want: "invalid value for flag --unsafe-acknowledge-risk"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseArgs(test.args)
			if err == nil || err.Error() != test.want {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if strings.Contains(err.Error(), sentinel) {
				t.Fatalf("flag error leaked value: %v", err)
			}
		})
	}
}

func TestRunHelpDoesNotRequireConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"--help"}, strings.NewReader(""), &stdout, &stderr, nil); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Usage:", "--version", "config validate", "debug rpc"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), expected)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunSubcommandHelpDoesNotRequireConfig(t *testing.T) {
	for _, args := range [][]string{
		{"config", "--help"},
		{"config", "validate", "--help"},
		{"debug", "--help"},
		{"debug", "rpc", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		if err := run(context.Background(), args, strings.NewReader(""), &stdout, &stderr, nil); err != nil {
			t.Fatalf("run(%v): %v", args, err)
		}
		if !strings.Contains(stdout.String(), "Usage:") || stderr.Len() != 0 {
			t.Fatalf("run(%v) stdout=%q stderr=%q", args, stdout.String(), stderr.String())
		}
	}
}

func TestRunConfigValidateIsOfflineAndDoesNotRequireFFmpeg(t *testing.T) {
	t.Setenv("PATH", "")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("devices:\n  offline-lab:\n    url: https://unreachable.invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"config", "validate", "--config", configPath}, strings.NewReader(""), &stdout, &stderr, nil); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "configuration valid\n" || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunConfigValidateRejectsManagerInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "invalid Wake-on-LAN MAC",
			config: "devices:\n  lab:\n    url: https://lab.invalid\n" +
				"    wake_on_lan:\n      server:\n        mac_address: invalid\n",
			wantErr: `Wake-on-LAN target "server" has invalid MAC address`,
		},
		{
			name: "device names collide after trimming",
			config: "devices:\n  lab:\n    url: https://lab.invalid\n" +
				"  ' lab ':\n    url: https://other.invalid\n",
			wantErr: "device aliases must be unique after trimming",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configPath, []byte(test.config), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			err := run(context.Background(), []string{"config", "validate", "--config", configPath}, strings.NewReader(""), &stdout, &stderr, nil)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunConfigValidateFailureWritesNoPrivateStreams(t *testing.T) {
	const sentinel = "PRIVATE-CONFIG-VALUE-SENTINEL-b531"
	configPath := filepath.Join(t.TempDir(), sentinel+".yaml")
	if err := os.WriteFile(configPath, []byte("devices:\n  lab:\n    url: https://lab.invalid/?token="+sentinel+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"config", "validate", "--config", configPath}, strings.NewReader(""), &stdout, &stderr, nil)
	if err == nil || !strings.Contains(err.Error(), `device "lab" URL must not include a query or fragment`) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), sentinel) || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("error=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
}
