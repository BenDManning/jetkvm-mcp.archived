package jetkvm

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pion/webrtc/v4"
)

func TestDecodeSessionDescriptionRejectsInvalidAndZeroValues(t *testing.T) {
	for _, encoded := range []string{
		"***=",
		base64.StdEncoding.EncodeToString([]byte(`{}`)),
		base64.StdEncoding.EncodeToString([]byte(`null`)),
	} {
		if _, err := decodeSessionDescription(encoded); err == nil {
			t.Fatalf("decodeSessionDescription(%q) accepted", encoded)
		}
	}
}

func FuzzDecodeSessionDescription(f *testing.F) {
	for _, description := range []webrtc.SessionDescription{
		{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\n"},
		{Type: webrtc.SDPTypeAnswer, SDP: "v=0\r\na=inactive\r\n"},
	} {
		encoded, err := json.Marshal(description)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(base64.StdEncoding.EncodeToString(encoded))
	}
	f.Add("not-base64")
	f.Add(base64.StdEncoding.EncodeToString([]byte(`{"type":"answer","sdp":`)))
	f.Add(base64.StdEncoding.EncodeToString(make([]byte, maxSignalingFrame+1)))

	f.Fuzz(func(t *testing.T, encoded string) {
		maximumEncoded := base64.StdEncoding.EncodedLen(maxSignalingFrame + 1)
		if len(encoded) > maximumEncoded {
			encoded = encoded[:maximumEncoded]
		}
		description, err := decodeSessionDescription(encoded)
		if err != nil {
			return
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(raw) == 0 || len(raw) > maxSignalingFrame {
			t.Fatalf("accepted out-of-bounds session description")
		}
		canonical, err := json.Marshal(description)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := decodeSessionDescription(base64.StdEncoding.EncodeToString(canonical))
		if err != nil || roundTrip.Type != description.Type || roundTrip.SDP != description.SDP {
			t.Fatalf("session description did not round-trip: first=%#v second=%#v err=%v", description, roundTrip, err)
		}
	})
}
