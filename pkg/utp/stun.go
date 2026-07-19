package utp

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
)

const (
	stunMagicCookie = 0x2112A442
	attrMappedAddr  = 0x0001
	attrXorMapped   = 0x0020
)

// BuildSTUNBindingRequest constructs a standard 20-byte STUN Binding Request packet.
// Returns the raw packet bytes and the random 12-byte Transaction ID.
func BuildSTUNBindingRequest() ([]byte, [12]byte, error) {
	var txID [12]byte
	_, err := rand.Read(txID[:])
	if err != nil {
		return nil, txID, err
	}

	packet := make([]byte, 20)
	// Message Type: Binding Request (0x0001)
	binary.BigEndian.PutUint16(packet[0:2], 0x0001)
	// Message Length: 0 (no attributes)
	binary.BigEndian.PutUint16(packet[2:4], 0x0000)
	// Magic Cookie: 0x2112A442
	binary.BigEndian.PutUint32(packet[4:8], stunMagicCookie)
	// Transaction ID
	copy(packet[8:20], txID[:])

	return packet, txID, nil
}

// ParseSTUNBindingResponse decodes a STUN response and returns the public IP and Port.
func ParseSTUNBindingResponse(data []byte, expectedTxID [12]byte) (net.IP, int, error) {
	if len(data) < 20 {
		return nil, 0, errors.New("STUN packet too short")
	}

	msgType := binary.BigEndian.Uint16(data[0:2])
	if msgType != 0x0101 { // Binding Success Response
		return nil, 0, errors.New("not a STUN success response")
	}

	length := int(binary.BigEndian.Uint16(data[2:4]))
	cookie := binary.BigEndian.Uint32(data[4:8])
	if cookie != stunMagicCookie {
		return nil, 0, errors.New("invalid STUN magic cookie")
	}

	for i := 0; i < 12; i++ {
		if data[8+i] != expectedTxID[i] {
			return nil, 0, errors.New("STUN transaction ID mismatch")
		}
	}

	if len(data) < 20+length {
		return nil, 0, errors.New("STUN packet attribute length mismatch")
	}

	offset := 20
	end := 20 + length

	for offset < end {
		if offset+4 > end {
			break
		}
		attrType := binary.BigEndian.Uint16(data[offset : offset+2])
		attrLen := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		offset += 4

		if offset+attrLen > end {
			break
		}

		attrValue := data[offset : offset+attrLen]
		paddedLen := (attrLen + 3) &^ 3
		offset += paddedLen

		switch attrType {
		case attrMappedAddr:
			if len(attrValue) < 8 {
				continue
			}
			family := attrValue[1]
			port := int(binary.BigEndian.Uint16(attrValue[2:4]))
			if family == 1 { // IPv4
				ip := net.IP(attrValue[4:8])
				return ip, port, nil
			} else if family == 2 { // IPv6
				if len(attrValue) >= 20 {
					ip := net.IP(attrValue[4:20])
					return ip, port, nil
				}
			}

		case attrXorMapped:
			if len(attrValue) < 8 {
				continue
			}
			family := attrValue[1]
			rawPort := binary.BigEndian.Uint16(attrValue[2:4])
			// Port is XOR'ed with the most significant 16 bits of the Magic Cookie
			port := int(rawPort ^ (stunMagicCookie >> 16))

			if family == 1 { // IPv4
				xAddress := binary.BigEndian.Uint32(attrValue[4:8])
				ipVal := xAddress ^ stunMagicCookie
				ipBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(ipBytes, ipVal)
				return net.IP(ipBytes), port, nil
			} else if family == 2 { // IPv6
				if len(attrValue) >= 20 {
					xAddress := attrValue[4:20]
					ipBytes := make([]byte, 16)
					// IPv6 address is XOR'ed with Magic Cookie concat Transaction ID
					var xorKey [16]byte
					binary.BigEndian.PutUint32(xorKey[0:4], stunMagicCookie)
					copy(xorKey[4:16], expectedTxID[:])

					for j := 0; j < 16; j++ {
						ipBytes[j] = xAddress[j] ^ xorKey[j]
					}
					return net.IP(ipBytes), port, nil
				}
			}
		}
	}

	return nil, 0, errors.New("no mapped address found in STUN response")
}
