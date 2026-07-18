package hypercore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type Handshake struct {
	Protocol string
	Key      []byte
}

type Have struct {
	Start uint64
	Len   uint64
}

type Want struct {
	Start uint64
	Len   uint64
}

type Request struct {
	Index uint64
}

type Data struct {
	Index     uint64
	Value     []byte
	Signature []byte
}

const MaxMessageSize = 10 * 1024 * 1024 // 10MB limit

// Helpers for encoding/decoding
func writeString(w io.Writer, s string) error {
	b := []byte(s)
	if err := binary.Write(w, binary.BigEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func readString(r io.Reader) (string, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}
	if length > MaxMessageSize {
		return "", errors.New("message string field exceeds safety limit")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func writeBytes(w io.Writer, b []byte) error {
	if err := binary.Write(w, binary.BigEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func readBytes(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length > MaxMessageSize {
		return nil, errors.New("message bytes field exceeds safety limit")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func EncodeHandshake(m *Handshake) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(0) // message type Handshake
	if err := writeString(&buf, m.Protocol); err != nil {
		return nil, err
	}
	if err := writeBytes(&buf, m.Key); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeHandshake(b []byte) (*Handshake, error) {
	if len(b) < 1 || b[0] != 0 {
		return nil, errors.New("invalid handshake message type")
	}
	r := bytes.NewReader(b[1:])
	protocol, err := readString(r)
	if err != nil {
		return nil, err
	}
	key, err := readBytes(r)
	if err != nil {
		return nil, err
	}
	return &Handshake{Protocol: protocol, Key: key}, nil
}

func EncodeHave(m *Have) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(2) // message type Have
	if err := binary.Write(&buf, binary.BigEndian, m.Start); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, m.Len); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeHave(b []byte) (*Have, error) {
	if len(b) < 1 || b[0] != 2 {
		return nil, errors.New("invalid Have message type")
	}
	r := bytes.NewReader(b[1:])
	var start, length uint64
	if err := binary.Read(r, binary.BigEndian, &start); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	return &Have{Start: start, Len: length}, nil
}

func EncodeWant(m *Want) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(3) // message type Want
	if err := binary.Write(&buf, binary.BigEndian, m.Start); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, m.Len); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeWant(b []byte) (*Want, error) {
	if len(b) < 1 || b[0] != 3 {
		return nil, errors.New("invalid Want message type")
	}
	r := bytes.NewReader(b[1:])
	var start, length uint64
	if err := binary.Read(r, binary.BigEndian, &start); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	return &Want{Start: start, Len: length}, nil
}

func EncodeRequest(m *Request) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(4) // message type Request
	if err := binary.Write(&buf, binary.BigEndian, m.Index); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeRequest(b []byte) (*Request, error) {
	if len(b) < 1 || b[0] != 4 {
		return nil, errors.New("invalid Request message type")
	}
	r := bytes.NewReader(b[1:])
	var index uint64
	if err := binary.Read(r, binary.BigEndian, &index); err != nil {
		return nil, err
	}
	return &Request{Index: index}, nil
}

func EncodeData(m *Data) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(5) // message type Data
	if err := binary.Write(&buf, binary.BigEndian, m.Index); err != nil {
		return nil, err
	}
	if err := writeBytes(&buf, m.Value); err != nil {
		return nil, err
	}
	if err := writeBytes(&buf, m.Signature); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeData(b []byte) (*Data, error) {
	if len(b) < 1 || b[0] != 5 {
		return nil, errors.New("invalid Data message type")
	}
	r := bytes.NewReader(b[1:])
	var index uint64
	if err := binary.Read(r, binary.BigEndian, &index); err != nil {
		return nil, err
	}
	value, err := readBytes(r)
	if err != nil {
		return nil, err
	}
	signature, err := readBytes(r)
	if err != nil {
		return nil, err
	}
	return &Data{Index: index, Value: value, Signature: signature}, nil
}
