// jetkvm-mcp-fixture-runner executes bounded, operator-prescribed MCP call
// batches for the physical qualification runbook. It never supplies authority
// or physical/browser observations and emits only value-free call outcomes.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/toolresult"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	maxPlanBytes     = 1 << 20
	maxCalls         = 256
	maxCallsPerBatch = 8
	maxCallTimeout   = 60
)

type options struct {
	binary                       string
	config                       string
	plan                         string
	acknowledgeAuthorizedFixture bool
}

type prescribedPlan struct {
	Batches []prescribedBatch `json:"batches"`
}

type prescribedBatch struct {
	Calls []prescribedCall `json:"calls"`
}

type prescribedCall struct {
	Tool           string         `json:"tool"`
	Arguments      map[string]any `json:"arguments"`
	TimeoutSeconds int            `json:"timeout_seconds"`
}

type callRecord struct {
	Batch   int    `json:"batch"`
	Call    int    `json:"call"`
	Tool    string `json:"tool"`
	Result  string `json:"result"`
	Code    string `json:"code,omitempty"`
	Outcome string `json:"outcome,omitempty"`
}

type runReport struct {
	Result    string       `json:"result"`
	Completed int          `json:"completed"`
	Failed    string       `json:"failed,omitempty"`
	Calls     []callRecord `json:"calls,omitempty"`
}

type runnerSession interface {
	ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(context.Context, *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		emit(runReport{Result: "fail", Failed: "arguments"})
		os.Exit(2)
	}
	plan, err := loadPlan(opts.plan)
	if err != nil {
		emit(runReport{Result: "fail", Failed: "plan"})
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, planTimeout(plan))
	defer cancel()
	report := run(ctx, opts, plan)
	emit(report)
	if report.Result != "pass" {
		os.Exit(1)
	}
}

func emit(report runReport) { _ = json.NewEncoder(os.Stdout).Encode(report) }

func parseArgs(args []string) (options, error) {
	flags := flag.NewFlagSet("jetkvm-mcp-fixture-runner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	binary := flags.String("binary", "", "path to the jetkvm-mcp binary")
	config := flags.String("config", "", "path to its configuration")
	plan := flags.String("plan", "", "path to a prescribed fixture plan")
	acknowledge := flags.Bool("acknowledge-owner-authorized-fixture", false, "confirm separate owner authorization for consequential calls")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return options{}, errors.New("invalid arguments")
	}
	if strings.TrimSpace(*binary) == "" || strings.TrimSpace(*config) == "" || strings.TrimSpace(*plan) == "" {
		return options{}, errors.New("binary, config, and plan are required")
	}
	return options{binary: *binary, config: *config, plan: *plan, acknowledgeAuthorizedFixture: *acknowledge}, nil
}

func loadPlan(path string) (prescribedPlan, error) {
	file, err := os.Open(path)
	if err != nil {
		return prescribedPlan{}, errors.New("open plan")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxPlanBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxPlanBytes {
		return prescribedPlan{}, errors.New("read plan")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var plan prescribedPlan
	if err := decoder.Decode(&plan); err != nil {
		return prescribedPlan{}, errors.New("decode plan")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return prescribedPlan{}, errors.New("plan must contain one JSON value")
	}
	if err := validatePlan(plan); err != nil {
		return prescribedPlan{}, err
	}
	return plan, nil
}

func validatePlan(plan prescribedPlan) error {
	if len(plan.Batches) == 0 || len(plan.Batches) > maxCalls {
		return errors.New("invalid batch count")
	}
	total := 0
	for _, batch := range plan.Batches {
		if len(batch.Calls) == 0 || len(batch.Calls) > maxCallsPerBatch {
			return errors.New("invalid batch width")
		}
		total += len(batch.Calls)
		if total > maxCalls {
			return errors.New("too many calls")
		}
		for _, call := range batch.Calls {
			if strings.TrimSpace(call.Tool) == "" || call.TimeoutSeconds < 1 || call.TimeoutSeconds > maxCallTimeout {
				return errors.New("invalid call")
			}
		}
	}
	return nil
}

func planTimeout(plan prescribedPlan) time.Duration {
	total := 30 * time.Second
	for _, batch := range plan.Batches {
		longest := 0
		for _, call := range batch.Calls {
			if call.TimeoutSeconds > longest {
				longest = call.TimeoutSeconds
			}
		}
		total += time.Duration(longest) * time.Second
	}
	return total
}

func run(ctx context.Context, opts options, plan prescribedPlan) runReport {
	command := exec.CommandContext(ctx, opts.binary, "--config", opts.config)
	client := mcp.NewClient(&mcp.Implementation{Name: "jetkvm-mcp-fixture-runner", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		return runReport{Result: "fail", Failed: "connect"}
	}
	defer session.Close()
	return runPlan(ctx, session, plan, opts.acknowledgeAuthorizedFixture)
}

func runPlan(ctx context.Context, session runnerSession, plan prescribedPlan, acknowledge bool) runReport {
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return runReport{Result: "fail", Failed: "tools_list"}
	}
	tools := make(map[string]*mcp.Tool, len(listed.Tools))
	consequential := make(map[string]bool, len(listed.Tools))
	for _, tool := range listed.Tools {
		tools[tool.Name] = tool
	}
	for _, batch := range plan.Batches {
		for _, call := range batch.Calls {
			tool := tools[call.Tool]
			if tool == nil || tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || tool.Annotations.OpenWorldHint == nil {
				return runReport{Result: "fail", Failed: "tools"}
			}
			clearlyReadOnly := tool.Annotations.ReadOnlyHint && !*tool.Annotations.DestructiveHint && !*tool.Annotations.OpenWorldHint
			consequential[call.Tool] = !clearlyReadOnly
			if !clearlyReadOnly && !acknowledge {
				return runReport{Result: "fail", Failed: "authorization"}
			}
		}
	}

	report := runReport{Result: "pass", Calls: make([]callRecord, 0, maxCalls)}
	for batchIndex, batch := range plan.Batches {
		records := make([]callRecord, len(batch.Calls))
		var wait sync.WaitGroup
		for callIndex, call := range batch.Calls {
			wait.Add(1)
			go func() {
				defer wait.Done()
				callCtx, cancel := context.WithTimeout(ctx, time.Duration(call.TimeoutSeconds)*time.Second)
				defer cancel()
				arguments := call.Arguments
				if arguments == nil {
					arguments = map[string]any{}
				}
				result, callErr := session.CallTool(callCtx, &mcp.CallToolParams{Name: call.Tool, Arguments: arguments})
				records[callIndex] = classifyCall(batchIndex+1, callIndex+1, call.Tool, consequential[call.Tool], result, callErr)
			}()
		}
		wait.Wait()
		report.Calls = append(report.Calls, records...)
		report.Completed += len(records)
		failed := false
		for _, record := range records {
			failed = failed || record.Result != "pass"
		}
		if failed {
			report.Result = "fail"
			report.Failed = "call"
			break
		}
	}
	return report
}

func classifyCall(batch, call int, tool string, consequential bool, result *mcp.CallToolResult, err error) callRecord {
	record := callRecord{Batch: batch, Call: call, Tool: tool, Result: "pass"}
	if err != nil || result == nil {
		record.Result, record.Code, record.Outcome = "fail", "transport_error", "failed"
		if consequential {
			record.Outcome = "unknown"
		}
		return record
	}
	if !result.IsError {
		return record
	}
	record.Result, record.Code, record.Outcome = "fail", "tool_error", "failed"
	if len(result.Content) != 1 {
		return record
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return record
	}
	var failure struct {
		Code    string `json:"code"`
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal([]byte(text.Text), &failure) == nil && toolresult.ValidCode(failure.Code) && toolresult.ValidOutcome(failure.Outcome) {
		record.Code, record.Outcome = failure.Code, failure.Outcome
	}
	return record
}
