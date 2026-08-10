package jetkvm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/pion/webrtc/v4"
)

func TestSignalingWireFormatAndBounds(t *testing.T) {
	offer, err := newOfferEnvelope(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\n"})
	if err != nil {
		t.Fatal(err)
	}
	encodedOffer, err := json.Marshal(offer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encodedOffer, []byte(`"type":"offer"`)) || !bytes.Contains(encodedOffer, []byte(`"sd":`)) {
		t.Fatalf("offer = %s", encodedOffer)
	}

	candidate := newCandidateEnvelope(webrtc.ICECandidateInit{Candidate: "candidate:1 1 UDP 1 127.0.0.1 9 typ host"})
	if candidate.Type != "new-ice-candidate" || candidate.Data.Candidate == "" {
		t.Fatalf("candidate = %#v", candidate)
	}

	answerJSON, _ := json.Marshal(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: "v=0\r\n"})
	answer, err := decodeSessionDescription(base64.StdEncoding.EncodeToString(answerJSON))
	if err != nil || answer.Type != webrtc.SDPTypeAnswer || answer.SDP != "v=0\r\n" {
		t.Fatalf("answer=%#v err=%v", answer, err)
	}
	for _, invalid := range []string{"not-base64", base64.StdEncoding.EncodeToString(make([]byte, maxSignalingFrame+1))} {
		if _, err := decodeSessionDescription(invalid); err == nil {
			t.Fatalf("decodeSessionDescription(%d bytes) error = nil", len(invalid))
		}
	}
}

func TestPeerProfileControlsVideoOffer(t *testing.T) {
	dataPeer, err := newPeerConnection(SessionProfileData, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dataPeer.Close()
	if _, err := dataPeer.CreateDataChannel("rpc", nil); err != nil {
		t.Fatal(err)
	}
	dataOffer, err := dataPeer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dataOffer.SDP, "m=video") {
		t.Fatalf("data-only offer contains video:\n%s", dataOffer.SDP)
	}

	videoPeer, err := newPeerConnection(SessionProfileVideo, func(*webrtc.TrackRemote, *webrtc.RTPReceiver) {})
	if err != nil {
		t.Fatal(err)
	}
	defer videoPeer.Close()
	if _, err := videoPeer.CreateDataChannel("rpc", nil); err != nil {
		t.Fatal(err)
	}
	videoOffer, err := videoPeer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(videoOffer.SDP, "m=video") || !strings.Contains(strings.ToUpper(videoOffer.SDP), "H264/90000") || !strings.Contains(videoOffer.SDP, "packetization-mode=1") {
		t.Fatalf("video offer missing required H264 mode:\n%s", videoOffer.SDP)
	}
}

func TestRPCRequestAndResponseWireRules(t *testing.T) {
	payload, err := marshalRPCRequest(7, "setDCPowerState", map[string]any{"enabled": true})
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal([]byte(payload), &request); err != nil {
		t.Fatal(err)
	}
	if request["jsonrpc"] != "2.0" || request["method"] != "setDCPowerState" || request["id"] != float64(7) {
		t.Fatalf("request = %#v", request)
	}

	for _, test := range []struct {
		name       string
		wire       string
		wantID     uint64
		wantResult string
		wantError  error
	}{
		{name: "result", wire: `{"jsonrpc":"2.0","id":7,"result":{"ready":true}}`, wantID: 7, wantResult: `{"ready":true}`},
		{name: "mutation omission", wire: `{"jsonrpc":"2.0","id":8}`, wantID: 8},
		{name: "protocol error", wire: `{"jsonrpc":"2.0","id":9,"error":{"code":-32601,"message":"private text"}}`, wantID: 9, wantError: ErrRPCMethodUnavailable},
		{name: "both result and error", wire: `{"jsonrpc":"2.0","id":10,"result":null,"error":{"code":1,"message":"x"}}`, wantError: ErrInvalidResponse},
		{name: "unsolicited method", wire: `{"jsonrpc":"2.0","method":"event","params":{}}`, wantError: ErrUnsolicitedRPC},
		{name: "duplicate id", wire: `{"jsonrpc":"2.0","id":1,"id":2,"result":null}`, wantError: ErrInvalidResponse},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := decodeRPCResponse([]byte(test.wire))
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err != nil {
				if strings.Contains(err.Error(), "private text") {
					t.Fatalf("error leaked firmware message: %v", err)
				}
				return
			}
			if response.ID != test.wantID || string(response.Result) != test.wantResult {
				t.Fatalf("response = %#v", response)
			}
		})
	}
}
