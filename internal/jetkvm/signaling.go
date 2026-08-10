package jetkvm

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

const (
	maxSignalingFrame = 64 << 10
	signalingPath     = "/webrtc/signaling/client"
)

type signalingEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type offerEnvelope struct {
	Type string `json:"type"`
	Data struct {
		SD string `json:"sd"`
	} `json:"data"`
}

type candidateEnvelope struct {
	Type string                  `json:"type"`
	Data webrtc.ICECandidateInit `json:"data"`
}

func newOfferEnvelope(description webrtc.SessionDescription) (offerEnvelope, error) {
	encoded, err := json.Marshal(description)
	if err != nil {
		return offerEnvelope{}, err
	}
	if len(encoded) > maxSignalingFrame {
		return offerEnvelope{}, errors.New("session description too large")
	}
	envelope := offerEnvelope{Type: "offer"}
	envelope.Data.SD = base64.StdEncoding.EncodeToString(encoded)
	return envelope, nil
}

func newCandidateEnvelope(candidate webrtc.ICECandidateInit) candidateEnvelope {
	return candidateEnvelope{Type: "new-ice-candidate", Data: candidate}
}

func decodeSessionDescription(encoded string) (webrtc.SessionDescription, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) > maxSignalingFrame {
		return webrtc.SessionDescription{}, errors.New("invalid session description")
	}
	var description webrtc.SessionDescription
	if err := json.Unmarshal(raw, &description); err != nil {
		return webrtc.SessionDescription{}, errors.New("invalid session description")
	}
	return description, nil
}

func newPeerConnection(profile SessionProfile, onTrack func(*webrtc.TrackRemote, *webrtc.RTPReceiver)) (*webrtc.PeerConnection, error) {
	mediaEngine := &webrtc.MediaEngine{}
	interceptors := &interceptor.Registry{}
	if profile == SessionProfileVideo {
		feedback := []webrtc.RTCPFeedback{{Type: "goog-remb"}, {Type: "ccm", Parameter: "fir"}, {Type: "nack"}, {Type: "nack", Parameter: "pli"}}
		for _, codec := range []struct {
			payload webrtc.PayloadType
			profile string
		}{
			{102, "42001f"},
			{106, "42e01f"},
			{127, "4d001f"},
			{112, "64001f"},
		} {
			if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType: webrtc.MimeTypeH264, ClockRate: 90000,
					SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=" + codec.profile,
					RTCPFeedback: feedback,
				},
				PayloadType: codec.payload,
			}, webrtc.RTPCodecTypeVideo); err != nil {
				return nil, err
			}
		}
		if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptors); err != nil {
			return nil, err
		}
	}

	settings := webrtc.SettingEngine{}
	settings.SetIncludeLoopbackCandidate(true)
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptors),
		webrtc.WithSettingEngine(settings),
	)
	peer, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}
	if profile != SessionProfileVideo {
		return peer, nil
	}
	if onTrack != nil {
		peer.OnTrack(onTrack)
	}
	if _, err := peer.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		_ = peer.Close()
		return nil, err
	}
	return peer, nil
}
