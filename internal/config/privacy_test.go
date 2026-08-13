package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const privacySentinel = "JETKVM-PRIVATE-SENTINEL-7eea7c9f"

func TestDiagnosticsExcludeResolvedSecretsAndConfigPaths(t *testing.T) {
	t.Run("resolved credentials", func(t *testing.T) {
		path := writePrivacyConfig(t, `devices:
  lab:
    url: https://lab.invalid
    password_env: JETKVM_PRIVACY_PASSWORD
http:
  bearer_token_env: JETKVM_PRIVACY_TOKEN
  allowed_origins:
    - invalid-origin
`)
		resolved := make(map[string]bool)
		lookup := func(name string) (string, bool) {
			resolved[name] = true
			return privacySentinel, true
		}
		_, err := Load(path, lookup)
		if err == nil {
			t.Fatal("invalid origin was accepted after resolving credentials")
		}
		if !resolved["JETKVM_PRIVACY_PASSWORD"] || !resolved["JETKVM_PRIVACY_TOKEN"] {
			t.Fatalf("test did not resolve both secret values: %#v", resolved)
		}
		assertPrivacySentinelAbsent(t, "load diagnostic", err)
	})

	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), privacySentinel+".yaml")
		_, err := Load(path, os.LookupEnv)
		if err == nil {
			t.Fatal("missing configuration was accepted")
		}
		assertPrivacySentinelAbsent(t, "open diagnostic", err)
	})
}

func TestConfigDiagnosticsExcludeInlineSecretValues(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
	}{
		{
			name: "unknown inline password",
			yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    password: " + privacySentinel + "\n",
		},
		{
			name: "device URL user information",
			yaml: "devices:\n  lab:\n    url: https://admin:" + privacySentinel + "@lab.invalid\n",
		},
		{
			name: "allowed origin user information",
			yaml: "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins:\n    - https://admin:" + privacySentinel + "@mcp.invalid\n",
		},
		{
			name: "media origin user information",
			yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    media_url_allowed_origins:\n      - https://admin:" + privacySentinel + "@media.invalid\n",
		},
		{
			name: "invalid typed value",
			yaml: "devices:\n  lab:\n    url: https://lab.invalid\n    insecure_skip_verify: " + privacySentinel + "\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(writePrivacyConfig(t, test.yaml), os.LookupEnv)
			if err == nil {
				t.Fatal("private inline value was accepted")
			}
			assertPrivacySentinelAbsent(t, "config diagnostic", err)
		})
	}
}

func TestMalformedConfigDiagnosticsExcludeShortScalarValues(t *testing.T) {
	for _, test := range []struct {
		name         string
		yaml         string
		privateValue string
	}{
		{
			name:         "invalid boolean",
			yaml:         "devices:\n  lab:\n    url: https://lab.invalid\n    insecure_skip_verify: s3cr3t\n",
			privateValue: "s3cr3t",
		},
		{
			name:         "invalid devices map",
			yaml:         "devices: hunter2\n",
			privateValue: "hunter2",
		},
		{
			name:         "invalid origins sequence",
			yaml:         "devices:\n  lab:\n    url: https://lab.invalid\nhttp:\n  allowed_origins: token123\n",
			privateValue: "token123",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(writePrivacyConfig(t, test.yaml), os.LookupEnv)
			if err == nil {
				t.Fatal("malformed configuration was accepted")
			}
			if strings.Contains(err.Error(), test.privateValue) {
				t.Fatalf("config diagnostic exposed private scalar %q: %v", test.privateValue, err)
			}
		})
	}
}

func TestPrivacySentinelGuardRejectsLeakingDiagnostic(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		err := errors.New("diagnostic contains " + privacySentinel)
		if !bytes.Contains([]byte(err.Error()), []byte(privacySentinel)) {
			t.Fatal("sentinel guard failed to detect an error leak")
		}
	})

	t.Run("output", func(t *testing.T) {
		output := []byte("output contains " + privacySentinel)
		if !bytes.Contains(output, []byte(privacySentinel)) {
			t.Fatal("sentinel guard failed to detect an output leak")
		}
	})
}

func TestThreatModelClassifiesEveryConfigurationField(t *testing.T) {
	document, err := os.ReadFile("../../docs/threat-model.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range yamlFieldNames(reflect.TypeFor[fileConfig](), make(map[reflect.Type]bool)) {
		if !bytes.Contains(document, []byte("`"+field+"`")) {
			t.Errorf("threat model does not classify YAML field %q", field)
		}
	}
}

func assertPrivacySentinelAbsent(t *testing.T, surface string, err error) {
	t.Helper()
	if err != nil && bytes.Contains([]byte(err.Error()), []byte(privacySentinel)) {
		t.Fatalf("%s exposed a private sentinel: %v", surface, err)
	}
}

func writePrivacyConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "privacy.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func yamlFieldNames(current reflect.Type, seen map[reflect.Type]bool) []string {
	for current.Kind() == reflect.Pointer || current.Kind() == reflect.Map || current.Kind() == reflect.Slice {
		current = current.Elem()
	}
	if current.Kind() != reflect.Struct || seen[current] {
		return nil
	}
	seen[current] = true
	var names []string
	for index := 0; index < current.NumField(); index++ {
		field := current.Field(index)
		if name := strings.Split(field.Tag.Get("yaml"), ",")[0]; name != "" && name != "-" {
			names = append(names, name)
		}
		names = append(names, yamlFieldNames(field.Type, seen)...)
	}
	return names
}
