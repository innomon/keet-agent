package utp

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
)

const (
	attrXorPeerAddress     = 0x0012
	attrData               = 0x0013
	attrXorRelayedAddress  = 0x0016
	attrRequestedTransport = 0x0019
)

// BuildTURNAllocateRequest constructs a TURN Allocate Request packet (with REQUESTED-TRANSPORT attribute).
func BuildTURNAllocateRequest() ([]byte, [12]byte, error) {
	var txID [12]byte
	_, err := rand.Read(txID[:])
	if err != nil {
		return nil, txID, err
	}

	// 20 bytes header + 8 bytes REQUESTED-TRANSPORT attribute
	packet := make([]byte, 28)
	// Message Type: Allocate Request (0x0003)
	binary.BigEndian.PutUint16(packet[0:2], 0x0003)
	// Message Length: 8 bytes
	binary.BigEndian.PutUint16(packet[2:4], 0x0008)
	// Magic Cookie: 0x2112A442
	binary.BigEndian.PutUint32(packet[4:8], stunMagicCookie)
	// Transaction ID
	copy(packet[8:20], txID[:])

	// Attribute: REQUESTED-TRANSPORT (0x0019), Length (4)
	binary.BigEndian.PutUint16(packet[20:22], attrRequestedTransport)
	binary.BigEndian.PutUint16(packet[22:24], 0x0004)
	// Protocol: UDP (17), Reserved (0, 0, 0)
	packet[24] = 17
	packet[25] = 0
	packet[26] = 0
	packet[27] = 0

	return packet, txID, nil
}

// ParseTURNAllocateResponse decodes a TURN Allocate Response and extracts the relayed IP and Port.
func ParseTURNAllocateResponse(data []byte, expectedTxID [12]byte) (net.IP, int, error) {
	if len(data) < 20 {
		return nil, 0, errors.New("TURN packet too short")
	}

	msgType := binary.BigEndian.Uint16(data[0:2])
	if msgType != 0x0103 { // Allocate Success Response
		return nil, 0, errors.New("not a TURN Allocate success response")
	}

	length := int(binary.BigEndian.Uint16(data[2:4]))
	cookie := binary.BigEndian.Uint32(data[4:8])
	if cookie != stunMagicCookie {
		return nil, 0, errors.New("invalid STUN magic cookie")
	}

	for i := 0; i < 12; i++ {
		if data[8+i] != expectedTxID[i] {
			return nil, 0, errors.New("TURN transaction ID mismatch")
		}
	}

	if len(data) < 20+length {
		return nil, 0, errors.New("TURN packet attribute length mismatch")
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

		if attrType == attrXorRelayedAddress {
			if len(attrValue) < 8 {
				continue
			}
			family := attrValue[1]
			rawPort := binary.BigEndian.Uint16(attrValue[2:4])
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

	return nil, 0, errors.New("no relayed address found in TURN response")
}

// BuildTURNCreatePermissionRequest constructs a TURN CreatePermission Request for a peer address.
func BuildTURNCreatePermissionRequest(peerIP net.IP, peerPort int) ([]byte, [12]byte, error) {
	var txID [12]byte
	_, err := rand.Read(txID[:])
	if err != nil {
		return nil, txID, err
	}

	ipv4 := peerIP.To4()
	if ipv4 == nil {
		return nil, txID, errors.New("only IPv4 peers are currently supported for permissions")
	}

	// 20 bytes header + 12 bytes XOR-PEER-ADDRESS attribute
	packet := make([]byte, 32)
	binary.BigEndian.PutUint16(packet[0:2], 0x0008) // CreatePermission Request
	binary.BigEndian.PutUint16(packet[2:4], 12)     // Attribute length
	binary.BigEndian.PutUint32(packet[4:8], stunMagicCookie)
	copy(packet[8:20], txID[:])

	// Attribute: XOR-PEER-ADDRESS
	binary.BigEndian.PutUint16(packet[20:22], attrXorPeerAddress)
	binary.BigEndian.PutUint16(packet[22:24], 8)
	packet[24] = 0 // Reserved
	packet[25] = 1 // IPv4 Family

	// X-Port
	binary.BigEndian.PutUint16(packet[26:28], uint16(peerPort)^(stunMagicCookie>>16))
	// X-Address
	ipVal := binary.BigEndian.Uint32(ipv4) ^ stunMagicCookie
	binary.BigEndian.PutUint32(packet[28:32], ipVal)

	return packet, txID, nil
}

// BuildTURNSendIndication wraps payload into a TURN Send Indication targeting the peer.
func BuildTURNSendIndication(peerIP net.IP, peerPort int, payload []byte) ([]byte, error) {
	ipv4 := peerIP.To4()
	if ipv4 == nil {
		return nil, errors.New("only IPv4 peers are supported for Send Indication")
	}

	// Attributes: XOR-PEER-ADDRESS (12 bytes) + DATA (4 bytes type/len + padded payload)
	paddedPayloadLen := (len(payload) + 3) &^ 3
	attrLen := 12 + 4 + paddedPayloadLen

	packet := make([]byte, 20+attrLen)
	binary.BigEndian.PutUint16(packet[0:2], 0x0016) // Send Indication
	binary.BigEndian.PutUint16(packet[2:4], uint16(attrLen))
	binary.BigEndian.PutUint32(packet[4:8], stunMagicCookie)
	_, _ = rand.Read(packet[8:20])

	// 1. XOR-PEER-ADDRESS
	binary.BigEndian.PutUint16(packet[20:22], attrXorPeerAddress)
	binary.BigEndian.PutUint16(packet[22:24], 8)
	packet[24] = 0
	packet[25] = 1
	binary.BigEndian.PutUint16(packet[26:28], uint16(peerPort)^(stunMagicCookie>>16))
	ipVal := binary.BigEndian.Uint32(ipv4) ^ stunMagicCookie
	binary.BigEndian.PutUint32(packet[28:32], ipVal)

	// 2. DATA
	binary.BigEndian.PutUint16(packet[32:34], attrData)
	binary.BigEndian.PutUint16(packet[34:36], uint16(len(payload)))
	copy(packet[36:36+len(payload)], payload)

	return packet, nil
}

// ParseTURNDataIndication parses a TURN Data Indication and returns sender address and payload.
func ParseTURNDataIndication(data []byte) (net.IP, int, []byte, error) {
	if len(data) < 20 {
		return nil, 0, nil, errors.New("TURN indication packet too short")
	}

	msgType := binary.BigEndian.Uint16(data[0:2])
	if msgType != 0x0017 { // Data Indication
		return nil, 0, nil, errors.New("not a TURN Data Indication")
	}

	length := int(binary.BigEndian.Uint16(data[2:4]))
	cookie := binary.BigEndian.Uint32(data[4:8])
	if cookie != stunMagicCookie {
		return nil, 0, nil, errors.New("invalid STUN magic cookie")
	}

	if len(data) < 20+length {
		return nil, 0, nil, errors.New("TURN packet length mismatch")
	}

	var peerIP net.IP
	var peerPort int
	var payload []byte

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
		case attrXorPeerAddress:
			if len(attrValue) < 8 {
				continue
			}
			family := attrValue[1]
			rawPort := binary.BigEndian.Uint16(attrValue[2:4])
			peerPort = int(rawPort ^ (stunMagicCookie >> 16))

			if family == 1 { // IPv4
				xAddress := binary.BigEndian.Uint32(attrValue[4:8])
				ipVal := xAddress ^ stunMagicCookie
				ipBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(ipBytes, ipVal)
				peerIP = net.IP(ipBytes)
			}
		case attrData:
			payload = make([]byte, len(attrValue))
			copy(payload, attrValue)
		}
	}

	if len(payload) == 0 {
		return nil, 0, nil, errors.New("no DATA attribute in indication")
	}

	return peerIP, peerPort, payload, nil
}
