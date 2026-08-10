package jetkvm

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

func TestConnectedSessionCapturesH264FromRealPionTrack(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connectedCtx, connectedCancel := context.WithCancel(context.Background())
	connected := &connectedSession{ctx: connectedCtx, cancel: connectedCancel, video: newVideoReceiver()}
	client, err := newPeerConnection(SessionProfileVideo, connected.handleTrack)
	if err != nil {
		t.Fatal(err)
	}
	connected.peer = client
	defer connected.Close()
	peerConnected := make(chan struct{})
	var peerConnectedOnce sync.Once
	client.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateConnected {
			peerConnectedOnce.Do(func() { close(peerConnected) })
		}
	})
	if _, err := client.CreateDataChannel("rpc", nil); err != nil {
		t.Fatal(err)
	}

	mediaEngine := &webrtc.MediaEngine{}
	codec := webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264, ClockRate: 90000,
		SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
	}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: codec, PayloadType: 102}, webrtc.RTPCodecTypeVideo); err != nil {
		t.Fatal(err)
	}
	remoteAPI := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	remote, err := remoteAPI.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()
	remote.OnDataChannel(func(*webrtc.DataChannel) {})
	track, err := webrtc.NewTrackLocalStaticRTP(codec, "video", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	sender, err := remote.AddTrack(track)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		buffer := make([]byte, 1500)
		for {
			if _, _, err := sender.Read(buffer); err != nil {
				return
			}
		}
	}()

	offer, err := client.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	clientGathered := webrtc.GatheringCompletePromise(client)
	if err := client.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-clientGathered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := remote.SetRemoteDescription(*client.LocalDescription()); err != nil {
		t.Fatal(err)
	}
	answer, err := remote.CreateAnswer(nil)
	if err != nil {
		t.Fatal(err)
	}
	remoteGathered := webrtc.GatheringCompletePromise(remote)
	if err := remote.SetLocalDescription(answer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-remoteGathered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := client.SetRemoteDescription(*remote.LocalDescription()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-peerConnected:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	sequence := uint16(0)
	timestamp := uint32(99)
	for {
		connected.video.mu.Lock()
		generation := connected.video.generation
		connected.video.mu.Unlock()
		if generation >= 2 {
			break
		}
		if err := track.WriteRTP(rtpPacket(sequence, timestamp, true, []byte{0x61, 0})); err != nil {
			t.Fatal(err)
		}
		if err := track.WriteRTP(rtpPacket(sequence+1, timestamp+1, true, stapA(
			[]byte{0x67, 0x42, 0x00, 0x1f}, []byte{0x68, 0xce, 0x06, 0xe2},
		))); err != nil {
			t.Fatal(err)
		}
		sequence += 2
		timestamp += 2
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}

	type capture struct {
		data []byte
		err  error
	}
	done := make(chan capture, 1)
	go func() {
		data, _, err := connected.CaptureH264(ctx)
		done <- capture{data: data, err: err}
	}()
	for !connected.video.Waiting() {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond):
		}
	}
	if err := track.WriteRTP(rtpPacket(sequence, timestamp, true, []byte{0x65, 0x88, 0x84})); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if !bytes.Equal(result.data, captureAnnexB(
		[]byte{0x67, 0x42, 0x00, 0x1f}, []byte{0x68, 0xce, 0x06, 0xe2}, []byte{0x65, 0x88, 0x84},
	)) {
		t.Fatalf("capture = %x", result.data)
	}
}
