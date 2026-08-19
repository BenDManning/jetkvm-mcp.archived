package compatibility

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

type ledger struct {
	SchemaVersion int           `json:"schemaVersion"`
	Entries       []ledgerEntry `json:"entries"`
}

type ledgerEntry struct {
	ID               string       `json:"id"`
	EvidenceClass    string       `json:"evidenceClass"`
	JetKVM           ledgerJetKVM `json:"jetkvm"`
	Server           ledgerServer `json:"server"`
	ObservedOn       string       `json:"observedOn"`
	EnvironmentClass string       `json:"environmentClass"`
	Checks           []string     `json:"checks"`
	Result           string       `json:"result"`
	Limitations      []string     `json:"limitations"`
}

type ledgerJetKVM struct {
	FirmwareVersion string `json:"firmwareVersion"`
	SourceRef       string `json:"sourceRef"`
}

type ledgerServer struct {
	Version   string `json:"version"`
	SourceRef string `json:"sourceRef"`
}

type evidencePolicy struct {
	checks                map[string]bool
	jetKVMSource          string
	jetKVMSourceMustBeSHA bool
	jetKVMFirmwareVersion string
	environmentClass      string
	result                string
}

func TestJetKVMCompatibilityLedgerIsSanitizedAndSourceGrounded(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "compatibility", "jetkvm-ledger.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var got ledger
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		t.Fatal("ledger contains trailing JSON")
	}
	if got.SchemaVersion != 1 || len(got.Entries) == 0 {
		t.Fatalf("ledger version/entries = %d/%d", got.SchemaVersion, len(got.Entries))
	}

	ids := make(map[string]bool)
	sha := regexp.MustCompile(`^[0-9a-f]{40}$`)
	classPolicies := map[string]evidencePolicy{
		"source_review": {
			checks: allowedSet(
				"auth_source_review", "signaling_source_review", "rpc_source_review",
				"video_source_review", "hid_source_review", "virtual_media_source_review",
			),
			jetKVMSourceMustBeSHA: true,
			jetKVMFirmwareVersion: "not_observed",
			environmentClass:      "offline_upstream_source",
			result:                "pass",
		},
		"source_drift": {
			checks: allowedSet(
				"auth_path_drift", "signaling_path_drift", "rpc_path_drift",
				"video_path_drift", "hid_path_drift", "virtual_media_path_drift",
			),
			jetKVMSourceMustBeSHA: true,
			jetKVMFirmwareVersion: "not_observed",
			environmentClass:      "offline_upstream_source",
			result:                "review_required",
		},
		"read_only_hardware": {
			checks:                allowedSet("mcp_discovery", "status", "capture"),
			jetKVMSource:          "not_attributed",
			jetKVMFirmwareVersion: "not_retained",
			environmentClass:      "non_production_physical",
			result:                "pass",
		},
		"mutation_hardware": {
			checks:                allowedSet("virtual_media_http_mount"),
			jetKVMSource:          "not_attributed",
			jetKVMFirmwareVersion: "application 0.5.8 / system 0.2.8",
			environmentClass:      "non_production_physical",
			result:                "observed_failure",
		},
	}
	for _, entry := range got.Entries {
		if entry.ID == "" || ids[entry.ID] {
			t.Fatalf("invalid or duplicate entry id %q", entry.ID)
		}
		ids[entry.ID] = true
		if _, err := time.Parse("2006-01-02", entry.ObservedOn); err != nil {
			t.Fatalf("entry %q date: %v", entry.ID, err)
		}
		if entry.Server.Version == "" || len(entry.Checks) == 0 || len(entry.Limitations) == 0 {
			t.Fatalf("entry %q lacks bounded result/checks/limitations", entry.ID)
		}
		if !sha.MatchString(entry.Server.SourceRef) {
			t.Fatalf("entry %q server source ref is not exact", entry.ID)
		}
		policy, ok := classPolicies[entry.EvidenceClass]
		if !ok {
			t.Fatalf("entry %q contains unsupported evidence class %q", entry.ID, entry.EvidenceClass)
		}
		seenChecks := make(map[string]bool, len(entry.Checks))
		for _, check := range entry.Checks {
			if !policy.checks[check] {
				t.Fatalf("entry %q contains check %q outside the %q claim", entry.ID, check, entry.EvidenceClass)
			}
			if seenChecks[check] {
				t.Fatalf("entry %q repeats check %q", entry.ID, check)
			}
			seenChecks[check] = true
		}
		if entry.JetKVM.FirmwareVersion != policy.jetKVMFirmwareVersion ||
			entry.EnvironmentClass != policy.environmentClass || entry.Result != policy.result ||
			policy.jetKVMSourceMustBeSHA && !sha.MatchString(entry.JetKVM.SourceRef) ||
			!policy.jetKVMSourceMustBeSHA && entry.JetKVM.SourceRef != policy.jetKVMSource {
			t.Fatalf("entry %q exceeds the %q provenance or claim boundary", entry.ID, entry.EvidenceClass)
		}
		for _, limitation := range entry.Limitations {
			if limitation == "" {
				t.Fatalf("entry %q contains an empty limitation", entry.ID)
			}
		}
	}
	if !ids["mutation-hardware-http-mount-2026-08-19"] {
		t.Fatal("ledger lacks the retained HTTP virtual-media mount failure")
	}
}

func allowedSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
