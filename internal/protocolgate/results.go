package protocolgate

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var officialScenarioLine = regexp.MustCompile(`^\s+- ([a-z0-9][a-z0-9-]*) \[(extension|[0-9]{4}-[0-9]{2}-[0-9]{2}(?:,[0-9]{4}-[0-9]{2}-[0-9]{2})*)\]\s*$`)

var (
	conformanceSummaryLine = regexp.MustCompile(`(?m)^Passed: ([0-9]+)/([0-9]+), 0 failed, 0 warnings[ 	]*$`)
	conformanceSkippedLine = regexp.MustCompile(`^\S+ \[([^]]+)\]\s+SKIPPED .+$`)
	ansiSGRLine            = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

const officialScenarioTitle = "Server scenarios (test against a server):"

func ParseOfficialServerScenarioList(output string) ([]string, error) {
	scenarios, _, err := parseOfficialServerScenarioInventory(output)
	return scenarios, err
}

func OfficialServerScenarioInventoryDigest(output string) (string, error) {
	_, canonical, err := parseOfficialServerScenarioInventory(output)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(strings.Join(canonical, "\n")))
	return fmt.Sprintf("%x", digest), nil
}

func parseOfficialServerScenarioInventory(output string) ([]string, []string, error) {
	var scenarios []string
	var canonical []string
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(output))
	titleSeen := false
	for scanner.Scan() {
		line := scanner.Text()
		if !titleSeen {
			if line != officialScenarioTitle {
				return nil, nil, errors.New("official MCP scenario inventory title was not recognized")
			}
			titleSeen = true
			continue
		}
		match := officialScenarioLine.FindStringSubmatch(line)
		if len(match) != 3 {
			return nil, nil, errors.New("official MCP scenario inventory contains an unrecognized line")
		}
		if _, duplicate := seen[match[1]]; duplicate {
			return nil, nil, errors.New("official MCP scenario inventory contains a duplicate")
		}
		seen[match[1]] = struct{}{}
		scenarios = append(scenarios, match[1])
		canonical = append(canonical, match[1]+"["+match[2]+"]")
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, errors.New("read official MCP scenario inventory")
	}
	if !titleSeen || len(scenarios) == 0 {
		return nil, nil, errors.New("official MCP scenario inventory was not recognized")
	}
	return scenarios, canonical, nil
}

func ValidateConformanceScenarioResult(output string, allowedSkippedChecks []string) error {
	output = ansiSGRLine.ReplaceAllString(output, "")
	if strings.Contains(output, "SKIPPED: scenario '") {
		return errors.New("gated MCP scenario was skipped")
	}
	matches := conformanceSummaryLine.FindAllStringSubmatch(output, -1)
	if len(matches) != 1 {
		return errors.New("MCP conformance result summary was not recognized")
	}
	match := matches[0]
	passed, passedErr := strconv.Atoi(match[1])
	total, totalErr := strconv.Atoi(match[2])
	if passedErr != nil || totalErr != nil || passed == 0 || passed != total {
		return errors.New("MCP conformance result did not pass every executed check")
	}
	var observed []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "Passed:") && conformanceSummaryLine.FindString(line) == "" {
			return errors.New("MCP conformance result contains an unrecognized summary")
		}
		if strings.Contains(line, "SKIPPED") {
			skip := conformanceSkippedLine.FindStringSubmatch(line)
			if len(skip) != 2 {
				return errors.New("MCP conformance result contains an unrecognized skipped check")
			}
			observed = append(observed, strings.TrimSpace(skip[1]))
		}
		if strings.Contains(line, " WARNING") || strings.Contains(line, " FAILED") || strings.Contains(line, " FAILURE") {
			return errors.New("MCP conformance result contains an unrecognized failure or warning")
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.New("read MCP conformance result")
	}
	slices.Sort(observed)
	expected := append([]string(nil), allowedSkippedChecks...)
	slices.Sort(expected)
	if !slices.Equal(observed, expected) {
		return errors.New("MCP conformance skipped-check set differs from the reviewed pin")
	}
	return nil
}

func ValidateInspectorResult(method string, output []byte, fixtureSafeTool string) error {
	decoder := json.NewDecoder(bytes.NewReader(output))
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := decoder.Decode(&envelope); err != nil || len(envelope.Result) == 0 {
		return errors.New("Inspector did not return a JSON result envelope")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("Inspector returned more than one JSON value")
	}

	switch method {
	case "initialize":
		var result struct {
			ServerInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			ProtocolVersion string         `json:"protocolVersion"`
			Capabilities    map[string]any `json:"capabilities"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil || result.ServerInfo.Name == "" || result.ServerInfo.Version == "" || result.ProtocolVersion == "" || result.Capabilities == nil {
			return errors.New("Inspector initialize result lacks required MCP fields")
		}
		return nil
	case "tools/list":
		var result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil || len(result.Tools) == 0 {
			return errors.New("Inspector tools/list result lacks tools")
		}
		for _, tool := range result.Tools {
			if tool.Name == fixtureSafeTool {
				return nil
			}
		}
		return errors.New("Inspector tools/list result lacks the fixture-safe tool")
	case "tools/call:" + fixtureSafeTool:
		var result struct {
			Content           []json.RawMessage `json:"content"`
			StructuredContent struct {
				Devices []json.RawMessage `json:"devices"`
			} `json:"structuredContent"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil || len(result.Content) == 0 || result.StructuredContent.Devices == nil {
			return errors.New("Inspector fixture-safe tool result lacks typed content")
		}
		return nil
	default:
		return errors.New("Inspector method is outside the reviewed gate")
	}
}

func CanonicalInspectorResult(output []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(output))
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := decoder.Decode(&envelope); err != nil {
		return nil, errors.New("Inspector did not return JSON")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.New("Inspector returned more than one JSON value")
	}
	if len(envelope.Result) == 0 || bytes.Equal(envelope.Result, []byte("null")) {
		return nil, errors.New("Inspector result is missing")
	}
	var result any
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		return nil, errors.New("Inspector result is invalid")
	}
	canonical, err := json.Marshal(result)
	if err != nil {
		return nil, errors.New("Inspector result cannot be canonicalized")
	}
	return canonical, nil
}
