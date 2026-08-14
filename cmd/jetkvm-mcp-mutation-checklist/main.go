// jetkvm-mcp-mutation-checklist validates an offline, dry-run-only mutation plan.
// It contains no hardware or mutation execution path.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

const (
	planSchema   = "jetkvm.mutation-validation.v1"
	reportSchema = "jetkvm.mutation-validation.report.v1"
	maxPlanBytes = 64 << 10
)

var requiredOperations = map[string]struct{}{
	"jetkvm_keyboard":                  {},
	"jetkvm_mouse":                     {},
	"jetkvm_press_host_power_button":   {},
	"jetkvm_press_host_reset_button":   {},
	"jetkvm_force_host_power_off":      {},
	"jetkvm_turn_host_dc_power_on":     {},
	"jetkvm_turn_host_dc_power_off":    {},
	"jetkvm_wake_host_lan":             {},
	"jetkvm_wake_host_usb":             {},
	"jetkvm_upload_virtual_media_file": {},
	"jetkvm_mount_virtual_media_url":   {},
	"jetkvm_mount_virtual_media_file":  {},
	"jetkvm_unmount_virtual_media":     {},
}

var allowedHIDOperations = map[string]map[string]struct{}{
	"jetkvm_keyboard": {
		"type_text": {},
		"press_key": {},
	},
	"jetkvm_mouse": {
		"move_absolute": {},
		"move_relative": {},
		"click":         {},
		"scroll":        {},
	},
}

var allowedPlanMemberNames = map[string]struct{}{
	"schema":                      {},
	"mode":                        {},
	"target":                      {},
	"controls":                    {},
	"steps":                       {},
	"marked_expendable":           {},
	"identity_confirmed":          {},
	"non_production":              {},
	"observer_ready":              {},
	"recovery_ready":              {},
	"emergency_stop_ready":        {},
	"per_plan_acknowledgement":    {},
	"execution_approved":          {},
	"operation":                   {},
	"hid_operation":               {},
	"consequence_acknowledged":    {},
	"preconditions_confirmed":     {},
	"postcondition_observable":    {},
	"timeout_seconds":             {},
	"never_retry_unknown_outcome": {},
	"integrity_check_planned":     {},
	"cleanup_planned":             {},
}

type plan struct {
	Schema   string   `json:"schema"`
	Mode     string   `json:"mode"`
	Target   target   `json:"target"`
	Controls controls `json:"controls"`
	Steps    []step   `json:"steps"`
}

type target struct {
	MarkedExpendable  bool `json:"marked_expendable"`
	IdentityConfirmed bool `json:"identity_confirmed"`
	NonProduction     bool `json:"non_production"`
}

type controls struct {
	ObserverReady          bool         `json:"observer_ready"`
	RecoveryReady          bool         `json:"recovery_ready"`
	EmergencyStopReady     bool         `json:"emergency_stop_ready"`
	PerPlanAcknowledgement bool         `json:"per_plan_acknowledgement"`
	ExecutionApproved      optionalBool `json:"execution_approved"`
}

type optionalBool struct {
	Value   bool
	Present bool
}

func (value *optionalBool) UnmarshalJSON(data []byte) error {
	*value = optionalBool{}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null boolean")
	}
	if err := json.Unmarshal(data, &value.Value); err != nil {
		return err
	}
	value.Present = true
	return nil
}

type optionalString struct {
	Value   string
	Present bool
}

func (value *optionalString) UnmarshalJSON(data []byte) error {
	*value = optionalString{}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null string")
	}
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = decoded
	value.Present = true
	return nil
}

type step struct {
	Operation                string         `json:"operation"`
	HIDOperation             optionalString `json:"hid_operation,omitempty"`
	ConsequenceAcknowledged  bool           `json:"consequence_acknowledged"`
	PreconditionsConfirmed   bool           `json:"preconditions_confirmed"`
	PostconditionObservable  bool           `json:"postcondition_observable"`
	TimeoutSeconds           int            `json:"timeout_seconds"`
	RecoveryReady            bool           `json:"recovery_ready"`
	NeverRetryUnknownOutcome bool           `json:"never_retry_unknown_outcome"`
	IntegrityCheckPlanned    optionalBool   `json:"integrity_check_planned,omitempty"`
	CleanupPlanned           optionalBool   `json:"cleanup_planned,omitempty"`
}

type report struct {
	Schema              string `json:"schema"`
	Result              string `json:"result"`
	CheckedSteps        int    `json:"checked_steps"`
	ExecutionAuthorized bool   `json:"execution_authorized"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, output io.Writer) int {
	flags := flag.NewFlagSet("jetkvm-mcp-mutation-checklist", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("plan", "", "path to a dry-run mutation-validation plan")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *path == "" {
		emit(output, report{Schema: reportSchema, Result: "fail"})
		return 2
	}

	value, err := readPlan(*path)
	if err != nil || validatePlan(value) != nil {
		emit(output, report{Schema: reportSchema, Result: "fail"})
		return 1
	}
	if err := emit(output, report{Schema: reportSchema, Result: "pass", CheckedSteps: len(value.Steps)}); err != nil {
		return 1
	}
	return 0
}

func emit(output io.Writer, value report) error {
	if file, ok := output.(*os.File); ok {
		outputInfo, outputErr := file.Stat()
		nullInfo, nullErr := os.Stat(os.DevNull)
		if outputErr == nil && nullErr == nil && os.SameFile(outputInfo, nullInfo) {
			return errors.New("report output unavailable")
		}
	}
	return json.NewEncoder(output).Encode(value)
}

func readPlan(path string) (plan, error) {
	file, err := openPlanFile(path)
	if err != nil {
		return plan{}, errors.New("invalid plan input")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxPlanBytes {
		return plan{}, errors.New("invalid plan input")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPlanBytes+1))
	if err != nil || len(data) > maxPlanBytes {
		return plan{}, errors.New("invalid plan input")
	}
	if err := rejectUnsafeJSONMembers(data); err != nil {
		return plan{}, errors.New("invalid plan input")
	}
	var value plan
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return plan{}, errors.New("invalid plan input")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return plan{}, errors.New("invalid plan input")
	}
	return value, nil
}

func rejectUnsafeJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("invalid JSON input")
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object member")
			}
			if _, duplicate := seen[key]; duplicate {
				return errors.New("duplicate JSON object member")
			}
			if _, allowed := allowedPlanMemberNames[key]; !allowed {
				return errors.New("unknown JSON object member")
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("invalid JSON object")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("invalid JSON array")
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	return nil
}

func openPlanFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file == nil {
		_ = unix.Close(descriptor)
		return nil, errors.New("invalid plan input")
	}
	return file, nil
}

func validatePlan(value plan) error {
	if value.Schema != planSchema || value.Mode != "dry_run" {
		return errors.New("invalid plan")
	}
	if !value.Target.MarkedExpendable || !value.Target.IdentityConfirmed || !value.Target.NonProduction {
		return errors.New("invalid target controls")
	}
	if !value.Controls.ObserverReady || !value.Controls.RecoveryReady || !value.Controls.EmergencyStopReady ||
		!value.Controls.PerPlanAcknowledgement || !value.Controls.ExecutionApproved.Present || value.Controls.ExecutionApproved.Value {
		return errors.New("invalid execution controls")
	}
	if len(value.Steps) != len(requiredOperations) {
		return errors.New("incomplete mutation inventory")
	}
	seen := make(map[string]struct{}, len(value.Steps))
	for _, current := range value.Steps {
		if _, required := requiredOperations[current.Operation]; !required {
			return errors.New("unknown mutation operation")
		}
		if _, duplicate := seen[current.Operation]; duplicate {
			return errors.New("duplicate mutation operation")
		}
		seen[current.Operation] = struct{}{}
		if !current.ConsequenceAcknowledged || !current.PreconditionsConfirmed || !current.PostconditionObservable ||
			!current.RecoveryReady || !current.NeverRetryUnknownOutcome || current.TimeoutSeconds < 1 || current.TimeoutSeconds > 300 {
			return errors.New("incomplete mutation step")
		}
		if !validHIDOperation(current) {
			return errors.New("invalid HID operation")
		}
		if !validMediaControls(current) {
			return errors.New("invalid media controls")
		}
	}
	return nil
}

func validHIDOperation(value step) bool {
	allowed, isHID := allowedHIDOperations[value.Operation]
	if !isHID {
		return !value.HIDOperation.Present
	}
	if !value.HIDOperation.Present {
		return false
	}
	_, valid := allowed[value.HIDOperation.Value]
	return valid
}

func validMediaControls(value step) bool {
	trueValue := func(control optionalBool) bool { return control.Present && control.Value }
	switch value.Operation {
	case "jetkvm_upload_virtual_media_file", "jetkvm_mount_virtual_media_file":
		return trueValue(value.IntegrityCheckPlanned) && trueValue(value.CleanupPlanned)
	case "jetkvm_mount_virtual_media_url", "jetkvm_unmount_virtual_media":
		return !value.IntegrityCheckPlanned.Present && trueValue(value.CleanupPlanned)
	default:
		return !value.IntegrityCheckPlanned.Present && !value.CleanupPlanned.Present
	}
}
