package utp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// MsgType represents the µTP packet type.
type MsgType uint8

const (
	// ST_DATA is used to transmit data payload.
	ST_DATA MsgType = 0
	// ST_FIN is used to initiate connection termination.
	ST_FIN MsgType = 1
	// ST_STATE is used to acknowledge data or handshakes without any payload.
	ST_STATE MsgType = 2
	// ST_RESET is used to forcefully terminate a connection.
	ST_RESET MsgType = 3
	// ST_SYN is used to initiate a connection.
	ST_SYN MsgType = 4
)

// Header represents the 20-byte standard µTP packet header.
type Header struct {
	Type    MsgType
	Version uint8
	Ext     uint8
	ConnID  uint16
	TMicro  uint32
	TDiff   uint32
	WNDSize uint32
	SeqNum  uint16
	AckNum  uint16
}

// Packet represents a full µTP packet including the Header and optional payload.
type Packet struct {
	Header  Header
	Payload []byte
}

// Encode serializes the Packet struct into its wire-format binary representation.
func (p *Packet) Encode() ([]byte, error) {
	buf := make([]byte, 20+len(p.Payload))

	// Byte 0: Type (4 bits) | Version (4 bits)
	buf[0] = (uint8(p.Header.Type) << 4) | (p.Header.Version & 0x0f)

	// Byte 1: Extension
	buf[1] = p.Header.Ext

	// Bytes 2-3: Connection ID
	binary.BigEndian.PutUint16(buf[2:4], p.Header.ConnID)

	// Bytes 4-7: Timestamp Microseconds
	binary.BigEndian.PutUint32(buf[4:8], p.Header.TMicro)

	// Bytes 8-11: Timestamp Difference
	binary.BigEndian.PutUint32(buf[8:12], p.Header.TDiff)

	// Bytes 12-15: Window Size
	binary.BigEndian.PutUint32(buf[12:16], p.Header.WNDSize)

	// Bytes 16-17: Sequence Number
	binary.BigEndian.PutUint16(buf[16:18], p.Header.SeqNum)

	// Bytes 18-19: Acknowledgment Number
	binary.BigEndian.PutUint16(buf[18:20], p.Header.AckNum)

	// Payload
	if len(p.Payload) > 0 {
		copy(buf[20:], p.Payload)
	}

	return buf, nil
}

// DecodePacket parses a wire-format binary slice into a Packet struct.
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < 20 {
		return nil, errors.New("packet too short to contain µTP header")
	}

	p := &Packet{}

	// Byte 0: Type and Version
	p.Header.Type = MsgType(data[0] >> 4)
	p.Header.Version = data[0] & 0x0f

	if p.Header.Version != 1 {
		return nil, fmt.Errorf("unsupported µTP version: %d (only version 1 is supported)", p.Header.Version)
	}

	// Byte 1: Extension
	p.Header.Ext = data[1]

	// Bytes 2-3: Connection ID
	p.Header.ConnID = binary.BigEndian.Uint16(data[2:4])

	// Bytes 4-7: Timestamp Microseconds
	p.Header.TMicro = binary.BigEndian.Uint32(data[4:8])

	// Bytes 8-11: Timestamp Difference
	p.Header.TDiff = binary.BigEndian.Uint32(data[8:12])

	// Bytes 12-15: Window Size
	p.Header.WNDSize = binary.BigEndian.Uint32(data[12:16])

	// Bytes 16-17: Sequence Number
	p.Header.SeqNum = binary.BigEndian.Uint16(data[16:18])

	// Bytes 18-19: Acknowledgment Number
	p.Header.AckNum = binary.BigEndian.Uint16(data[18:20])

	// Payload
	if len(data) > 20 {
		p.Payload = make([]byte, len(data)-20)
		copy(p.Payload, data[20:])
	}

	return p, nil
}
