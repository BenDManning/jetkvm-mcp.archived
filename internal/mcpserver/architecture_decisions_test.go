package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIDiagnosticsAndRawRPCBoundaryAreDocumented(t *testing.T) {
	documents := map[string]string{
		"README":           readRepositoryDocument(t, "README.md"),
		"product contract": readRepositoryDocument(t, filepath.Join("docs", "product-contract.md")),
		"raw RPC ADR":      readRepositoryDocument(t, filepath.Join("docs", "adr", "0005-local-only-raw-rpc.md")),
		"protocol sources": readRepositoryDocument(t, filepath.Join("docs", "protocol-sources.md")),
	}
	for name, document := range map[string]string{
		"README":           documents["README"],
		"product contract": documents["product contract"],
	} {
		for _, required := range []string{
			"config validate",
			"--unsafe-acknowledge-risk",
			"ping",
			"getLocalVersion",
			"getActiveExtension",
			"query",
			"fragment",
		} {
			if !strings.Contains(document, required) {
				t.Errorf("%s does not document %q", name, required)
			}
		}
	}
	for _, required := range []string{"--unsafe-acknowledge-risk", "ping", "getLocalVersion", "getActiveExtension"} {
		if !strings.Contains(documents["raw RPC ADR"], required) {
			t.Errorf("raw RPC ADR does not document %q", required)
		}
	}
	for _, required := range []string{"jsonrpc.go", "ota.go", "ping", "getLocalVersion", "getActiveExtension"} {
		if !strings.Contains(documents["protocol sources"], required) {
			t.Errorf("protocol sources do not document %q", required)
		}
	}
	for name, document := range map[string]string{
		"README":           documents["README"],
		"product contract": documents["product contract"],
	} {
		for _, required := range []string{"go install", "devel+", "vcs.revision"} {
			if !strings.Contains(document, required) {
				t.Errorf("%s does not document build provenance marker %q", name, required)
			}
		}
	}
}

func readRepositoryDocument(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
