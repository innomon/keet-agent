package network

import (
	"errors"
	"io"
)

// MaxProtobufFrameLength is the maximum allowed frame length for Hypercore wire messages (8MB).
const MaxProtobufFrameLength = 8 * 1024 * 1024

// Feed represents a Hypercore feed message.
type Feed struct {
	DiscoveryKey []byte
}

// Handshake represents a Hypercore handshake message.
type Handshake struct {
	Protocol   string
	Key        []byte
	Extensions []string
	Live       bool
	UserData   []byte
}

// Request represents a Hypercore block request message.
type Request struct {
	Index uint64
	Bytes uint64
	Hash  bool
	Nodes uint64
}

// Data represents a Hypercore data message carrying the actual payload block.
type Data struct {
	Index     uint64
	Value     []byte
	Signature []byte
}

// Cancel represents a Hypercore cancellation of block request.
type Cancel struct {
	Index uint64
	Bytes uint64
}

// Helper methods for Feed

func (f *Feed) Marshal() ([]byte, error) {
	if len(f.DiscoveryKey) > MaxProtobufFrameLength {
		return nil, errors.New("discovery key length exceeds limit")
	}
	var buf []byte
	if len(f.DiscoveryKey) > 0 {
		buf = append(buf, encodeBytes(1, f.DiscoveryKey)...)
	}
	return buf, nil
}

func (f *Feed) Unmarshal(data []byte) error {
	return unmarshalFields(data, func(fieldNum int, wireType int, value []byte, varintVal uint64) error {
		if fieldNum == 1 && wireType == 2 {
			f.DiscoveryKey = make([]byte, len(value))
			copy(f.DiscoveryKey, value)
		}
		return nil
	})
}

// Helper methods for Handshake

func (h *Handshake) Validate() error {
	if h.Protocol == "" {
		return errors.New("protocol cannot be empty")
	}
	if len(h.Key) != 32 {
		return errors.New("key must be exactly 32 bytes")
	}
	return nil
}

func (h *Handshake) Marshal() ([]byte, error) {
	var buf []byte
	if h.Protocol != "" {
		buf = append(buf, encodeString(1, h.Protocol)...)
	}
	if len(h.Key) > 0 {
		buf = append(buf, encodeBytes(2, h.Key)...)
	}
	for _, ext := range h.Extensions {
		buf = append(buf, encodeString(3, ext)...)
	}
	if h.Live {
		buf = append(buf, encodeBool(4, h.Live)...)
	}
	if len(h.UserData) > 0 {
		buf = append(buf, encodeBytes(5, h.UserData)...)
	}
	if len(buf) > MaxProtobufFrameLength {
		return nil, errors.New("serialized handshake exceeds maximum limit")
	}
	return buf, nil
}

func (h *Handshake) Unmarshal(data []byte) error {
	return unmarshalFields(data, func(fieldNum int, wireType int, value []byte, varintVal uint64) error {
		switch fieldNum {
		case 1:
			if wireType == 2 {
				h.Protocol = string(value)
			}
		case 2:
			if wireType == 2 {
				h.Key = make([]byte, len(value))
				copy(h.Key, value)
			}
		case 3:
			if wireType == 2 {
				h.Extensions = append(h.Extensions, string(value))
			}
		case 4:
			if wireType == 0 {
				h.Live = varintVal != 0
			}
		case 5:
			if wireType == 2 {
				h.UserData = make([]byte, len(value))
				copy(h.UserData, value)
			}
		}
		return nil
	})
}

// Helper methods for Request

func (r *Request) Marshal() ([]byte, error) {
	var buf []byte
	buf = append(buf, encodeUint64(1, r.Index)...)
	buf = append(buf, encodeUint64(2, r.Bytes)...)
	if r.Hash {
		buf = append(buf, encodeBool(3, r.Hash)...)
	}
	if r.Nodes > 0 {
		buf = append(buf, encodeUint64(4, r.Nodes)...)
	}
	return buf, nil
}

func (r *Request) Unmarshal(data []byte) error {
	return unmarshalFields(data, func(fieldNum int, wireType int, value []byte, varintVal uint64) error {
		switch fieldNum {
		case 1:
			if wireType == 0 {
				r.Index = varintVal
			}
		case 2:
			if wireType == 0 {
				r.Bytes = varintVal
			}
		case 3:
			if wireType == 0 {
				r.Hash = varintVal != 0
			}
		case 4:
			if wireType == 0 {
				r.Nodes = varintVal
			}
		}
		return nil
	})
}

// Helper methods for Data

func (d *Data) Marshal() ([]byte, error) {
	if len(d.Value) > MaxProtobufFrameLength {
		return nil, errors.New("data value length exceeds limit")
	}
	var buf []byte
	buf = append(buf, encodeUint64(1, d.Index)...)
	if len(d.Value) > 0 {
		buf = append(buf, encodeBytes(2, d.Value)...)
	}
	if len(d.Signature) > 0 {
		buf = append(buf, encodeBytes(4, d.Signature)...)
	}
	if len(buf) > MaxProtobufFrameLength {
		return nil, errors.New("serialized data exceeds maximum limit")
	}
	return buf, nil
}

func (d *Data) Unmarshal(data []byte) error {
	if len(data) > MaxProtobufFrameLength {
		return errors.New("data frame exceeds maximum limit")
	}
	return unmarshalFields(data, func(fieldNum int, wireType int, value []byte, varintVal uint64) error {
		switch fieldNum {
		case 1:
			if wireType == 0 {
				d.Index = varintVal
			}
		case 2:
			if wireType == 2 {
				d.Value = make([]byte, len(value))
				copy(d.Value, value)
			}
		case 4:
			if wireType == 2 {
				d.Signature = make([]byte, len(value))
				copy(d.Signature, value)
			}
		}
		return nil
	})
}

// Helper methods for Cancel

func (c *Cancel) Marshal() ([]byte, error) {
	var buf []byte
	buf = append(buf, encodeUint64(1, c.Index)...)
	buf = append(buf, encodeUint64(2, c.Bytes)...)
	return buf, nil
}

func (c *Cancel) Unmarshal(data []byte) error {
	return unmarshalFields(data, func(fieldNum int, wireType int, value []byte, varintVal uint64) error {
		switch fieldNum {
		case 1:
			if wireType == 0 {
				c.Index = varintVal
			}
		case 2:
			if wireType == 0 {
				c.Bytes = varintVal
			}
		}
		return nil
	})
}

// Protobuf manual varint and field encoding/decoding helpers

func encodeVarint(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val|0x80))
		val >>= 7
	}
	buf = append(buf, byte(val))
	return buf
}

func decodeVarint(data []byte) (uint64, int, error) {
	var val uint64
	var shift uint
	for i, b := range data {
		val |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return val, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, errors.New("varint overflow")
		}
	}
	return 0, 0, io.ErrUnexpectedEOF
}

func encodeTag(fieldNumber int, wireType int) []byte {
	return encodeVarint(uint64((fieldNumber << 3) | wireType))
}

func encodeBytes(fieldNumber int, val []byte) []byte {
	var buf []byte
	buf = append(buf, encodeTag(fieldNumber, 2)...)
	buf = append(buf, encodeVarint(uint64(len(val)))...)
	buf = append(buf, val...)
	return buf
}

func encodeString(fieldNumber int, val string) []byte {
	return encodeBytes(fieldNumber, []byte(val))
}

func encodeUint64(fieldNumber int, val uint64) []byte {
	var buf []byte
	buf = append(buf, encodeTag(fieldNumber, 0)...)
	buf = append(buf, encodeVarint(val)...)
	return buf
}

func encodeBool(fieldNumber int, val bool) []byte {
	var v uint64
	if val {
		v = 1
	}
	return encodeUint64(fieldNumber, v)
}

func unmarshalFields(data []byte, handleField func(fieldNum int, wireType int, value []byte, varintVal uint64) error) error {
	if len(data) > MaxProtobufFrameLength {
		return errors.New("protobuf frame exceeds max size limit")
	}

	offset := 0
	for offset < len(data) {
		tag, n, err := decodeVarint(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case 0: // Varint
			val, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			err = handleField(fieldNum, wireType, nil, val)
			if err != nil {
				return err
			}
		case 2: // Length-delimited
			length, n, err := decodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			if offset+int(length) > len(data) {
				return errors.New("unexpected EOF reading length-delimited field")
			}
			val := data[offset : offset+int(length)]
			offset += int(length)
			err = handleField(fieldNum, wireType, val, 0)
			if err != nil {
				return err
			}
		default:
			return errors.New("unsupported wire type")
		}
	}
	return nil
}
