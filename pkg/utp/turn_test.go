package utp

import (
	"bytes"
	"net"
	"testing"
)

func TestTURN_BuildAllocateRequest(t *testing.T) {
	req, txID, err := BuildTURNAllocateRequest()
	if err != nil {
		t.Fatalf("failed to build TURN Allocate Request: %v", err)
	}

	if len(req) < 20 {
		t.Fatalf("expected length >= 20, got %d", len(req))
	}

	// Verify type is 0x0003 (Allocate Request)
	msgType := uint16(req[0])<<8 | uint16(req[1])
	if msgType != 0x0003 {
		t.Errorf("expected TURN Allocate Request type 0x0003, got %04x", msgType)
	}

	// Verify magic cookie 0x2112A442
	cookie := req[4:8]
	expectedCookie := []byte{0x21, 0x12, 0xa4, 0x42}
	if !bytes.Equal(cookie, expectedCookie) {
		t.Errorf("expected magic cookie %v, got %v", expectedCookie, cookie)
	}

	// Verify Transaction ID matches returned txID
	if !bytes.Equal(req[8:20], txID[:]) {
		t.Errorf("transaction ID mismatch")
	}
}

func TestTURN_ParseAllocateResponse(t *testing.T) {
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	header := []byte{
		0x01, 0x03, // Type: Allocate Success Response (0x0103)
		0x00, 0x0c, // Length: 12 bytes of attributes
		0x21, 0x12, 0xa4, 0x42, // Magic Cookie
	}
	header = append(header, txID[:]...)

	// Attribute: XOR-RELAYED-ADDRESS (0x0016), Length (8)
	// Value: 1 byte reserved (0x00), 1 byte family (0x01 = IPv4)
	// X-Port: 12345 (0x3039) XOR 0x2112 = 0x112b
	// X-Address: 192.168.1.100 (0xc0a80164) XOR 0x2112A442 = 0xe1baa526
	attr := []byte{
		0x00, 0x16, // Type
		0x00, 0x08, // Length
		0x00, 0x01, // Reserved + Family
		0x11, 0x2b, // X-Port
		0xe1, 0xba, 0xa5, 0x26, // X-Address
	}

	response := append(header, attr...)

	ip, port, err := ParseTURNAllocateResponse(response, txID)
	if err != nil {
		t.Fatalf("failed to parse TURN response: %v", err)
	}

	expectedIP := "192.168.1.100"
	if ip.String() != expectedIP {
		t.Errorf("expected IP %s, got %s", expectedIP, ip.String())
	}

	expectedPort := 12345
	if port != expectedPort {
		t.Errorf("expected port %d, got %d", expectedPort, port)
	}
}

func TestTURN_BuildCreatePermissionRequest(t *testing.T) {
	peerIP := net.ParseIP("192.168.1.200")
	peerPort := 54321

	req, _, err := BuildTURNCreatePermissionRequest(peerIP, peerPort)
	if err != nil {
		t.Fatalf("failed to build CreatePermission: %v", err)
	}

	// Verify type is 0x0008 (CreatePermission Request)
	msgType := uint16(req[0])<<8 | uint16(req[1])
	if msgType != 0x0008 {
		t.Errorf("expected CreatePermission Request type 0x0008, got %04x", msgType)
	}
}

func TestTURN_SendAndDataIndication(t *testing.T) {
	peerIP := net.ParseIP("192.168.1.200")
	peerPort := 54321
	payload := []byte("hello turn relay")

	// 1. Build Send Indication
	sendInd, err := BuildTURNSendIndication(peerIP, peerPort, payload)
	if err != nil {
		t.Fatalf("failed to build SendIndication: %v", err)
	}

	// Verify type is 0x0016 (Send Indication)
	msgType := uint16(sendInd[0])<<8 | uint16(sendInd[1])
	if msgType != 0x0016 {
		t.Errorf("expected Send Indication type 0x0016, got %04x", msgType)
	}

	// 2. Parse Data Indication
	// Header: Data Indication (0x0017), Length of attributes
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	// Attribute: XOR-PEER-ADDRESS (0x0012), Length (8)
	peerAttr := []byte{
		0x00, 0x12, // Type
		0x00, 0x08, // Length
		0x00, 0x01, // Reserved + Family
		0x30, 0x39, // Port (un-XOR'ed in some indications, but let's assume XOR standard)
		0xc0, 0xa8, 0x01, 0xc8,
	}
	// Attribute: DATA (0x0013), Length (16)
	dataAttr := []byte{
		0x00, 0x13, // Type
		0x00, 0x10, // Length
	}
	dataAttr = append(dataAttr, payload...)

	header := []byte{
		0x00, 0x17, // Type: Data Indication
		0x00, uint8(len(peerAttr) + len(dataAttr)), // Length
		0x21, 0x12, 0xa4, 0x42, // Cookie
	}
	header = append(header, txID[:]...)

	indPacket := append(header, peerAttr...)
	indPacket = append(indPacket, dataAttr...)

	parsedPeerIP, parsedPeerPort, parsedPayload, err := ParseTURNDataIndication(indPacket)
	if err != nil {
		t.Fatalf("failed to parse Data Indication: %v", err)
	}

	_ = parsedPeerIP
	_ = parsedPeerPort
	if !bytes.Equal(parsedPayload, payload) {
		t.Errorf("expected payload %q, got %q", string(payload), string(parsedPayload))
	}
}

func TestTURN_ParseAllocateResponseErrors(t *testing.T) {
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	// 1. Packet too short
	_, _, err := ParseTURNAllocateResponse([]byte{0, 1}, txID)
	if err == nil {
		t.Error("expected error for short packet")
	}

	// 2. Wrong message type
	wrongType := []byte{0x00, 0x01, 0, 0, 0x21, 0x12, 0xa4, 0x42}
	wrongType = append(wrongType, txID[:]...)
	_, _, err = ParseTURNAllocateResponse(wrongType, txID)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestTURN_ParseDataIndicationErrors(t *testing.T) {
	// 1. Indication too short
	_, _, _, err := ParseTURNDataIndication([]byte{0, 1})
	if err == nil {
		t.Error("expected error for short packet")
	}

	// 2. Wrong message type
	_, _, _, err = ParseTURNDataIndication(make([]byte, 20))
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestTURN_ParseAllocateResponseIPv6(t *testing.T) {
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	header := []byte{
		0x01, 0x03, // Type: Allocate Success Response
		0x00, 0x18, // Length: 24 bytes
		0x21, 0x12, 0xa4, 0x42, // Cookie
	}
	header = append(header, txID[:]...)

	// XOR-RELAYED-ADDRESS IPv6 (0x0016), Length (20)
	attr := []byte{
		0x00, 0x16, // Type
		0x00, 0x14, // Length (20)
		0x00, 0x02, // Reserved + Family (IPv6)
		0x11, 0x2b, // X-Port
		0x21, 0x12, 0xa4, 0x42, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	}

	response := append(header, attr...)

	ip, port, err := ParseTURNAllocateResponse(response, txID)
	if err != nil {
		t.Fatalf("failed to parse IPv6 TURN response: %v", err)
	}

	expectedIP := "::"
	if ip.String() != expectedIP {
		t.Errorf("expected IP %s, got %s", expectedIP, ip.String())
	}

	expectedPort := 12345
	if port != expectedPort {
		t.Errorf("expected port %d, got %d", expectedPort, port)
	}
}


