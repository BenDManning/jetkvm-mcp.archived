package compatibility

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	pinnedJetKVMSource  = "b3c29a44d9e2862b8ff7530830781803ce27b060"
	currentJetKVMSource = "fe77acd5f00300a4ab9acd5da57d7bb0916351d9"
	validatedServerRef  = "f1be6653e494ba618c40dab9dd12cda34bd0bfab"
	ledgerBaselineRef   = "35f9aac20e55e4b690c084b2ad90b4134d0e0f53"
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
	if got.SchemaVersion != 1 || len(got.Entries) != 3 {
		t.Fatalf("ledger version/entries = %d/%d", got.SchemaVersion, len(got.Entries))
	}

	ids := make(map[string]bool)
	sha := regexp.MustCompile(`^[0-9a-f]{40}$`)
	allowedChecks := map[string]bool{
		"auth_source_review": true, "signaling_source_review": true, "rpc_source_review": true,
		"video_source_review": true, "hid_source_review": true, "virtual_media_source_review": true,
		"auth_path_drift": true, "signaling_path_drift": true, "rpc_path_drift": true,
		"video_path_drift": true, "hid_path_drift": true, "virtual_media_path_drift": true,
		"mcp_discovery": true, "status": true, "capture": true,
	}
	for _, entry := range got.Entries {
		if entry.ID == "" || ids[entry.ID] {
			t.Fatalf("invalid or duplicate entry id %q", entry.ID)
		}
		ids[entry.ID] = true
		if _, err := time.Parse("2006-01-02", entry.ObservedOn); err != nil {
			t.Fatalf("entry %q date: %v", entry.ID, err)
		}
		if entry.Result != "pass" && entry.Result != "review_required" || len(entry.Checks) == 0 || len(entry.Limitations) == 0 {
			t.Fatalf("entry %q lacks bounded result/checks/limitations", entry.ID)
		}
		if !sha.MatchString(entry.Server.SourceRef) {
			t.Fatalf("entry %q server source ref is not exact", entry.ID)
		}
		for _, check := range entry.Checks {
			if !allowedChecks[check] {
				t.Fatalf("entry %q contains unreviewed or mutating check %q", entry.ID, check)
			}
		}
	}

	source := got.Entries[0]
	if source.EvidenceClass != "source_review" || source.JetKVM.SourceRef != pinnedJetKVMSource ||
		source.JetKVM.FirmwareVersion != "not_observed" || source.Server.SourceRef != ledgerBaselineRef ||
		source.EnvironmentClass != "offline_upstream_source" || source.Result != "pass" ||
		!reflect.DeepEqual(source.Checks, []string{"auth_source_review", "signaling_source_review", "rpc_source_review", "video_source_review", "hid_source_review", "virtual_media_source_review"}) {
		t.Fatalf("source-review seed = %#v", source)
	}

	drift := got.Entries[1]
	driftChecks := []string{"auth_path_drift", "signaling_path_drift", "rpc_path_drift", "video_path_drift", "hid_path_drift", "virtual_media_path_drift"}
	if drift.EvidenceClass != "source_drift" || drift.JetKVM.SourceRef != currentJetKVMSource ||
		drift.JetKVM.FirmwareVersion != "not_observed" || drift.Server.SourceRef != ledgerBaselineRef ||
		drift.EnvironmentClass != "offline_upstream_source" || drift.Result != "review_required" ||
		!reflect.DeepEqual(drift.Checks, driftChecks) {
		t.Fatalf("source-drift seed = %#v", drift)
	}

	hardware := got.Entries[2]
	if hardware.EvidenceClass != "read_only_hardware" || hardware.JetKVM.SourceRef != "not_attributed" ||
		hardware.JetKVM.FirmwareVersion != "not_retained" || hardware.Server.SourceRef != validatedServerRef ||
		hardware.EnvironmentClass != "non_production_physical" || hardware.Result != "pass" ||
		!reflect.DeepEqual(hardware.Checks, []string{"mcp_discovery", "status", "capture"}) {
		t.Fatalf("read-only hardware seed = %#v", hardware)
	}
}

func TestCompatibilityPolicyMapsEverySurfaceToBoundedEvidenceTriggers(t *testing.T) {
	policy, err := os.ReadFile(filepath.Join("..", "..", "docs", "compatibility", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(policy)
	for _, required := range []string{
		"Update the corresponding synthetic fixtures and focused tests, then perform source review.",
		"Run the separately approved read-only validator",
		"Use the separately approval-gated mutation checklist",
		"Keep or add an explicit compatibility warning.",
		"Do not infer support, auto-upgrade, or treat fake-device success as authority.",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("compatibility policy lacks trigger %q", required)
		}
	}
	for _, surface := range []string{"Authentication", "signaling", "RPC", "RTP/video", "HID/media"} {
		if !strings.Contains(text, surface) {
			t.Fatalf("compatibility policy lacks surface %q", surface)
		}
	}
}
