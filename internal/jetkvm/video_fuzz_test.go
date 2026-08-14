package jetkvm

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/pion/rtp"
)

const (
	maxFuzzH264Input   = 64 << 10
	maxFuzzH264Packets = 16
)

func FuzzDepacketizeH264AccessUnit(f *testing.F) {
	for _, packets := range [][][]byte{
		{{0x65, 0xaa}},
		{{0x78, 0, 2, 0x67, 1, 0, 2, 0x68, 2, 0, 2, 0x65, 3}},
		{{0x7c, 0x85, 0xaa}, {0x7c, 0x05, 0xbb}, {0x7c, 0x45, 0xcc}},
		{{0xe5, 1}},
		{{0x78, 0, 2, 0x67}},
		{{0x65, 0, 0, 1, 0x61}},
	} {
		f.Add(encodeFuzzH264Packets(packets))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxFuzzH264Input {
			data = data[:maxFuzzH264Input]
		}
		packets := decodeFuzzH264Packets(data)
		if len(packets) == 0 {
			return
		}
		accessUnit, err := depacketizeH264AccessUnit(packets)
		if err != nil {
			return
		}
		if len(accessUnit.annexB) == 0 || len(accessUnit.annexB) > maxH264AccessUnit || !bytes.HasPrefix(accessUnit.annexB, annexBStartCode) {
			t.Fatalf("accepted invalid bounded access unit of %d bytes", len(accessUnit.annexB))
		}
	})
}

func FuzzH264Stream(f *testing.F) {
	for _, seed := range [][]byte{
		encodeFuzzH264Packets([][]byte{{0x67, 1}, {0x68, 2}, {0x65, 3}}),
		encodeFuzzH264Packets([][]byte{{0x7c, 0x85, 1}, {0x7c, 0x05, 2}, {0x7c, 0x45, 3}}),
		{3, 0, 2, 0x65, 1, 0, 2, 0x61, 2, 0, 2, 0x61, 3},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxFuzzH264Input {
			data = data[:maxFuzzH264Input]
		}
		packets := decodeFuzzH264Packets(data)
		stream := newH264Stream()
		observedAt := time.Unix(0, 0)
		for index, packet := range packets {
			if len(data) >= index+4 {
				packet.SequenceNumber = binary.BigEndian.Uint16(data[index : index+2])
				packet.Timestamp = uint32(binary.BigEndian.Uint16(data[index+2:index+4])) + 1
				packet.Marker = data[index]&1 != 0
			}
			unit, ok := stream.push(packet, uint64(index+1), uint64(index+1), observedAt.Add(time.Duration(index)*time.Millisecond))
			if ok && (len(unit) == 0 || len(unit) > maxH264AccessUnit+maxH264Parameters || !bytes.HasPrefix(unit, annexBStartCode)) {
				t.Fatalf("stream emitted invalid bounded access unit of %d bytes", len(unit))
			}
			if stream.pending != nil && (len(stream.pending.packets) > maxH264ReorderWindow || stream.pending.payloadBytes > maxH264AccessUnit) {
				t.Fatal("stream exceeded pending packet or byte bound")
			}
		}
	})
}

func encodeFuzzH264Packets(payloads [][]byte) []byte {
	encoded := []byte{byte(len(payloads))}
	for _, payload := range payloads {
		length := min(len(payload), 0xffff)
		encoded = binary.BigEndian.AppendUint16(encoded, uint16(length))
		encoded = append(encoded, payload[:length]...)
	}
	return encoded
}

func decodeFuzzH264Packets(data []byte) []*rtp.Packet {
	if len(data) == 0 {
		return nil
	}
	count := min(int(data[0]), maxFuzzH264Packets)
	data = data[1:]
	packets := make([]*rtp.Packet, 0, count)
	for index := 0; index < count && len(data) >= 2; index++ {
		length := min(int(binary.BigEndian.Uint16(data[:2])), len(data)-2)
		data = data[2:]
		payload := append([]byte(nil), data[:length]...)
		data = data[length:]
		packets = append(packets, rtpPacket(uint16(index+1), 1, index == count-1, payload))
	}
	return packets
}
