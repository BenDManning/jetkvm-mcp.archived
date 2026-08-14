package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var expectedOperations = []string{
	"jetkvm_keyboard",
	"jetkvm_mouse",
	"jetkvm_press_host_power_button",
	"jetkvm_press_host_reset_button",
	"jetkvm_force_host_power_off",
	"jetkvm_turn_host_dc_power_on",
	"jetkvm_turn_host_dc_power_off",
	"jetkvm_wake_host_lan",
	"jetkvm_wake_host_usb",
	"jetkvm_upload_virtual_media_file",
	"jetkvm_mount_virtual_media_url",
	"jetkvm_mount_virtual_media_file",
	"jetkvm_unmount_virtual_media",
}

func completePlan() map[string]any {
	steps := make([]any, 0, len(expectedOperations))
	for _, operation := range expectedOperations {
		step := map[string]any{
			"operation":                   operation,
			"consequence_acknowledged":    true,
			"preconditions_confirmed":     true,
			"postcondition_observable":    true,
			"timeout_seconds":             30,
			"recovery_ready":              true,
			"never_retry_unknown_outcome": true,
		}
		if strings.Contains(operation, "virtual_media") {
			step["cleanup_planned"] = true
		}
		if operation == "jetkvm_upload_virtual_media_file" || operation == "jetkvm_mount_virtual_media_file" {
			step["integrity_check_planned"] = true
		}
		steps = append(steps, step)
	}
	return map[string]any{
		"schema": "jetkvm.mutation-validation.v1",
		"mode":   "dry_run",
		"target": map[string]any{
			"marked_expendable":  true,
			"identity_confirmed": true,
			"non_production":     true,
		},
		"controls": map[string]any{
			"observer_ready":           true,
			"recovery_ready":           true,
			"emergency_stop_ready":     true,
			"per_plan_acknowledgement": true,
			"execution_approved":       false,
		},
		"steps": steps,
	}
}

func writePlan(t *testing.T, plan map[string]any) string {
	t.Helper()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunAcceptsCompleteDryRunAndNeverAuthorizesExecution(t *testing.T) {
	var output bytes.Buffer
	if code := run([]string{"--plan", writePlan(t, completePlan())}, &output); code != 0 {
		t.Fatalf("run exit = %d, output = %q", code, output.String())
	}
	var report struct {
		Schema              string `json:"schema"`
		Result              string `json:"result"`
		CheckedSteps        int    `json:"checked_steps"`
		ExecutionAuthorized bool   `json:"execution_authorized"`
	}
	decoder := json.NewDecoder(&output)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != "jetkvm.mutation-validation.report.v1" || report.Result != "pass" || report.CheckedSteps != len(expectedOperations) || report.ExecutionAuthorized {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunFailsClosedForUnsafeOrIncompletePlan(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "live mode", mutate: func(plan map[string]any) { plan["mode"] = "execute" }},
		{name: "unmarked target", mutate: func(plan map[string]any) { plan["target"].(map[string]any)["marked_expendable"] = false }},
		{name: "unconfirmed identity", mutate: func(plan map[string]any) { plan["target"].(map[string]any)["identity_confirmed"] = false }},
		{name: "production target", mutate: func(plan map[string]any) { plan["target"].(map[string]any)["non_production"] = false }},
		{name: "execution approval embedded", mutate: func(plan map[string]any) { plan["controls"].(map[string]any)["execution_approved"] = true }},
		{name: "missing observer", mutate: func(plan map[string]any) { plan["controls"].(map[string]any)["observer_ready"] = false }},
		{name: "missing operation", mutate: func(plan map[string]any) { plan["steps"] = plan["steps"].([]any)[1:] }},
		{name: "duplicate operation", mutate: func(plan map[string]any) {
			plan["steps"].([]any)[1].(map[string]any)["operation"] = expectedOperations[0]
		}},
		{name: "unbounded timeout", mutate: func(plan map[string]any) { plan["steps"].([]any)[0].(map[string]any)["timeout_seconds"] = 301 }},
		{name: "retry on unknown", mutate: func(plan map[string]any) {
			plan["steps"].([]any)[0].(map[string]any)["never_retry_unknown_outcome"] = false
		}},
		{name: "upload without integrity", mutate: func(plan map[string]any) {
			delete(plan["steps"].([]any)[9].(map[string]any), "integrity_check_planned")
		}},
		{name: "mount without cleanup", mutate: func(plan map[string]any) { delete(plan["steps"].([]any)[10].(map[string]any), "cleanup_planned") }},
		{name: "mount file without integrity", mutate: func(plan map[string]any) {
			delete(plan["steps"].([]any)[11].(map[string]any), "integrity_check_planned")
		}},
		{name: "deprecated combined tool", mutate: func(plan map[string]any) {
			plan["steps"].([]any)[0].(map[string]any)["operation"] = "jetkvm_virtual_media"
		}},
		{name: "unknown field", mutate: func(plan map[string]any) { plan["private_target_name"] = "PRIVATE-SENTINEL" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := completePlan()
			test.mutate(plan)
			var output bytes.Buffer
			if code := run([]string{"--plan", writePlan(t, plan)}, &output); code != 1 {
				t.Fatalf("run exit = %d, output = %q", code, output.String())
			}
			if strings.Contains(output.String(), "PRIVATE-SENTINEL") || strings.Contains(output.String(), "jetkvm_virtual_media") {
				t.Fatalf("sanitized report leaked plan content: %q", output.String())
			}
		})
	}
}

func TestCheckedExamplePlanAndChecklistCoverEveryMutationWithoutAuthorizingExecution(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	value, err := readPlan(filepath.Join(root, "testdata", "mutation-validation-plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePlan(value); err != nil {
		t.Fatalf("checked example plan is invalid: %v", err)
	}
	document, err := os.ReadFile(filepath.Join(root, "docs", "mutation-validation.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(document)
	requiredText := append(append([]string(nil), expectedOperations...),
		"No execution authority",
		"Designated expendable target",
		"Observer and emergency stop",
		"Outcome unknown: stop and never retry",
		"Sanitized evidence",
	)
	for _, required := range requiredText {
		if !strings.Contains(text, required) {
			t.Fatalf("mutation checklist missing %q", required)
		}
	}
}

func TestRunRejectsArgumentsMalformedAndTrailingJSON(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		data string
		want int
	}{
		{name: "missing plan", want: 2},
		{name: "extra argument", args: []string{"--plan", "unused", "extra"}, want: 2},
		{name: "malformed", data: `{`, want: 1},
		{name: "trailing", data: `{}` + `{}`, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := test.args
			if test.data != "" {
				path := filepath.Join(t.TempDir(), "plan.json")
				if err := os.WriteFile(path, []byte(test.data), 0o600); err != nil {
					t.Fatal(err)
				}
				args = []string{"--plan", path}
			}
			var output bytes.Buffer
			if code := run(args, &output); code != test.want {
				t.Fatalf("run exit = %d, want %d", code, test.want)
			}
		})
	}
}
