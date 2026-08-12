package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadResolvesEnvironmentWithoutInlineSecrets(t *testing.T) {
	path := writeConfig(t, `
devices:
  rack-b:
    url: https://rack-b.invalid
    password_env: JETKVM_RACK_B_PASSWORD
  rack-a:
    url: http://rack-a.invalid/base
    password_env: JETKVM_RACK_A_PASSWORD
    insecure_skip_verify: true
    media_directory: /media
    wake_on_lan:
      server:
        mac_address: "02:00:00:00:00:01"
        broadcast_ip: 192.0.2.255
http:
  bearer_token_env: JETKVM_MCP_HTTP_TOKEN
  allowed_origins:
    - https://mcp.example.invalid
    - https://mcp.example.invalid
`)
	values := map[string]string{
		"JETKVM_RACK_A_PASSWORD": "a-secret", "JETKVM_RACK_B_PASSWORD": "b-secret", "JETKVM_MCP_HTTP_TOKEN": "http-secret",
	}
	loaded, err := Load(path, func(name string) (string, bool) { value, ok := values[name]; return value, ok })
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Devices) != 2 || loaded.Devices[0].Name != "rack-a" || loaded.Devices[1].Name != "rack-b" {
		t.Fatalf("devices = %#v", loaded.Devices)
	}
	first := loaded.Devices[0]
	if first.Password != "a-secret" || !first.InsecureSkipVerify || first.MediaDirectory != "/media" || first.WakeOnLAN["server"].MACAddress != "02:00:00:00:00:01" {
		t.Fatalf("first device = %#v", first)
	}
	if loaded.HTTPBearerToken != "http-secret" {
		t.Fatalf("HTTP token was not resolved")
	}
	if len(loaded.HTTPAllowedOrigins) != 1 || loaded.HTTPAllowedOrigins[0] != "https://mcp.example.invalid" {
		t.Fatalf("HTTP allowed origins = %#v", loaded.HTTPAllowedOrigins)
	}
}

func TestLoadRejectsUnknownInlineCredentialAndMissingEnvironment(t *testing.T) {
	for _, test := range []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{name: "inline password", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    password: secret\n", wantErr: "field password not found"},
		{name: "URL credentials", yaml: "devices:\n  lab:\n    url: https://admin:secret@lab.invalid\n", wantErr: "URL must not include credentials"},
		{name: "missing password environment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    password_env: MISSING_PASSWORD\n", wantErr: "MISSING_PASSWORD"},
		{name: "missing bearer environment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  bearer_token_env: MISSING_TOKEN\n", wantErr: "MISSING_TOKEN"},
		{name: "allowed origin credentials", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://admin:secret@mcp.invalid\n", wantErr: "must not include credentials"},
		{name: "allowed origin path", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid/path\n", wantErr: "without a path"},
		{name: "allowed origin empty query", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid?\n", wantErr: "without a path"},
		{name: "allowed origin empty fragment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid#\n", wantErr: "without a path"},
		{name: "allowed origin invalid port", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid:99999\n", wantErr: "valid port"},
		{name: "allowed origin empty port", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - 'https://mcp.invalid:'\n", wantErr: "valid port"},
		{name: "allowed origin unbracketed IPv6", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - 'http://::1'\n", wantErr: "valid port"},
		{name: "empty devices", yaml: "devices: {}\n", wantErr: "at least one device"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, test.yaml), func(string) (string, bool) { return "", false })
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("error leaked credential: %v", err)
			}
		})
	}
}

func TestLoadRejectsTrailingYAMLDocument(t *testing.T) {
	_, err := Load(writeConfig(t, "devices:\n  lab:\n    url: https://lab.invalid\n---\ndevices: {}\n"), os.LookupEnv)
	if err == nil {
		t.Fatal("trailing YAML document accepted")
	}
}

func TestLoadOpenErrorDoesNotExposeConfigPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "PRIVATE-CONFIG-PATH-SENTINEL.yaml")
	_, err := Load(path, os.LookupEnv)
	if err == nil {
		t.Fatal("missing configuration was accepted")
	}
	if strings.Contains(err.Error(), "PRIVATE-CONFIG-PATH-SENTINEL") {
		t.Fatalf("error leaked configuration path: %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
