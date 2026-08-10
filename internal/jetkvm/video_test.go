package jetkvm

import (
	"bytes"
	"testing"
	"time"

	"github.com/pion/rtp"
)

func TestH264AccessUnitPacketizationAndValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		packets []*rtp.Packet
		want    []byte
	}{
		{
			name: "single NAL",
			packets: []*rtp.Packet{
				rtpPacket(1, 1, true, []byte{0x65, 0xaa}),
			},
			want: annexB([]byte{0x65, 0xaa}),
		},
		{
			name: "STAP-A",
			packets: []*rtp.Packet{
				rtpPacket(2, 2, true, stapA([]byte{0x67, 1}, []byte{0x68, 2}, []byte{0x65, 3})),
			},
			want: joinAnnexB([]byte{0x67, 1}, []byte{0x68, 2}, []byte{0x65, 3}),
		},
		{
			name: "FU-A",
			packets: []*rtp.Packet{
				rtpPacket(3, 3, false, []byte{0x7c, 0x85, 0xaa}),
				rtpPacket(4, 3, false, []byte{0x7c, 0x05, 0xbb}),
				rtpPacket(5, 3, true, []byte{0x7c, 0x45, 0xcc}),
			},
			want: annexB([]byte{0x65, 0xaa, 0xbb, 0xcc}),
		},
		{
			name: "sequence wrap",
			packets: []*rtp.Packet{
				rtpPacket(65535, 4, false, []byte{0x7c, 0x85, 0xaa}),
				rtpPacket(0, 4, true, []byte{0x7c, 0x45, 0xbb}),
			},
			want: annexB([]byte{0x65, 0xaa, 0xbb}),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			unit, err := depacketizeH264AccessUnit(test.packets)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(unit.annexB, test.want) || !unit.hasIDR {
				t.Fatalf("unit = %x IDR=%v, want %x", unit.annexB, unit.hasIDR, test.want)
			}
		})
	}

	for _, test := range []struct {
		name    string
		packets []*rtp.Packet
	}{
		{name: "forbidden bit", packets: []*rtp.Packet{rtpPacket(1, 1, true, []byte{0xe5, 1})}},
		{name: "reserved type", packets: []*rtp.Packet{rtpPacket(1, 1, true, []byte{0x71, 1})}},
		{name: "truncated STAP-A", packets: []*rtp.Packet{rtpPacket(1, 1, true, []byte{0x78, 0, 2, 0x67})}},
		{name: "nested STAP-A", packets: []*rtp.Packet{rtpPacket(1, 1, true, stapA([]byte{0x78, 1}))}},
		{name: "FU-A without start", packets: []*rtp.Packet{rtpPacket(1, 1, true, []byte{0x7c, 0x45, 1})}},
		{name: "FU-A changed type", packets: []*rtp.Packet{
			rtpPacket(1, 1, false, []byte{0x7c, 0x85, 1}),
			rtpPacket(2, 1, true, []byte{0x7c, 0x47, 2}),
		}},
		{name: "sequence gap", packets: []*rtp.Packet{
			rtpPacket(1, 1, false, []byte{0x7c, 0x85, 1}),
			rtpPacket(3, 1, true, []byte{0x7c, 0x45, 2}),
		}},
		{name: "embedded Annex-B", packets: []*rtp.Packet{rtpPacket(1, 1, true, []byte{0x65, 0, 0, 1, 0x61})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := depacketizeH264AccessUnit(test.packets); err == nil {
				t.Fatal("malformed access unit was accepted")
			}
		})
	}
}

func TestH264StreamRetainsParametersAndRejectsStraddlingFrames(t *testing.T) {
	stream := newH264Stream()
	now := time.Unix(1, 0)
	drainAnchored(stream, rtpPacket(1, 1, true, stapA([]byte{0x67, 1}, []byte{0x68, 2})), 1, now)

	stream.push(rtpPacket(11, 2, false, []byte{0x7c, 0x05, 0xbb}), 2, 0, now)
	watermark := uint64(2)
	stream.push(rtpPacket(10, 2, false, []byte{0x7c, 0x85, 0xaa}), 3, watermark, now)
	if data, ok := stream.push(rtpPacket(12, 2, true, []byte{0x7c, 0x45, 0xcc}), 4, watermark, now); ok || data != nil {
		t.Fatal("straddling reordered FU-A was accepted")
	}

	data, ok := stream.push(rtpPacket(13, 3, true, []byte{0x65, 0xdd}), 5, watermark, now)
	want := joinAnnexB([]byte{0x67, 1}, []byte{0x68, 2}, []byte{0x65, 0xdd})
	if !ok || !bytes.Equal(data, want) {
		t.Fatalf("fresh candidate = %x, %v, want %x", data, ok, want)
	}
}

func TestH264StreamRecoversAfterLossAndSequenceWrap(t *testing.T) {
	for _, test := range []struct {
		name                       string
		anchor, rejected, recovery uint16
	}{
		{name: "ordinary", anchor: 1, rejected: 70, recovery: 71},
		{name: "sequence wrap", anchor: 65500, rejected: 33, recovery: 34},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream := newH264Stream()
			now := time.Unix(1, 0)
			drainAnchored(stream, rtpPacket(test.anchor, 1, true, stapA([]byte{0x67, 1}, []byte{0x68, 2}, []byte{0x61, 3})), 1, now)
			if data, ok := stream.push(rtpPacket(test.rejected, 2, true, []byte{0x65, 4}), 2, 1, now); ok || data != nil {
				t.Fatal("unit beyond reorder window was accepted")
			}
			if data, ok := stream.push(rtpPacket(test.recovery, 3, true, []byte{0x65, 5}), 3, 1, now); !ok || len(data) == 0 {
				t.Fatal("stream did not recover after discarded unit")
			}
		})
	}
}

func rtpPacket(sequence uint16, timestamp uint32, marker bool, payload []byte) *rtp.Packet {
	return &rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: sequence, Timestamp: timestamp, Marker: marker}, Payload: payload}
}

func annexB(nalu []byte) []byte {
	return append([]byte{0, 0, 0, 1}, nalu...)
}

func joinAnnexB(nalus ...[]byte) []byte {
	var result []byte
	for _, nalu := range nalus {
		result = append(result, annexB(nalu)...)
	}
	return result
}

func stapA(nalus ...[]byte) []byte {
	payload := []byte{0x78}
	for _, nalu := range nalus {
		payload = append(payload, byte(len(nalu)>>8), byte(len(nalu)))
		payload = append(payload, nalu...)
	}
	return payload
}

func drainAnchored(stream *h264Stream, packet *rtp.Packet, generation uint64, observedAt time.Time) {
	stream.drain(rtpPacket(packet.SequenceNumber-1, packet.Timestamp-1, true, []byte{0x61, 0}), generation, observedAt.Add(-time.Nanosecond))
	stream.drain(packet, generation+1, observedAt)
}
