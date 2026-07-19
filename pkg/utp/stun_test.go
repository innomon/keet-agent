package utp

import (
	"bytes"
	"testing"
)

func TestSTUN_BuildBindingRequest(t *testing.T) {
	req, txID, err := BuildSTUNBindingRequest()
	if err != nil {
		t.Fatalf("failed to build STUN request: %v", err)
	}

	if len(req) != 20 {
		t.Errorf("expected header length 20, got %d", len(req))
	}

	// Verify type is 0x0001 (Binding Request)
	if req[0] != 0x00 || req[1] != 0x01 {
		t.Errorf("expected message type 0x0001, got %x%x", req[0], req[1])
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

func TestSTUN_ParseBindingResponse(t *testing.T) {
	// Construct a mock STUN XOR-MAPPED-ADDRESS response
	// Header: Success Response (0x0101), Length (12), Magic Cookie (0x2112A442)
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	header := []byte{
		0x01, 0x01, // Type: Binding Success Response
		0x00, 0x0c, // Length: 12 bytes of attributes
		0x21, 0x12, 0xa4, 0x42, // Magic Cookie
	}
	header = append(header, txID[:]...)

	// Attribute: XOR-MAPPED-ADDRESS (0x0020), Length (8)
	// Value: 1 byte reserved (0x00), 1 byte family (0x01 = IPv4)
	// X-Port: 12345 (0x3039) XOR 0x2112 = 0x112b
	// X-Address: 192.168.1.100 (0xc0a80164) XOR 0x2112A442 = 0xe1baa526
	attr := []byte{
		0x00, 0x20, // Type
		0x00, 0x08, // Length
		0x00, 0x01, // Reserved + Family
		0x11, 0x2b, // X-Port
		0xe1, 0xba, 0xa5, 0x26, // X-Address
	}

	response := append(header, attr...)

	ip, port, err := ParseSTUNBindingResponse(response, txID)
	if err != nil {
		t.Fatalf("failed to parse STUN response: %v", err)
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

func TestSTUN_ParseBindingResponseErrors(t *testing.T) {
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	// 1. Packet too short
	_, _, err := ParseSTUNBindingResponse([]byte{0, 1}, txID)
	if err == nil {
		t.Error("expected error for short packet")
	}

	// 2. Wrong message type
	wrongType := []byte{0x00, 0x01, 0, 0, 0x21, 0x12, 0xa4, 0x42}
	wrongType = append(wrongType, txID[:]...)
	_, _, err = ParseSTUNBindingResponse(wrongType, txID)
	if err == nil {
		t.Error("expected error for wrong type")
	}

	// 3. Wrong cookie
	wrongCookie := []byte{0x01, 0x01, 0, 0, 0, 0, 0, 0}
	wrongCookie = append(wrongCookie, txID[:]...)
	_, _, err = ParseSTUNBindingResponse(wrongCookie, txID)
	if err == nil {
		t.Error("expected error for wrong cookie")
	}

	// 4. Wrong Transaction ID
	wrongTx := []byte{0x01, 0x01, 0, 0, 0x21, 0x12, 0xa4, 0x42}
	wrongTx = append(wrongTx, make([]byte, 12)...)
	_, _, err = ParseSTUNBindingResponse(wrongTx, txID)
	if err == nil {
		t.Error("expected error for wrong transaction ID")
	}
}

func TestSTUN_ParseBindingResponseMappedAddress(t *testing.T) {
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	header := []byte{
		0x01, 0x01, // Type: Binding Success Response
		0x00, 0x0c, // Length: 12 bytes of attributes
		0x21, 0x12, 0xa4, 0x42, // Magic Cookie
	}
	header = append(header, txID[:]...)

	// Attribute: MAPPED-ADDRESS (0x0001), Length (8)
	// Value: 1 byte reserved (0x00), 1 byte family (0x01 = IPv4)
	// Port: 12345 (0x3039)
	// Address: 192.168.1.100 (0xc0a80164)
	attr := []byte{
		0x00, 0x01, // Type
		0x00, 0x08, // Length
		0x00, 0x01, // Reserved + Family
		0x30, 0x39, // Port
		0xc0, 0xa8, 0x01, 0x64, // Address
	}

	response := append(header, attr...)

	ip, port, err := ParseSTUNBindingResponse(response, txID)
	if err != nil {
		t.Fatalf("failed to parse standard MAPPED-ADDRESS STUN response: %v", err)
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

func TestSTUN_ParseBindingResponseUnknownAttribute(t *testing.T) {
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	header := []byte{
		0x01, 0x01, // Type: Binding Success Response
		0x00, 0x08, // Length: 8 bytes of attributes
		0x21, 0x12, 0xa4, 0x42, // Magic Cookie
	}
	header = append(header, txID[:]...)

	// Attribute: Unknown (0x9999), Length (4)
	attr := []byte{
		0x99, 0x99, // Type
		0x00, 0x04, // Length
		0x00, 0x00, 0x00, 0x00, // Value
	}

	response := append(header, attr...)

	_, _, err := ParseSTUNBindingResponse(response, txID)
	if err == nil {
		t.Error("expected error since no mapped address is in the response")
	}
}


