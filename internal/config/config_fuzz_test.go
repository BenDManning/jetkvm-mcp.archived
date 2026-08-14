package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BenDManning/jetkvm-mcp/internal/jetkvm"
)

const maxFuzzConfigBytes = 64 << 10

func FuzzLoadConfig(f *testing.F) {
	for _, seed := range []string{
		"devices:\n  fixture:\n    url: https://jetkvm.example.invalid\n",
		"devices:\n  fixture:\n    url: https://jetkvm.example.invalid/base\n    password_env: JETKVM_PASSWORD\n",
		"devices:\n  fixture:\n    url: https://jetkvm.example.invalid?private=value\n",
		"devices:\n  fixture:\n    url: https://user:private@jetkvm.example.invalid\n",
		"devices:\n  fixture:\n    url: https://jetkvm.example.invalid\n    unknown: true\n",
		"devices: {}\n---\ndevices: {}\n",
		"",
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxFuzzConfigBytes {
			data = data[:maxFuzzConfigBytes]
		}
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		lookup := func(name string) (string, bool) {
			if !environmentNamePattern.MatchString(name) {
				t.Fatalf("invalid environment name reached lookup")
			}
			return "synthetic-value", true
		}
		first, firstErr := Load(path, lookup)
		second, secondErr := Load(path, lookup)
		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("config acceptance was not deterministic")
		}
		if firstErr != nil {
			if firstErr.Error() != secondErr.Error() {
				t.Fatalf("config rejection was not deterministic")
			}
			return
		}
		if !reflect.DeepEqual(first, second) || len(first.Devices) == 0 {
			t.Fatalf("accepted config did not round-trip deterministically")
		}
		if _, err := jetkvm.ValidateLimits(first.Limits); err != nil {
			t.Fatalf("accepted invalid limits")
		}
		for _, device := range first.Devices {
			if device.Name == "" || device.BaseURL.User != nil || device.BaseURL.RawQuery != "" || device.BaseURL.ForceQuery || device.BaseURL.Fragment != "" {
				t.Fatalf("accepted unsafe device URL")
			}
			if device.Password != "" && device.Password != "synthetic-value" {
				t.Fatalf("accepted password outside deterministic lookup")
			}
		}
	})
}
