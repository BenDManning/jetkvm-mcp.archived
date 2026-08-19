package toolresult

import "testing"

func TestStableTaxonomyIsClosed(t *testing.T) {
	for _, code := range []string{
		"operation_failed", "canceled", "timeout", "invalid_input", "busy",
		"authentication_failed", "device_unavailable", "video_unavailable",
		"no_signal", "protocol_error", "session_released", "session_taken_over",
		"ownership_uncertain",
	} {
		if !ValidCode(code) {
			t.Errorf("stable code %q rejected", code)
		}
	}
	if ValidCode("private_error") {
		t.Fatal("unreviewed code accepted")
	}
	for _, outcome := range []string{"not_sent", "failed", "unknown"} {
		if !ValidOutcome(outcome) {
			t.Errorf("stable outcome %q rejected", outcome)
		}
	}
	if ValidOutcome("completed") {
		t.Fatal("unreviewed outcome accepted")
	}
}
