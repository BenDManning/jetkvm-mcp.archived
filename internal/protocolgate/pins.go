package protocolgate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
)

var (
	commitPattern    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	integrityPattern = regexp.MustCompile(`^sha512-[A-Za-z0-9+/]+={0,2}$`)
	versionPattern   = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
	namePattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(?:/[a-z0-9][a-z0-9-]*)?$`)
)

const (
	reviewedConformancePackage   = "@modelcontextprotocol/conformance"
	reviewedConformanceVersion   = "0.2.0-alpha.11"
	reviewedConformanceCommit    = "c321dd32035556e6769d3724a8ee97d87c3faaac"
	reviewedConformanceIntegrity = "sha512-imPK9tx5gQsL6ZKQq4MrsyDYfSaIwpRmX6+ogjbeAXs9LGvxkBxWcY7KcS7TvwaBk/ZiVWl6b/naF4q83UwDRA=="
	reviewedInspectorPackage     = "@modelcontextprotocol/inspector"
	reviewedInspectorVersion     = "2.2.0"
	reviewedInspectorCommit      = "672f9f41c548487a468b9e7007d2f9de14da5a69"
	reviewedInspectorIntegrity   = "sha512-IUyZsx6XzSr1Rl3xScqkOtqAwCsguI6kdc+/6rP0efKL/O22/5ougEkhVxudN7yIVGiSd49FZKhUk47/iMJRxA=="
	reviewedScenarioInventory    = "cd729ec47ce5ca74438fa0e83836f302d486182fda62f246ad2e76350eee1406"
	reviewedProtocolVersion      = "2026-07-28"
)

type Pins struct {
	SchemaVersion int               `json:"schemaVersion"`
	Conformance   ConformanceSource `json:"conformance"`
	Inspector     InspectorSource   `json:"inspector"`
}

type ConformanceSource struct {
	Package                 string                   `json:"package"`
	Version                 string                   `json:"version"`
	Commit                  string                   `json:"commit"`
	Integrity               string                   `json:"integrity"`
	SpecVersion             string                   `json:"specVersion"`
	ScenarioInventorySHA256 string                   `json:"serverScenarioInventorySHA256"`
	ServerScenarios         []ScenarioClassification `json:"serverScenarios"`
	ExpectedFailures        []string                 `json:"expectedFailures"`
}

type ScenarioClassification struct {
	Name                 string   `json:"name"`
	Disposition          string   `json:"disposition"`
	Reason               string   `json:"reason,omitempty"`
	Replacement          string   `json:"replacement,omitempty"`
	AllowedSkippedChecks []string `json:"allowedSkippedChecks,omitempty"`
}

type InspectorSource struct {
	Package          string   `json:"package"`
	Version          string   `json:"version"`
	Commit           string   `json:"commit"`
	Integrity        string   `json:"integrity"`
	Transports       []string `json:"transports"`
	Methods          []string `json:"methods"`
	FixtureSafeTools []string `json:"fixtureSafeTools"`
}

func LoadPins(filePath string) (Pins, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Pins{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var pins Pins
	if err := decoder.Decode(&pins); err != nil {
		return Pins{}, fmt.Errorf("decode MCP gate pins: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Pins{}, errors.New("decode MCP gate pins: trailing JSON value")
	}
	if err := pins.validate(); err != nil {
		return Pins{}, err
	}
	return pins, nil
}

func (pins Pins) validate() error {
	if pins.SchemaVersion != 1 {
		return errors.New("MCP gate pins: unsupported schema version")
	}
	if err := validateSource(pins.Conformance.Package, pins.Conformance.Version, pins.Conformance.Commit, pins.Conformance.Integrity); err != nil {
		return fmt.Errorf("MCP gate pins: conformance: %w", err)
	}
	if pins.Conformance.Package != reviewedConformancePackage || pins.Conformance.Version != reviewedConformanceVersion || pins.Conformance.Commit != reviewedConformanceCommit || pins.Conformance.Integrity != reviewedConformanceIntegrity {
		return errors.New("MCP gate pins: unreviewed conformance source")
	}
	if pins.Conformance.SpecVersion != reviewedProtocolVersion {
		return errors.New("MCP gate pins: unreviewed conformance specification version")
	}
	if pins.Conformance.ScenarioInventorySHA256 != reviewedScenarioInventory {
		return errors.New("MCP gate pins: unreviewed official scenario applicability inventory")
	}
	seen := make(map[string]string, len(pins.Conformance.ServerScenarios))
	for _, scenario := range pins.Conformance.ServerScenarios {
		if !namePattern.MatchString(scenario.Name) {
			return fmt.Errorf("MCP gate pins: invalid scenario name %q", scenario.Name)
		}
		if _, duplicate := seen[scenario.Name]; duplicate {
			return fmt.Errorf("MCP gate pins: duplicate scenario %q", scenario.Name)
		}
		seen[scenario.Name] = scenario.Disposition
		switch scenario.Disposition {
		case "gate":
			if scenario.Reason != "" || scenario.Replacement != "" {
				return fmt.Errorf("MCP gate pins: gated scenario %q has an exclusion", scenario.Name)
			}
			if err := validateUniqueOptionalValues("allowed skipped check", scenario.AllowedSkippedChecks); err != nil {
				return err
			}
		case "not_applicable":
			if strings.TrimSpace(scenario.Reason) == "" || strings.TrimSpace(scenario.Replacement) == "" || len(scenario.AllowedSkippedChecks) != 0 {
				return fmt.Errorf("MCP gate pins: scenario %q lacks a bounded exclusion", scenario.Name)
			}
		default:
			return fmt.Errorf("MCP gate pins: scenario %q has invalid disposition", scenario.Name)
		}
	}
	if len(seen) == 0 {
		return errors.New("MCP gate pins: no official scenarios classified")
	}
	if len(pins.Conformance.ExpectedFailures) != 0 {
		return errors.New("MCP gate pins: expected-failure baselines are not accepted")
	}
	if err := validateSource(pins.Inspector.Package, pins.Inspector.Version, pins.Inspector.Commit, pins.Inspector.Integrity); err != nil {
		return fmt.Errorf("MCP gate pins: inspector: %w", err)
	}
	if pins.Inspector.Package != reviewedInspectorPackage || pins.Inspector.Version != reviewedInspectorVersion || pins.Inspector.Commit != reviewedInspectorCommit || pins.Inspector.Integrity != reviewedInspectorIntegrity {
		return errors.New("MCP gate pins: unreviewed Inspector source")
	}
	if err := validateUniqueValues("inspector transport", pins.Inspector.Transports); err != nil {
		return err
	}
	if err := validateUniqueValues("inspector method", pins.Inspector.Methods); err != nil {
		return err
	}
	if err := validateUniqueValues("fixture-safe tool", pins.Inspector.FixtureSafeTools); err != nil {
		return err
	}
	if !slices.Equal(pins.Inspector.Transports, []string{"stdio", "http"}) ||
		!slices.Equal(pins.Inspector.Methods, []string{"initialize", "tools/list"}) ||
		!slices.Equal(pins.Inspector.FixtureSafeTools, []string{"jetkvm_list_devices"}) {
		return errors.New("MCP gate pins: unreviewed Inspector matrix")
	}
	return nil
}

func validateSource(packageName, version, commit, integrity string) error {
	if !strings.HasPrefix(packageName, "@modelcontextprotocol/") {
		return errors.New("package is not an official Model Context Protocol package")
	}
	if !versionPattern.MatchString(version) {
		return errors.New("version is not an exact semantic version")
	}
	if !commitPattern.MatchString(commit) {
		return errors.New("commit is not an exact source revision")
	}
	if !integrityPattern.MatchString(integrity) {
		return errors.New("integrity is not an exact SHA-512 npm artifact pin")
	}
	return nil
}

func validateUniqueValues(label string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("MCP gate pins: no %ss declared", label)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("MCP gate pins: empty %s", label)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("MCP gate pins: duplicate %s %q", label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateUniqueOptionalValues(label string, values []string) error {
	if len(values) == 0 {
		return nil
	}
	return validateUniqueValues(label, values)
}

func (pins Pins) GatedScenarios() []string {
	var scenarios []string
	for _, scenario := range pins.Conformance.ServerScenarios {
		if scenario.Disposition == "gate" {
			scenarios = append(scenarios, scenario.Name)
		}
	}
	return scenarios
}

func (pins Pins) Scenario(name string) (ScenarioClassification, bool) {
	for _, scenario := range pins.Conformance.ServerScenarios {
		if scenario.Name == name {
			return scenario, true
		}
	}
	return ScenarioClassification{}, false
}

func (pins Pins) ClassifiedScenarioNames() []string {
	names := make([]string, 0, len(pins.Conformance.ServerScenarios))
	for _, scenario := range pins.Conformance.ServerScenarios {
		names = append(names, scenario.Name)
	}
	return names
}

func (pins Pins) ValidateScenarioInventory(observed []string) error {
	expected := pins.ClassifiedScenarioNames()
	sort.Strings(expected)
	observedCopy := append([]string(nil), observed...)
	sort.Strings(observedCopy)
	if len(expected) != len(observedCopy) {
		return errors.New("official MCP scenario inventory differs from the classified pin")
	}
	for index := range expected {
		if expected[index] != observedCopy[index] {
			return errors.New("official MCP scenario inventory differs from the classified pin")
		}
	}
	return nil
}
