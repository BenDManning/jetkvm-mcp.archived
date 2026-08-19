package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseArgsRequiresPlanInputsAndExplicitAcknowledgement(t *testing.T) {
	got, err := parseArgs([]string{"--binary", "/opt/jetkvm-mcp", "--config", "/run/config.yaml", "--plan", "/run/plan.json", "--acknowledge-owner-authorized-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if got.binary != "/opt/jetkvm-mcp" || got.config != "/run/config.yaml" || got.plan != "/run/plan.json" || !got.acknowledgeAuthorizedFixture {
		t.Fatalf("options = %#v", got)
	}
	for _, args := range [][]string{
		{"--config", "config.yaml", "--plan", "plan.json"},
		{"--binary", "jetkvm-mcp", "--plan", "plan.json"},
		{"--binary", "jetkvm-mcp", "--config", "config.yaml"},
		{"--binary", "jetkvm-mcp", "--config", "config.yaml", "--plan", "plan.json", "extra"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("parseArgs(%q) succeeded", args)
		}
	}
}

func TestValidatePlanBoundsCallsBatchesAndTimeouts(t *testing.T) {
	valid := prescribedPlan{Batches: []prescribedBatch{{Calls: []prescribedCall{{
		Tool: "jetkvm_get_status", Arguments: map[string]any{"device": "lab"}, TimeoutSeconds: 30,
	}}}}}
	if err := validatePlan(valid); err != nil {
		t.Fatal(err)
	}
	invalid := []prescribedPlan{
		{},
		{Batches: []prescribedBatch{{}}},
		{Batches: []prescribedBatch{{Calls: []prescribedCall{{Tool: "", TimeoutSeconds: 30}}}}},
		{Batches: []prescribedBatch{{Calls: []prescribedCall{{Tool: "jetkvm_get_status", TimeoutSeconds: 0}}}}},
		{Batches: []prescribedBatch{{Calls: []prescribedCall{{Tool: "jetkvm_get_status", TimeoutSeconds: 61}}}}},
	}
	tooWide := prescribedBatch{}
	for range maxCallsPerBatch + 1 {
		tooWide.Calls = append(tooWide.Calls, prescribedCall{Tool: "jetkvm_get_status", TimeoutSeconds: 1})
	}
	invalid = append(invalid, prescribedPlan{Batches: []prescribedBatch{tooWide}})
	tooMany := prescribedPlan{}
	for range maxCalls + 1 {
		tooMany.Batches = append(tooMany.Batches, prescribedBatch{Calls: []prescribedCall{{Tool: "jetkvm_get_status", TimeoutSeconds: 1}}})
	}
	invalid = append(invalid, tooMany)
	for index, plan := range invalid {
		if err := validatePlan(plan); err == nil {
			t.Fatalf("invalid plan %d accepted", index)
		}
	}
}

type fixtureRunnerSession struct {
	mu      sync.Mutex
	tools   []*mcp.Tool
	calls   []string
	results map[string]*mcp.CallToolResult
	errors  map[string]error
}

func (session *fixtureRunnerSession) ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{Tools: session.tools}, nil
}

func (session *fixtureRunnerSession) CallTool(_ context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	session.mu.Lock()
	session.calls = append(session.calls, params.Name)
	session.mu.Unlock()
	if err := session.errors[params.Name]; err != nil {
		return nil, err
	}
	if result := session.results[params.Name]; result != nil {
		return result, nil
	}
	return &mcp.CallToolResult{}, nil
}

func fixtureTool(name string, readOnly bool) *mcp.Tool {
	no := false
	return &mcp.Tool{Name: name, Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: readOnly, DestructiveHint: &no, IdempotentHint: readOnly, OpenWorldHint: &no,
	}}
}

func TestRunPlanRequiresAcknowledgementBeforeConsequentialDispatch(t *testing.T) {
	session := &fixtureRunnerSession{tools: []*mcp.Tool{
		fixtureTool("jetkvm_get_status", true), fixtureTool("jetkvm_take_over_session", false),
	}}
	plan := prescribedPlan{Batches: []prescribedBatch{{Calls: []prescribedCall{
		{Tool: "jetkvm_get_status", TimeoutSeconds: 1},
		{Tool: "jetkvm_take_over_session", TimeoutSeconds: 1},
	}}}}
	report := runPlan(context.Background(), session, plan, false)
	if report.Result != "fail" || report.Failed != "authorization" || len(session.calls) != 0 || len(report.Calls) != 0 {
		t.Fatalf("report=%#v calls=%v", report, session.calls)
	}

	report = runPlan(context.Background(), session, plan, true)
	if report.Result != "pass" || report.Completed != 2 || len(report.Calls) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if !reflect.DeepEqual(session.calls, []string{"jetkvm_take_over_session", "jetkvm_get_status"}) &&
		!reflect.DeepEqual(session.calls, []string{"jetkvm_get_status", "jetkvm_take_over_session"}) {
		t.Fatalf("calls = %v", session.calls)
	}
}

func TestRunPlanStopsAfterFailingBatchAndSanitizesErrors(t *testing.T) {
	const private = "PRIVATE-FIXTURE-SENTINEL"
	failureJSON := `{"version":"v1","code":"ownership_uncertain","outcome":"failed","retryable":false,"message":"` + private + `"}`
	session := &fixtureRunnerSession{
		tools: []*mcp.Tool{fixtureTool("jetkvm_get_status", true), fixtureTool("jetkvm_capture_screen", true)},
		results: map[string]*mcp.CallToolResult{"jetkvm_get_status": {
			IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: failureJSON}},
		}},
		errors: map[string]error{"jetkvm_capture_screen": errors.New(private)},
	}
	plan := prescribedPlan{Batches: []prescribedBatch{
		{Calls: []prescribedCall{{Tool: "jetkvm_get_status", Arguments: map[string]any{"device": private}, TimeoutSeconds: 1}}},
		{Calls: []prescribedCall{{Tool: "jetkvm_capture_screen", TimeoutSeconds: 1}}},
	}}
	report := runPlan(context.Background(), session, plan, false)
	if report.Result != "fail" || report.Completed != 1 || len(session.calls) != 1 || len(report.Calls) != 1 {
		t.Fatalf("report=%#v calls=%v", report, session.calls)
	}
	if report.Calls[0].Code != "ownership_uncertain" || report.Calls[0].Outcome != "failed" {
		t.Fatalf("call result = %#v", report.Calls[0])
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(private)) {
		t.Fatalf("report leaked private value: %s", encoded)
	}
}

func TestRunPlanTreatsConsequentialTransportFailureAsUnknown(t *testing.T) {
	session := &fixtureRunnerSession{
		tools:  []*mcp.Tool{fixtureTool("jetkvm_take_over_session", false)},
		errors: map[string]error{"jetkvm_take_over_session": errors.New("connection lost after dispatch")},
	}
	plan := prescribedPlan{Batches: []prescribedBatch{{Calls: []prescribedCall{{
		Tool: "jetkvm_take_over_session", TimeoutSeconds: 1,
	}}}}}
	report := runPlan(context.Background(), session, plan, true)
	if len(report.Calls) != 1 || report.Calls[0].Code != "transport_error" || report.Calls[0].Outcome != "unknown" {
		t.Fatalf("report = %#v", report)
	}
}
