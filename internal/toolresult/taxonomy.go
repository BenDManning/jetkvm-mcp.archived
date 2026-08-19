// Package toolresult owns the closed public MCP tool-result taxonomy shared by
// server result construction and sanitized supporting-evidence tooling.
package toolresult

// ValidCode reports whether code is part of the reviewed stable error surface.
func ValidCode(code string) bool {
	switch code {
	case "operation_failed", "canceled", "timeout", "invalid_input", "busy", "authentication_failed", "device_unavailable", "video_unavailable", "no_signal", "protocol_error", "session_released", "session_taken_over", "ownership_uncertain":
		return true
	default:
		return false
	}
}

// ValidOutcome reports whether outcome is part of the dispatch-evidence surface.
func ValidOutcome(outcome string) bool {
	return outcome == "not_sent" || outcome == "failed" || outcome == "unknown"
}
