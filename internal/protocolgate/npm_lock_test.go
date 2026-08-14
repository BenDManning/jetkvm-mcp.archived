package protocolgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyNPMLockRejectsPinAndGraphDrift(t *testing.T) {
	pins, err := LoadPins(filepath.Join("..", "..", "testdata", "mcp-gates", "pins.json"))
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join("..", "..", "testdata", "mcp-gates", "npm", "package-lock.json")
	if err := VerifyNPMLock(lockPath, pins); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutation := range map[string]string{
		"top_integrity":            strings.Replace(string(original), pins.Conformance.Integrity, "sha512-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==", 1),
		"unhashed_graph_entry":     strings.Replace(string(original), `"packages": {`, `"packages": {"node_modules/unreviewed": {"version":"1.0.0"},`, 1),
		"trailing_json":            string(original) + `{}`,
		"root_dev_dependency":      strings.Replace(string(original), `"dependencies": {`, `"devDependencies":{"unreviewed":"1.0.0"},"dependencies": {`, 1),
		"root_optional_dependency": strings.Replace(string(original), `"dependencies": {`, `"optionalDependencies":{"unreviewed":"1.0.0"},"dependencies": {`, 1),
		"root_peer_dependency":     strings.Replace(string(original), `"dependencies": {`, `"peerDependencies":{"unreviewed":"1.0.0"},"dependencies": {`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "package-lock.json")
			if err := os.WriteFile(path, []byte(mutation), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := VerifyNPMLock(path, pins); err == nil {
				t.Fatal("mutated npm lock was accepted")
			}
		})
	}
}
