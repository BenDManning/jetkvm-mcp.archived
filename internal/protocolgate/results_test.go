package protocolgate

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseOfficialServerScenarioList(t *testing.T) {
	output := `Server scenarios (test against a server):
  - server-initialize [2025-06-18,2025-11-25]
  - ping [2025-06-18,2025-11-25]
`
	got, err := ParseOfficialServerScenarioList(output)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"server-initialize", "ping"}) {
		t.Fatalf("scenarios = %v", got)
	}
	digest, err := OfficialServerScenarioInventoryDigest(output)
	if err != nil || digest != "5aa42307145d053011425ba79ede7a330c207b826cf1c3924f50bb01d1de140c" {
		t.Fatalf("inventory digest = %q err=%v", digest, err)
	}
	drifted := strings.Replace(output, "ping [2025-06-18,2025-11-25]", "ping [2026-07-28]", 1)
	driftedDigest, err := OfficialServerScenarioInventoryDigest(drifted)
	if err != nil || driftedDigest == digest {
		t.Fatalf("applicability drift digest = %q err=%v", driftedDigest, err)
	}
	if _, err := ParseOfficialServerScenarioList("Server scenarios:\nnoise\n"); err == nil {
		t.Fatal("unparseable inventory accepted")
	}
	if _, err := ParseOfficialServerScenarioList(output + "  - NEW_SCENARIO [2025-11-25]\n"); err == nil {
		t.Fatal("partially unrecognized inventory drift accepted")
	}
}

func TestValidateConformanceScenarioResultFailsClosedOnSkipsAndSummaries(t *testing.T) {
	passing := "Checks:\n2026-08-14T00:00:00Z [wire-schema-valid] SUCCESS valid\n\nTest Results:\nPassed: 1/1, 0 failed, 0 warnings\n"
	if err := ValidateConformanceScenarioResult(passing, nil); err != nil {
		t.Fatal(err)
	}
	passingWithSkip := "Checks:\n\x1b[90m2026-08-14T00:00:00Z\x1b[0m [optional-check ] \x1b[0mSKIPPED\x1b[0m not exposed\n\nTest Results:\nPassed: 1/1, 0 failed, 0 warnings\n"
	if err := ValidateConformanceScenarioResult(passingWithSkip, []string{"optional-check"}); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{
		"SKIPPED: scenario 'ping' is not applicable\n",
		passingWithSkip,
		"Test Results:\nPassed: 1/2, 0 failed, 0 warnings\n",
		"Test Results:\nPassed: 1/1, 0 failed, 1 warnings\n",
		passing + "Passed: 0/1, 1 failed, 0 warnings\n",
		passing + "2026-08-14T00:00:00Z optional-check SKIPPED not exposed\n",
		passing + "2026-08-14T00:00:00Z [wire-schema-valid] WARNING suspicious\n",
	} {
		if err := ValidateConformanceScenarioResult(output, nil); err == nil {
			t.Fatalf("invalid conformance output accepted: %q", output)
		}
	}
}

func TestValidateInspectorResult(t *testing.T) {
	cases := []struct {
		name   string
		method string
		input  any
	}{
		{
			name:   "initialize",
			method: "initialize",
			input: map[string]any{"result": map[string]any{
				"serverInfo":      map[string]any{"name": "jetkvm-mcp", "version": "dev"},
				"protocolVersion": "2025-11-25",
				"capabilities":    map[string]any{"tools": map[string]any{}},
			}},
		},
		{
			name:   "tools list",
			method: "tools/list",
			input: map[string]any{"result": map[string]any{
				"tools": []any{map[string]any{"name": "jetkvm_list_devices"}},
			}},
		},
		{
			name:   "fixture-safe call",
			method: "tools/call:jetkvm_list_devices",
			input: map[string]any{"result": map[string]any{
				"content":           []any{map[string]any{"type": "text", "text": "Configured devices: fixture"}},
				"structuredContent": map[string]any{"devices": []any{"fixture"}},
			}},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateInspectorResult(test.method, encoded, "jetkvm_list_devices"); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, input := range []string{
		`not-json`,
		`{"result":{}}`,
		`{"result":{"tools":[{"name":"another_tool"}]}}`,
	} {
		if err := ValidateInspectorResult("tools/list", []byte(input), "jetkvm_list_devices"); err == nil {
			t.Fatalf("invalid Inspector result accepted: %s", input)
		}
	}
}

func TestCanonicalInspectorResultComparesResultPayloads(t *testing.T) {
	left := []byte(`{"result":{"tools":[{"name":"jetkvm_list_devices","description":"List configured aliases"}]}}`)
	right := []byte(`{"result":{"tools":[{"description":"List configured aliases","name":"jetkvm_list_devices"}]}}`)
	leftCanonical, err := CanonicalInspectorResult(left)
	if err != nil {
		t.Fatal(err)
	}
	rightCanonical, err := CanonicalInspectorResult(right)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(leftCanonical, rightCanonical) {
		t.Fatalf("canonical results differ:\n%s\n%s", leftCanonical, rightCanonical)
	}
	if _, err := CanonicalInspectorResult([]byte(`{"error":{"code":-32603}}`)); err == nil {
		t.Fatal("error envelope accepted as a comparable result")
	}
}
