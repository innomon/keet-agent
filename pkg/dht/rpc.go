package dht

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// MsgType represents Kademlia message type.
type MsgType uint8

const (
	// MsgPing is a message sent to verify peer liveness.
	MsgPing MsgType = 1
	// MsgPong is the response to a Ping.
	MsgPong MsgType = 2
	// MsgFindNode requests the closest contacts to a target ID.
	MsgFindNode MsgType = 3
	// MsgFindNodeResp returns the closest contacts.
	MsgFindNodeResp MsgType = 4
	// MsgAnnounce registers a node's presence on a swarm topic.
	MsgAnnounce MsgType = 5
	// MsgLookup queries the network for peers on a swarm topic.
	MsgLookup MsgType = 6
	// MsgLookupResp returns the registered peer addresses.
	MsgLookupResp MsgType = 7
)

// Message is a container representing any Kademlia RPC message payload.
type Message struct {
	TxID     [4]byte
	Type     MsgType
	SenderID [32]byte

	// Payloads based on Type
	Target   [32]byte  // For MsgFindNode
	Contacts []Contact // For MsgFindNodeResp
	Topic    [32]byte  // For MsgAnnounce, MsgLookup
	Port     uint16    // For MsgAnnounce
	Peers    []string  // For MsgLookupResp
}

// EncodeMessage serializes a Message struct into a binary byte slice.
func EncodeMessage(m *Message) ([]byte, error) {
	if m == nil {
		return nil, errors.New("cannot encode nil message")
	}

	buf := new(bytes.Buffer)

	// 1. Write Header (TxID, Type, SenderID)
	if _, err := buf.Write(m.TxID[:]); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(uint8(m.Type)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.SenderID[:]); err != nil {
		return nil, err
	}

	// 2. Write Type-specific Payload
	switch m.Type {
	case MsgPing, MsgPong:
		// No payload

	case MsgFindNode:
		if _, err := buf.Write(m.Target[:]); err != nil {
			return nil, err
		}

	case MsgFindNodeResp:
		count := uint16(len(m.Contacts))
		if err := binary.Write(buf, binary.BigEndian, count); err != nil {
			return nil, err
		}
		for _, c := range m.Contacts {
			if _, err := buf.Write(c.ID[:]); err != nil {
				return nil, err
			}
			addrBytes := []byte(c.Addr)
			addrLen := uint16(len(addrBytes))
			if err := binary.Write(buf, binary.BigEndian, addrLen); err != nil {
				return nil, err
			}
			if _, err := buf.Write(addrBytes); err != nil {
				return nil, err
			}
		}

	case MsgAnnounce:
		if _, err := buf.Write(m.Topic[:]); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, m.Port); err != nil {
			return nil, err
		}

	case MsgLookup:
		if _, err := buf.Write(m.Topic[:]); err != nil {
			return nil, err
		}

	case MsgLookupResp:
		count := uint16(len(m.Peers))
		if err := binary.Write(buf, binary.BigEndian, count); err != nil {
			return nil, err
		}
		for _, peer := range m.Peers {
			peerBytes := []byte(peer)
			peerLen := uint16(len(peerBytes))
			if err := binary.Write(buf, binary.BigEndian, peerLen); err != nil {
				return nil, err
			}
			if _, err := buf.Write(peerBytes); err != nil {
				return nil, err
			}
		}

	default:
		return nil, fmt.Errorf("unknown message type: %d", m.Type)
	}

	return buf.Bytes(), nil
}

// DecodeMessage deserializes a binary byte slice into a Message struct.
func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < 37 {
		return nil, errors.New("packet too short to contain header")
	}

	m := &Message{}
	reader := bytes.NewReader(data)

	// 1. Read Header
	if _, err := reader.Read(m.TxID[:]); err != nil {
		return nil, err
	}
	typeByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	m.Type = MsgType(typeByte)

	if _, err := reader.Read(m.SenderID[:]); err != nil {
		return nil, err
	}

	// 2. Read Payload
	switch m.Type {
	case MsgPing, MsgPong:
		// No payload

	case MsgFindNode:
		if _, err := reader.Read(m.Target[:]); err != nil {
			return nil, err
		}

	case MsgFindNodeResp:
		var count uint16
		if err := binary.Read(reader, binary.BigEndian, &count); err != nil {
			return nil, err
		}
		m.Contacts = make([]Contact, count)
		for i := 0; i < int(count); i++ {
			var c Contact
			if _, err := reader.Read(c.ID[:]); err != nil {
				return nil, err
			}
			var addrLen uint16
			if err := binary.Read(reader, binary.BigEndian, &addrLen); err != nil {
				return nil, err
			}
			addrBytes := make([]byte, addrLen)
			if _, err := reader.Read(addrBytes); err != nil {
				return nil, err
			}
			c.Addr = string(addrBytes)
			m.Contacts[i] = c
		}

	case MsgAnnounce:
		if _, err := reader.Read(m.Topic[:]); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.BigEndian, &m.Port); err != nil {
			return nil, err
		}

	case MsgLookup:
		if _, err := reader.Read(m.Topic[:]); err != nil {
			return nil, err
		}

	case MsgLookupResp:
		var count uint16
		if err := binary.Read(reader, binary.BigEndian, &count); err != nil {
			return nil, err
		}
		m.Peers = make([]string, count)
		for i := 0; i < int(count); i++ {
			var peerLen uint16
			if err := binary.Read(reader, binary.BigEndian, &peerLen); err != nil {
				return nil, err
			}
			peerBytes := make([]byte, peerLen)
			if _, err := reader.Read(peerBytes); err != nil {
				return nil, err
			}
			m.Peers[i] = string(peerBytes)
		}

	default:
		return nil, fmt.Errorf("unknown message type: %d", m.Type)
	}

	return m, nil
}
