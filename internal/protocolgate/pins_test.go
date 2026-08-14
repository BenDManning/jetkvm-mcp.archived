package protocolgate

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheckedInPinsClassifyEveryOfficialScenarioWithoutExpectedFailures(t *testing.T) {
	pins, err := LoadPins(filepath.Join("..", "..", "testdata", "mcp-gates", "pins.json"))
	if err != nil {
		t.Fatal(err)
	}
	if pins.Conformance.Package != "@modelcontextprotocol/conformance" || pins.Conformance.Version != "0.2.0-alpha.11" || pins.Conformance.Commit != "c321dd32035556e6769d3724a8ee97d87c3faaac" || pins.Conformance.Integrity != "sha512-imPK9tx5gQsL6ZKQq4MrsyDYfSaIwpRmX6+ogjbeAXs9LGvxkBxWcY7KcS7TvwaBk/ZiVWl6b/naF4q83UwDRA==" || pins.Conformance.SpecVersion != "2026-07-28" || pins.Conformance.ScenarioInventorySHA256 != reviewedScenarioInventory {
		t.Fatalf("conformance source = %#v", pins.Conformance)
	}
	if pins.Inspector.Package != "@modelcontextprotocol/inspector" || pins.Inspector.Version != "2.2.0" || pins.Inspector.Commit != "672f9f41c548487a468b9e7007d2f9de14da5a69" {
		t.Fatalf("inspector source = %#v", pins.Inspector)
	}
	if len(pins.Conformance.ExpectedFailures) != 0 {
		t.Fatalf("expected failures = %v", pins.Conformance.ExpectedFailures)
	}
	if got := pins.GatedScenarios(); !reflect.DeepEqual(got, []string{
		"tools-list", "dns-rebinding-protection", "caching", "http-header-validation",
	}) {
		t.Fatalf("gated scenarios = %v", got)
	}
	if !reflect.DeepEqual(pins.Inspector.Transports, []string{"stdio", "http"}) ||
		!reflect.DeepEqual(pins.Inspector.Methods, []string{"initialize", "tools/list"}) ||
		!reflect.DeepEqual(pins.Inspector.FixtureSafeTools, []string{"jetkvm_list_devices"}) {
		t.Fatalf("inspector matrix = %#v", pins.Inspector)
	}
}

func TestPinsRejectExpectedFailureBaseline(t *testing.T) {
	pins, err := LoadPins(filepath.Join("..", "..", "testdata", "mcp-gates", "pins.json"))
	if err != nil {
		t.Fatal(err)
	}
	pins.Conformance.ExpectedFailures = []string{"ping"}
	if err := pins.validate(); err == nil {
		t.Fatal("expected non-empty failure baseline to be rejected")
	}
}

func TestPinsRejectReviewedSourceCommitDrift(t *testing.T) {
	pins, err := LoadPins(filepath.Join("..", "..", "testdata", "mcp-gates", "pins.json"))
	if err != nil {
		t.Fatal(err)
	}
	pins.Conformance.Commit = "0000000000000000000000000000000000000000"
	pins.Inspector.Commit = "1111111111111111111111111111111111111111"
	if err := pins.validate(); err == nil {
		t.Fatal("coordinated reviewed source commit drift was accepted")
	}
}

func TestValidateScenarioInventoryRejectsUnexplainedAndStaleEntries(t *testing.T) {
	pins, err := LoadPins(filepath.Join("..", "..", "testdata", "mcp-gates", "pins.json"))
	if err != nil {
		t.Fatal(err)
	}
	inventory := pins.ClassifiedScenarioNames()
	if err := pins.ValidateScenarioInventory(inventory); err != nil {
		t.Fatal(err)
	}
	if err := pins.ValidateScenarioInventory(inventory[1:]); err == nil {
		t.Fatal("missing official scenario accepted")
	}
	if err := pins.ValidateScenarioInventory(append(inventory, "new-upstream-scenario")); err == nil {
		t.Fatal("unexplained official scenario accepted")
	}
}
