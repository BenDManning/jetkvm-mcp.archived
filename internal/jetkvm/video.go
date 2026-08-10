package jetkvm

// Selectively adapted from Ben Manning's MIT-licensed legacy JetKVM MCP implementation.
// This file contains only RTP/H.264 parsing and bounded stream assembly.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
)

const (
	maxH264ReorderWindow = 64
	maxH264Gap           = 200 * time.Millisecond
	maxH264Packets       = 4096
	maxH264AccessUnit    = 4 << 20
	maxH264ParameterNALU = 32 << 10
	maxH264Parameters    = 64 << 10
)

var (
	errInvalidH264AccessUnit = errors.New("invalid H.264 access unit")
	errIncompleteH264Unit    = errors.New("incomplete H.264 access unit")
	annexBStartCode          = []byte{0, 0, 0, 1}
)

type h264Unit struct {
	annexB    []byte
	nalus     [][]byte
	hasIDR    bool
	latestSPS []byte
	latestPPS []byte
}

type observedH264Packet struct {
	packet     *rtp.Packet
	generation uint64
	observedAt time.Time
}

type pendingH264Unit struct {
	timestamp      uint32
	packets        map[uint16]observedH264Packet
	expectedSet    bool
	expected       uint16
	reorderNext    uint16
	reorderBase    uint16
	frontierSet    bool
	frontier       uint16
	markerSet      bool
	markerExplicit bool
	marker         uint16
	lastObserved   time.Time
	payloadBytes   int
	minGeneration  uint64
}

type h264DrainUnit struct {
	timestamp      uint32
	anchored       bool
	lastSequence   uint16
	haveSequence   bool
	lastObserved   time.Time
	invalid        bool
	sequenceGap    bool
	packetCount    int
	payloadBytes   int
	projectedBytes int
	fuOpen         bool
	fuType         byte
	fuNRI          byte
	fuZeroRun      byte
	fuParameter    []byte
	latestSPS      []byte
	latestPPS      []byte
}

type completedH264Unit struct {
	frontier    uint16
	hasFrontier bool
}

type h264Stream struct {
	pending             *pendingH264Unit
	draining            *h264DrainUnit
	sps                 []byte
	pps                 []byte
	nextSequence        uint16
	haveNext            bool
	anchorTimestamp     uint32
	anchorSet           bool
	completed           map[uint32]completedH264Unit
	completedOrder      []uint32
	candidateAt         time.Time
	candidateGeneration uint64
}

func newH264Stream() *h264Stream {
	return &h264Stream{completed: make(map[uint32]completedH264Unit)}
}

func (stream *h264Stream) push(packet *rtp.Packet, generation, watermark uint64, observedAt time.Time) ([]byte, bool) {
	return stream.process(packet, generation, watermark, observedAt, true)
}

func (stream *h264Stream) drain(packet *rtp.Packet, _ uint64, observedAt time.Time) {
	if stream == nil || packet == nil {
		return
	}
	stream.expire(observedAt)
	if packet.Version != 2 || len(packet.Payload) == 0 {
		stream.poisonPacket(packet)
		return
	}
	if stream.pending != nil {
		stream.discardPending()
	}
	stream.drainPacket(packet, observedAt)
}

func (stream *h264Stream) drainPacket(packet *rtp.Packet, observedAt time.Time) {
	if stream.rejectCompleted(packet) {
		return
	}
	if stream.draining != nil && stream.draining.timestamp != packet.Timestamp {
		stream.finishDrainUnit(packet.SequenceNumber, false)
	}
	if stream.draining == nil {
		stream.draining = &h264DrainUnit{timestamp: packet.Timestamp, anchored: stream.haveNext}
	}
	unit := stream.draining
	unit.lastObserved = observedAt
	if unit.haveSequence {
		distance := packet.SequenceNumber - unit.lastSequence
		if distance != 1 {
			unit.invalid = true
			unit.sequenceGap = true
		}
		if distance > 0 && distance < 1<<15 {
			unit.lastSequence = packet.SequenceNumber
		}
	} else {
		if stream.haveNext && packet.SequenceNumber != stream.nextSequence {
			unit.invalid = true
			unit.sequenceGap = true
		}
		unit.haveSequence = true
		unit.lastSequence = packet.SequenceNumber
	}
	if unit.packetCount >= maxH264Packets || len(packet.Payload) > maxH264AccessUnit-unit.payloadBytes {
		unit.invalid = true
	} else {
		unit.packetCount++
		unit.payloadBytes += len(packet.Payload)
	}
	if !unit.invalid {
		incomplete, projected, err := validateH264Payload(packet.Payload, &unit.fuOpen, &unit.fuType, &unit.fuNRI)
		if err != nil || incomplete || projected > maxH264AccessUnit-unit.projectedBytes {
			unit.invalid = true
		} else {
			unit.projectedBytes += projected
			stream.captureDrainParameters(unit, packet.Payload)
		}
	}
	if packet.Marker {
		stream.finishDrainUnit(packet.SequenceNumber+1, true)
	}
}

func (stream *h264Stream) captureDrainParameters(unit *h264DrainUnit, payload []byte) {
	naluType := payload[0] & 0x1f
	switch {
	case validSingleNALType(naluType):
		if !validH264NALU(payload) {
			unit.invalid = true
			return
		}
		if naluType == 7 || naluType == 8 {
			stream.setDrainParameter(unit, naluType, payload)
		}
	case naluType == 24:
		for offset := 1; offset < len(payload); {
			size := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			nalu := payload[offset : offset+size]
			if !validH264NALU(nalu) {
				unit.invalid = true
				return
			}
			if parameterType := nalu[0] & 0x1f; parameterType == 7 || parameterType == 8 {
				stream.setDrainParameter(unit, parameterType, nalu)
			}
			offset += size
		}
	case naluType == 28:
		fragmentType := payload[1] & 0x1f
		start := payload[1]&0x80 != 0
		end := payload[1]&0x40 != 0
		if start {
			unit.fuZeroRun = 0
		}
		for _, value := range payload[2:] {
			if value == 0 {
				if unit.fuZeroRun < 2 {
					unit.fuZeroRun++
				}
				continue
			}
			if value == 1 && unit.fuZeroRun == 2 {
				unit.invalid = true
			}
			unit.fuZeroRun = 0
		}
		if end {
			unit.fuZeroRun = 0
		}
		if unit.invalid {
			return
		}
		if fragmentType != 7 && fragmentType != 8 {
			return
		}
		if start {
			unit.fuParameter = append(unit.fuParameter[:0], payload[0]&0xe0|fragmentType)
		}
		fragment := payload[2:]
		if len(fragment) > maxH264ParameterNALU-len(unit.fuParameter) {
			unit.invalid = true
			unit.fuParameter = nil
			return
		}
		unit.fuParameter = append(unit.fuParameter, fragment...)
		if end {
			stream.setDrainParameter(unit, fragmentType, unit.fuParameter)
			unit.fuParameter = nil
		}
	}
}

func (stream *h264Stream) setDrainParameter(unit *h264DrainUnit, naluType byte, nalu []byte) {
	if len(nalu) > maxH264ParameterNALU || !validH264NALU(nalu) {
		unit.invalid = true
		return
	}
	parameter := append([]byte(nil), nalu...)
	if naluType == 7 {
		unit.latestSPS = parameter
	} else {
		unit.latestPPS = parameter
	}
}

func (stream *h264Stream) finishDrainUnit(nextSequence uint16, markerObserved bool) {
	unit := stream.draining
	if unit == nil {
		return
	}
	continuous := unit.haveSequence && nextSequence == unit.lastSequence+1
	if !continuous {
		unit.invalid = true
	}
	stream.draining = nil
	stream.rememberCompleted(unit.timestamp, unit.lastSequence, unit.haveSequence)
	if continuous && (markerObserved || !unit.sequenceGap) {
		stream.setSequenceAnchor(unit.timestamp, nextSequence)
	} else {
		stream.clearSequenceAnchor()
	}
	if unit.invalid || unit.fuOpen || !unit.anchored {
		return
	}
	stream.accept(h264Unit{latestSPS: unit.latestSPS, latestPPS: unit.latestPPS}, false)
}

func (stream *h264Stream) beginWait() {
	if stream == nil || stream.draining == nil {
		return
	}
	// The in-flight access unit straddles waiter registration and can never
	// satisfy freshness or update retained parameters. Keep it only long enough
	// to recover a sequence baseline at its marker or timestamp boundary.
	stream.draining.invalid = true
	stream.draining.fuParameter = nil
	stream.draining.latestSPS = nil
	stream.draining.latestPPS = nil
}

func (stream *h264Stream) endWait() {
	if stream != nil && stream.pending != nil {
		stream.discardPending()
	}
}

func (stream *h264Stream) poisonPacket(packet *rtp.Packet) {
	if stream == nil || packet == nil {
		return
	}
	if _, replay := stream.completed[packet.Timestamp]; replay {
		return
	}
	if stream.pending != nil {
		stream.rememberCompleted(stream.pending.timestamp, stream.pending.frontier, stream.pending.frontierSet)
		stream.pending = nil
	}
	if stream.draining != nil {
		stream.rememberCompleted(stream.draining.timestamp, stream.draining.lastSequence, stream.draining.haveSequence)
		stream.draining = nil
	}
	stream.rememberCompleted(packet.Timestamp, packet.SequenceNumber, packet.Version == 2)
	if packet.Version == 2 {
		stream.setSequenceAnchor(packet.Timestamp, packet.SequenceNumber+1)
	} else {
		stream.clearSequenceAnchor()
	}
}

func (stream *h264Stream) process(packet *rtp.Packet, generation, watermark uint64, observedAt time.Time, wantCandidate bool) ([]byte, bool) {
	if stream == nil || packet == nil {
		return nil, false
	}
	stream.expire(observedAt)
	if packet.Version != 2 || len(packet.Payload) == 0 {
		stream.poisonPacket(packet)
		return nil, false
	}
	if stream.rejectCompleted(packet) {
		return nil, false
	}
	if stream.draining != nil {
		if stream.draining.timestamp == packet.Timestamp {
			stream.drainPacket(packet, observedAt)
			return nil, false
		}
		stream.finishDrainUnit(packet.SequenceNumber, false)
	}
	if stream.pending != nil && stream.pending.timestamp != packet.Timestamp {
		data, ok := stream.finishAtTimestampTransition(packet.SequenceNumber, watermark, observedAt, wantCandidate)
		if ok {
			stream.process(packet, generation, watermark, observedAt, false)
			return data, true
		}
	}
	if stream.pending == nil {
		stream.pending = &pendingH264Unit{
			timestamp:     packet.Timestamp,
			packets:       make(map[uint16]observedH264Packet),
			minGeneration: generation,
			expectedSet:   stream.haveNext,
			expected:      stream.nextSequence,
			reorderNext:   stream.nextSequence,
		}
	}
	pending := stream.pending
	pending.lastObserved = observedAt
	if existing, exists := pending.packets[packet.SequenceNumber]; exists {
		if !sameRTPPacket(existing.packet, packet) {
			stream.discardPending()
		}
		return nil, false
	}
	if !pending.sequenceWithinReorderWindow(packet.SequenceNumber) {
		pending.observeSequence(packet.SequenceNumber)
		if packet.Marker {
			pending.markerSet = true
			pending.markerExplicit = true
			pending.marker = packet.SequenceNumber
		}
		stream.discardPending()
		return nil, false
	}
	pending.observeSequence(packet.SequenceNumber)
	if len(pending.packets) >= maxH264Packets || len(packet.Payload) > maxH264AccessUnit-pending.payloadBytes {
		stream.discardPending()
		return nil, false
	}
	copyPacket := packet.Clone()
	pending.packets[packet.SequenceNumber] = observedH264Packet{packet: copyPacket, generation: generation, observedAt: observedAt}
	for {
		if _, exists := pending.packets[pending.reorderNext]; !exists {
			break
		}
		pending.reorderNext++
	}
	if !pending.expectedSet {
		pending.reorderBase = pending.reorderNext - 1
	}
	pending.payloadBytes += len(packet.Payload)
	if generation < pending.minGeneration {
		pending.minGeneration = generation
	}
	if packet.Marker {
		if pending.markerSet && pending.marker != packet.SequenceNumber {
			stream.discardPending()
			return nil, false
		}
		pending.markerSet = true
		pending.markerExplicit = true
		pending.marker = packet.SequenceNumber
	}
	packets, complete, invalid := pending.orderedPackets()
	if invalid {
		stream.discardPending()
		return nil, false
	}
	if !complete {
		return nil, false
	}
	unit, err := depacketizeH264AccessUnit(packets)
	if errors.Is(err, errIncompleteH264Unit) {
		return nil, false
	}
	stream.pending = nil
	stream.setSequenceAnchor(pending.timestamp, pending.marker+1)
	stream.rememberCompleted(pending.timestamp, pending.frontier, pending.frontierSet)
	if err != nil || !pending.expectedSet || (watermark != 0 && pending.minGeneration <= watermark) {
		return nil, false
	}
	data, ok := stream.accept(unit, wantCandidate)
	if ok {
		stream.candidateAt = observedAt
		stream.candidateGeneration = pending.minGeneration
	}
	return data, ok
}

func (stream *h264Stream) finishAtTimestampTransition(nextSequence uint16, watermark uint64, observedAt time.Time, wantCandidate bool) ([]byte, bool) {
	pending := stream.pending
	if pending == nil {
		return nil, false
	}
	if pending.markerSet {
		stream.discardPending()
		return nil, false
	}
	lastSequence := nextSequence - 1
	last, exists := pending.packets[lastSequence]
	if !exists {
		stream.discardPending()
		return nil, false
	}
	last.packet = last.packet.Clone()
	last.packet.Marker = true
	pending.packets[lastSequence] = last
	pending.markerSet = true
	pending.marker = lastSequence
	packets, complete, invalid := pending.orderedPackets()
	if invalid || !complete {
		stream.discardPending()
		return nil, false
	}
	unit, err := depacketizeH264AccessUnit(packets)
	stream.pending = nil
	stream.setSequenceAnchor(pending.timestamp, nextSequence)
	stream.rememberCompleted(pending.timestamp, pending.frontier, pending.frontierSet)
	if err != nil || !pending.expectedSet || (watermark != 0 && pending.minGeneration <= watermark) {
		return nil, false
	}
	data, ok := stream.accept(unit, wantCandidate)
	if ok {
		stream.candidateAt = observedAt
		stream.candidateGeneration = pending.minGeneration
	}
	return data, ok
}

func (stream *h264Stream) rememberCompleted(timestamp uint32, frontier uint16, hasFrontier bool) {
	if stream.completed == nil {
		stream.completed = make(map[uint32]completedH264Unit)
	}
	if _, exists := stream.completed[timestamp]; exists {
		return
	}
	stream.completed[timestamp] = completedH264Unit{frontier: frontier, hasFrontier: hasFrontier}
	stream.completedOrder = append(stream.completedOrder, timestamp)
	if len(stream.completedOrder) > maxH264ReorderWindow {
		delete(stream.completed, stream.completedOrder[0])
		stream.completedOrder = stream.completedOrder[1:]
	}
}

func (stream *h264Stream) setSequenceAnchor(timestamp uint32, nextSequence uint16) {
	stream.nextSequence = nextSequence
	stream.haveNext = true
	stream.anchorTimestamp = timestamp
	stream.anchorSet = true
}

func (stream *h264Stream) clearSequenceAnchor() {
	stream.haveNext = false
	stream.anchorSet = false
}

func (stream *h264Stream) rejectCompleted(packet *rtp.Packet) bool {
	completed, exists := stream.completed[packet.Timestamp]
	if !exists {
		return false
	}
	if packet.Version != 2 {
		return true
	}
	if !completed.hasFrontier {
		completed.frontier = packet.SequenceNumber
		completed.hasFrontier = true
		stream.completed[packet.Timestamp] = completed
		return true
	}
	distance := packet.SequenceNumber - completed.frontier
	if distance == 0 || distance >= 1<<15 {
		return true
	}
	completed.frontier = packet.SequenceNumber
	stream.completed[packet.Timestamp] = completed
	if stream.anchorSet && stream.anchorTimestamp == packet.Timestamp {
		stream.clearSequenceAnchor()
		if stream.pending != nil {
			stream.pending.expectedSet = false
		}
		if stream.draining != nil {
			stream.draining.anchored = false
		}
	}
	return true
}

func (stream *h264Stream) expire(observedAt time.Time) bool {
	if stream == nil {
		return false
	}
	expired := false
	if stream.pending != nil && !stream.pending.lastObserved.IsZero() && observedAt.Sub(stream.pending.lastObserved) >= maxH264Gap {
		stream.discardPending()
		expired = true
	}
	if stream.draining != nil && !stream.draining.lastObserved.IsZero() && observedAt.Sub(stream.draining.lastObserved) >= maxH264Gap {
		unit := stream.draining
		stream.draining = nil
		stream.rememberCompleted(unit.timestamp, unit.lastSequence, unit.haveSequence)
		if unit.haveSequence && !unit.invalid && !unit.fuOpen && !unit.sequenceGap {
			stream.setSequenceAnchor(unit.timestamp, unit.lastSequence+1)
		} else {
			stream.clearSequenceAnchor()
		}
		expired = true
	}
	return expired
}

func (stream *h264Stream) discardPending() {
	if stream.pending != nil {
		stream.rememberCompleted(stream.pending.timestamp, stream.pending.frontier, stream.pending.frontierSet)
		if stream.pending.markerExplicit && stream.pending.markerSet && stream.pending.frontierSet && stream.pending.marker == stream.pending.frontier {
			stream.setSequenceAnchor(stream.pending.timestamp, stream.pending.marker+1)
		} else {
			stream.clearSequenceAnchor()
		}
	}
	stream.pending = nil
}

func (pending *pendingH264Unit) sequenceWithinReorderWindow(sequence uint16) bool {
	if pending.expectedSet {
		return sequence-pending.reorderNext < maxH264ReorderWindow
	}
	if !pending.frontierSet {
		pending.reorderNext = sequence
		pending.reorderBase = sequence
		return true
	}
	forward := sequence - pending.reorderBase
	if forward < 1<<15 {
		return forward <= maxH264ReorderWindow
	}
	backward := pending.reorderBase - sequence
	span := pending.frontier - sequence
	if backward > maxH264ReorderWindow || (span < 1<<15 && span > maxH264ReorderWindow) {
		return false
	}
	pending.reorderNext = sequence
	return true
}

func (pending *pendingH264Unit) observeSequence(sequence uint16) {
	if !pending.frontierSet {
		pending.frontierSet = true
		pending.frontier = sequence
		return
	}
	distance := sequence - pending.frontier
	if distance > 0 && distance < 1<<15 {
		pending.frontier = sequence
	}
}

func sameRTPPacket(left, right *rtp.Packet) bool {
	leftBytes, leftErr := left.Marshal()
	rightBytes, rightErr := right.Marshal()
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func (pending *pendingH264Unit) orderedPackets() ([]*rtp.Packet, bool, bool) {
	if pending == nil || !pending.markerSet {
		return nil, false, false
	}
	maxDistance := uint16(0)
	if pending.expectedSet {
		maxDistance = pending.marker - pending.expected
		if maxDistance >= maxH264Packets {
			return nil, false, true
		}
	}
	for sequence := range pending.packets {
		distance := pending.marker - sequence
		if distance >= maxH264Packets {
			return nil, false, true
		}
		if distance > maxDistance {
			maxDistance = distance
		}
	}
	if int(maxDistance)+1 != len(pending.packets) {
		return nil, false, false
	}
	packets := make([]*rtp.Packet, 0, len(pending.packets))
	for distance := int(maxDistance); distance >= 0; distance-- {
		sequence := pending.marker - uint16(distance)
		observed, exists := pending.packets[sequence]
		if !exists {
			return nil, false, false
		}
		packets = append(packets, observed.packet)
	}
	return packets, true, false
}

func depacketizeH264AccessUnit(packets []*rtp.Packet) (h264Unit, error) {
	if len(packets) == 0 || len(packets) > maxH264Packets {
		return h264Unit{}, errInvalidH264AccessUnit
	}
	depacketizer := &codecs.H264Packet{IsAVC: false}
	var output []byte
	var nalus [][]byte
	var payloadBytes int
	var projectedBytes int
	var fuOpen bool
	var fuType, fuNRI byte
	for index, packet := range packets {
		if packet == nil || packet.Version != 2 || len(packet.Payload) == 0 {
			return h264Unit{}, errInvalidH264AccessUnit
		}
		if index > 0 {
			previous := packets[index-1]
			if packet.Timestamp != previous.Timestamp || packet.SequenceNumber != previous.SequenceNumber+1 || previous.Marker {
				return h264Unit{}, errInvalidH264AccessUnit
			}
		}
		if packet.Timestamp != packets[0].Timestamp || packet.Marker != (index == len(packets)-1) {
			return h264Unit{}, errInvalidH264AccessUnit
		}
		if len(packet.Payload) > maxH264AccessUnit-payloadBytes {
			return h264Unit{}, errInvalidH264AccessUnit
		}
		payloadBytes += len(packet.Payload)
		incomplete, projected, err := validateH264Payload(packet.Payload, &fuOpen, &fuType, &fuNRI)
		if err != nil {
			return h264Unit{}, err
		}
		if incomplete {
			return h264Unit{}, errIncompleteH264Unit
		}
		if projected > maxH264AccessUnit-projectedBytes {
			return h264Unit{}, errInvalidH264AccessUnit
		}
		projectedBytes += projected
		chunk, err := depacketizer.Unmarshal(packet.Payload)
		if err != nil || len(chunk) > maxH264AccessUnit-len(output) {
			return h264Unit{}, errInvalidH264AccessUnit
		}
		output = append(output, chunk...)
		naluType := packet.Payload[0] & 0x1f
		switch {
		case validSingleNALType(naluType):
			var valid bool
			nalus, valid = appendValidatedNALU(nalus, packet.Payload)
			if !valid {
				return h264Unit{}, errInvalidH264AccessUnit
			}
		case naluType == 24:
			for offset := 1; offset < len(packet.Payload); {
				size := int(binary.BigEndian.Uint16(packet.Payload[offset : offset+2]))
				offset += 2
				var valid bool
				nalus, valid = appendValidatedNALU(nalus, packet.Payload[offset:offset+size])
				if !valid {
					return h264Unit{}, errInvalidH264AccessUnit
				}
				offset += size
			}
		case naluType == 28 && len(chunk) != 0:
			if len(chunk) <= len(annexBStartCode) || !bytes.Equal(chunk[:len(annexBStartCode)], annexBStartCode) {
				return h264Unit{}, errInvalidH264AccessUnit
			}
			var valid bool
			nalus, valid = appendValidatedNALU(nalus, chunk[len(annexBStartCode):])
			if !valid {
				return h264Unit{}, errInvalidH264AccessUnit
			}
		}
	}
	if fuOpen {
		return h264Unit{}, errIncompleteH264Unit
	}
	unit := h264Unit{annexB: output, nalus: nalus}
	for _, nalu := range nalus {
		naluType := nalu[0] & 0x1f
		if (naluType == 7 || naluType == 8) && len(nalu) > maxH264ParameterNALU {
			return h264Unit{}, errInvalidH264AccessUnit
		}
		switch naluType {
		case 5:
			unit.hasIDR = true
		case 7:
			unit.latestSPS = nalu
		case 8:
			unit.latestPPS = nalu
		}
	}
	return unit, nil
}

func appendValidatedNALU(nalus [][]byte, nalu []byte) ([][]byte, bool) {
	if !validH264NALU(nalu) {
		return nil, false
	}
	return append(nalus, append([]byte(nil), nalu...)), true
}

func validH264NALU(nalu []byte) bool {
	return len(nalu) != 0 && !bytes.Contains(nalu[1:], annexBStartCode[1:])
}

func validateH264Payload(payload []byte, fuOpen *bool, fuType, fuNRI *byte) (bool, int, error) {
	if len(payload) == 0 || payload[0]&0x80 != 0 {
		return false, 0, errInvalidH264AccessUnit
	}
	naluType := payload[0] & 0x1f
	switch {
	case validSingleNALType(naluType):
		if *fuOpen {
			return false, 0, errInvalidH264AccessUnit
		}
		return false, len(annexBStartCode) + len(payload), nil
	case naluType == 24:
		projected, valid := stapAAnnexBSize(payload)
		if *fuOpen || !valid {
			return false, 0, errInvalidH264AccessUnit
		}
		return false, projected, nil
	case naluType == 28:
		if len(payload) <= 2 || payload[1]&0x20 != 0 {
			return false, 0, errInvalidH264AccessUnit
		}
		start := payload[1]&0x80 != 0
		end := payload[1]&0x40 != 0
		fragmentType := payload[1] & 0x1f
		fragmentNRI := payload[0] & 0x60
		if !validSingleNALType(fragmentType) || start && end {
			return false, 0, errInvalidH264AccessUnit
		}
		switch {
		case start:
			if *fuOpen {
				return false, 0, errInvalidH264AccessUnit
			}
			*fuOpen, *fuType, *fuNRI = true, fragmentType, fragmentNRI
		case !*fuOpen:
			return true, 0, nil
		case fragmentType != *fuType || fragmentNRI != *fuNRI:
			return false, 0, errInvalidH264AccessUnit
		}
		projected := len(payload) - 2
		if start {
			projected += len(annexBStartCode) + 1
		}
		if end {
			*fuOpen = false
		}
		return false, projected, nil
	default:
		return false, 0, errInvalidH264AccessUnit
	}
}

func validSingleNALType(naluType byte) bool {
	return naluType >= 1 && naluType <= 16 || naluType >= 19 && naluType <= 21
}

func stapAAnnexBSize(payload []byte) (int, bool) {
	if len(payload) < 4 {
		return 0, false
	}
	maxNRI := byte(0)
	projected := 0
	for offset := 1; offset < len(payload); {
		if len(payload)-offset < 2 {
			return 0, false
		}
		size := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
		offset += 2
		if size == 0 || size > len(payload)-offset {
			return 0, false
		}
		nalu := payload[offset : offset+size]
		naluType := nalu[0] & 0x1f
		if nalu[0]&0x80 != 0 || !validSingleNALType(naluType) {
			return 0, false
		}
		if nri := nalu[0] & 0x60; nri > maxNRI {
			maxNRI = nri
		}
		if size > maxH264AccessUnit-len(annexBStartCode)-projected {
			return 0, false
		}
		projected += len(annexBStartCode) + size
		offset += size
	}
	return projected, payload[0]&0x60 == maxNRI
}

func (stream *h264Stream) accept(unit h264Unit, wantCandidate bool) ([]byte, bool) {
	newSPS, newPPS := stream.sps, stream.pps
	if unit.latestSPS != nil {
		newSPS = unit.latestSPS
	}
	if unit.latestPPS != nil {
		newPPS = unit.latestPPS
	}
	if !validH264Parameters(newSPS, newPPS) {
		return nil, false
	}
	if !wantCandidate {
		stream.sps = append(stream.sps[:0], newSPS...)
		stream.pps = append(stream.pps[:0], newPPS...)
		return nil, false
	}
	var candidate []byte
	if unit.hasIDR {
		if newSPS == nil || newPPS == nil {
			stream.sps = append(stream.sps[:0], newSPS...)
			stream.pps = append(stream.pps[:0], newPPS...)
			return nil, false
		}
		nalus := make([][]byte, 0, len(unit.nalus)+2)
		nalus = append(nalus, newSPS, newPPS)
		for _, nalu := range unit.nalus {
			typeID := nalu[0] & 0x1f
			if typeID != 7 && typeID != 8 {
				nalus = append(nalus, nalu)
			}
		}
		var total int
		for _, nalu := range nalus {
			if len(nalu) > maxH264AccessUnit-len(annexBStartCode)-total {
				return nil, false
			}
			total += len(annexBStartCode) + len(nalu)
		}
		candidate = make([]byte, 0, total)
		for _, nalu := range nalus {
			candidate = append(candidate, annexBStartCode...)
			candidate = append(candidate, nalu...)
		}
	}
	stream.sps = append(stream.sps[:0], newSPS...)
	stream.pps = append(stream.pps[:0], newPPS...)
	return candidate, unit.hasIDR
}

func validH264Parameters(sps, pps []byte) bool {
	if sps == nil && pps == nil {
		return true
	}
	if sps != nil && (len(sps) > maxH264ParameterNALU || sps[0]&0x1f != 7) {
		return false
	}
	if pps != nil && (len(pps) > maxH264ParameterNALU || pps[0]&0x1f != 8) {
		return false
	}
	combined := 0
	for _, nalu := range [][]byte{sps, pps} {
		if nalu != nil {
			combined += len(annexBStartCode) + len(nalu)
		}
	}
	return combined <= maxH264Parameters
}
