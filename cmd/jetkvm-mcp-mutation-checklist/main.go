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
	ObserverReady          bool `json:"observer_ready"`
	RecoveryReady          bool `json:"recovery_ready"`
	EmergencyStopReady     bool `json:"emergency_stop_ready"`
	PerPlanAcknowledgement bool `json:"per_plan_acknowledgement"`
	ExecutionApproved      bool `json:"execution_approved"`
}

type step struct {
	Operation                string `json:"operation"`
	ConsequenceAcknowledged  bool   `json:"consequence_acknowledged"`
	PreconditionsConfirmed   bool   `json:"preconditions_confirmed"`
	PostconditionObservable  bool   `json:"postcondition_observable"`
	TimeoutSeconds           int    `json:"timeout_seconds"`
	RecoveryReady            bool   `json:"recovery_ready"`
	NeverRetryUnknownOutcome bool   `json:"never_retry_unknown_outcome"`
	IntegrityCheckPlanned    *bool  `json:"integrity_check_planned,omitempty"`
	CleanupPlanned           *bool  `json:"cleanup_planned,omitempty"`
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
	emit(output, report{Schema: reportSchema, Result: "pass", CheckedSteps: len(value.Steps)})
	return 0
}

func emit(output io.Writer, value report) {
	_ = json.NewEncoder(output).Encode(value)
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

func openPlanFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
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
		!value.Controls.PerPlanAcknowledgement || value.Controls.ExecutionApproved {
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
		if !validMediaControls(current) {
			return errors.New("invalid media controls")
		}
	}
	return nil
}

func validMediaControls(value step) bool {
	trueValue := func(pointer *bool) bool { return pointer != nil && *pointer }
	switch value.Operation {
	case "jetkvm_upload_virtual_media_file", "jetkvm_mount_virtual_media_file":
		return trueValue(value.IntegrityCheckPlanned) && trueValue(value.CleanupPlanned)
	case "jetkvm_mount_virtual_media_url", "jetkvm_unmount_virtual_media":
		return value.IntegrityCheckPlanned == nil && trueValue(value.CleanupPlanned)
	default:
		return value.IntegrityCheckPlanned == nil && value.CleanupPlanned == nil
	}
}
