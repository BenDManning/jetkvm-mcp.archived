package config

import (
	"os"
	"path/filepath"
	"reflect"
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
    media_url_allowed_origins:
      - https://MEDIA.EXAMPLE.INVALID:8443
      - https://media.example.invalid:8443
      - https://default.example.invalid
      - https://default.example.invalid:443
      - https://default.example.invalid:0443
      - http://127.0.0.1:8080
      - http://[::1]:8080
      - https://[2001:0db8:0:0:0:0:0:1]:8443
      - https://[2001:db8::1]:8443
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
	if len(first.MediaURLAllowedOrigins) != 5 || first.MediaURLAllowedOrigins[0] != "https://media.example.invalid:8443" || first.MediaURLAllowedOrigins[1] != "https://default.example.invalid" || first.MediaURLAllowedOrigins[2] != "http://127.0.0.1:8080" || first.MediaURLAllowedOrigins[3] != "http://[::1]:8080" || first.MediaURLAllowedOrigins[4] != "https://[2001:db8::1]:8443" {
		t.Fatalf("media URL allowed origins = %#v", first.MediaURLAllowedOrigins)
	}
	if loaded.HTTPBearerToken != "http-secret" {
		t.Fatalf("HTTP token was not resolved")
	}
	if len(loaded.HTTPAllowedOrigins) != 1 || loaded.HTTPAllowedOrigins[0] != "https://mcp.example.invalid" {
		t.Fatalf("HTTP allowed origins = %#v", loaded.HTTPAllowedOrigins)
	}
}

func TestLoadNormalizesHTTPAdmissionOriginsByEffectivePort(t *testing.T) {
	path := writeConfig(t, `
devices:
  lab:
    url: https://lab.invalid
http:
  allowed_origins:
    - https://MCP.EXAMPLE.INVALID
    - https://mcp.example.invalid:443
    - http://mcp.example.invalid
    - http://mcp.example.invalid:080
    - https://[2001:0db8:0:0:0:0:0:1]:8443
    - https://[2001:db8::1]:8443
`)
	loaded, err := Load(path, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://mcp.example.invalid",
		"http://mcp.example.invalid",
		"https://[2001:db8::1]:8443",
	}
	if !reflect.DeepEqual(loaded.HTTPAllowedOrigins, want) {
		t.Fatalf("HTTP allowed origins = %#v, want %#v", loaded.HTTPAllowedOrigins, want)
	}
}

func TestLoadRejectsUnknownInlineCredentialAndMissingEnvironment(t *testing.T) {
	for _, test := range []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{name: "inline password", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    password: secret\n", wantErr: "decode config"},
		{name: "URL credentials", yaml: "devices:\n  lab:\n    url: https://admin:secret@lab.invalid\n", wantErr: "URL must not include credentials"},
		{name: "missing password environment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    password_env: MISSING_PASSWORD\n", wantErr: "MISSING_PASSWORD"},
		{name: "missing bearer environment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  bearer_token_env: MISSING_TOKEN\n", wantErr: "MISSING_TOKEN"},
		{name: "allowed origin credentials", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://admin:secret@mcp.invalid\n", wantErr: "must not include credentials"},
		{name: "allowed origin wildcard host", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://*.mcp.invalid\n", wantErr: "without wildcard hosts"},
		{name: "allowed origin bare wildcard host", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://*\n", wantErr: "without wildcard hosts"},
		{name: "allowed origin embedded wildcard host", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.*.invalid\n", wantErr: "without wildcard hosts"},
		{name: "allowed origin path", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid/path\n", wantErr: "without a path"},
		{name: "allowed origin empty query", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid?\n", wantErr: "without a path"},
		{name: "allowed origin empty fragment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid#\n", wantErr: "without a path"},
		{name: "allowed origin invalid port", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://mcp.invalid:99999\n", wantErr: "valid port"},
		{name: "allowed origin empty port", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - 'https://mcp.invalid:'\n", wantErr: "valid port"},
		{name: "allowed origin unbracketed IPv6", yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - 'http://::1'\n", wantErr: "valid port"},
		{name: "media origin credentials", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://admin:secret@media.invalid\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin wildcard", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://*.media.invalid\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin path", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://media.invalid/images\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin query", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://media.invalid?token=secret\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin fragment", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://media.invalid#private\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin non-HTTP scheme", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - ftp://media.invalid\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin empty port", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - 'https://media.invalid:'\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin unbracketed IPv6", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - 'http://::1'\n", wantErr: "exact HTTP(S) origins"},
		{name: "media origin invalid port", yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://media.invalid:99999\n", wantErr: "exact HTTP(S) origins"},
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

func TestLoadRejectsDiscardedDeviceURLComponentsWithoutValues(t *testing.T) {
	const sentinel = "PRIVATE-URL-SENTINEL-93f0"
	for _, rawURL := range []string{
		"https://lab.invalid/base?token=" + sentinel,
		"https://lab.invalid/base#" + sentinel,
		"https://lab.invalid/base#",
	} {
		_, err := Load(writeConfig(t, "devices:\n  lab:\n    url: "+rawURL+"\n"), func(string) (string, bool) { return "", false })
		if err == nil || !strings.Contains(err.Error(), `device "lab" URL must not include a query or fragment`) {
			t.Fatalf("url=%q error=%v", rawURL, err)
		}
		if strings.Contains(err.Error(), sentinel) || strings.Contains(err.Error(), "lab.invalid") {
			t.Fatalf("error leaked URL content: %v", err)
		}
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
