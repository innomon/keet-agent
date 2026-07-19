package crypto

import (
	"crypto/ed25519"
	"crypto/sha512"
	"errors"
	"io"
	"net"
	"sync"

	"filippo.io/edwards25519"
	"github.com/flynn/noise"
	"golang.org/x/crypto/curve25519"
)

type SecureConn struct {
	net.Conn
	csSend  *noise.CipherState
	csRecv  *noise.CipherState
	readBuf []byte
	writeMu sync.Mutex
	readMu  sync.Mutex
}

func NewSecureConnection(conn net.Conn, localPriv ed25519.PrivateKey, initiator bool) (net.Conn, ed25519.PublicKey, error) {
	cipherSuite := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2b)

	localXPrivate := ed25519PrivateKeyToX25519(localPriv)
	localXPublic, err := curve25519.X25519(localXPrivate, curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	config := noise.Config{
		CipherSuite: cipherSuite,
		Pattern:     noise.HandshakeXX,
		Initiator:   initiator,
		StaticKeypair: noise.DHKey{
			Private: localXPrivate,
			Public:  localXPublic,
		},
	}

	hs, err := noise.NewHandshakeState(config)
	if err != nil {
		return nil, nil, err
	}

	var csSend, csRecv *noise.CipherState
	var remotePub ed25519.PublicKey
	localPub := localPriv.Public().(ed25519.PublicKey)

	if initiator {
		// Msg 1: Write (-> e)
		msg, _, _, err := hs.WriteMessage(nil, nil)
		if err != nil {
			return nil, nil, err
		}
		if err := writeFrame(conn, msg); err != nil {
			return nil, nil, err
		}

		// Msg 2: Read (<- e, ee, s, es)
		payload, err := readFrame(conn)
		if err != nil {
			return nil, nil, err
		}
		res, _, _, err := hs.ReadMessage(nil, payload)
		if err != nil {
			return nil, nil, err
		}
		if len(res) == ed25519.PublicKeySize {
			remotePub = make([]byte, ed25519.PublicKeySize)
			copy(remotePub, res)
		}

		// Msg 3: Write (-> s, se)
		msg, cs1, cs2, err := hs.WriteMessage(nil, localPub)
		if err != nil {
			return nil, nil, err
		}
		if err := writeFrame(conn, msg); err != nil {
			return nil, nil, err
		}
		csSend = cs1
		csRecv = cs2
	} else {
		// Msg 1: Read (-> e)
		payload, err := readFrame(conn)
		if err != nil {
			return nil, nil, err
		}
		_, _, _, err = hs.ReadMessage(nil, payload)
		if err != nil {
			return nil, nil, err
		}

		// Msg 2: Write (<- e, ee, s, es)
		msg, _, _, err := hs.WriteMessage(nil, localPub)
		if err != nil {
			return nil, nil, err
		}
		if err := writeFrame(conn, msg); err != nil {
			return nil, nil, err
		}

		// Msg 3: Read (-> s, se)
		payload, err = readFrame(conn)
		if err != nil {
			return nil, nil, err
		}
		var cs1, cs2 *noise.CipherState
		res, cs1, cs2, err := hs.ReadMessage(nil, payload)
		if err != nil {
			return nil, nil, err
		}
		if len(res) == ed25519.PublicKeySize {
			remotePub = make([]byte, ed25519.PublicKeySize)
			copy(remotePub, res)
		}
		csSend = cs2
		csRecv = cs1
	}

	return &SecureConn{
		Conn:   conn,
		csSend: csSend,
		csRecv: csRecv,
	}, remotePub, nil
}

func (s *SecureConn) Write(p []byte) (int, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	encrypted, err := s.csSend.Encrypt(nil, nil, p)
	if err != nil {
		return 0, err
	}
	if err := writeFrame(s.Conn, encrypted); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *SecureConn) Read(p []byte) (int, error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		return n, nil
	}

	encrypted, err := readFrame(s.Conn)
	if err != nil {
		return 0, err
	}

	decrypted, err := s.csRecv.Decrypt(nil, nil, encrypted)
	if err != nil {
		return 0, err
	}

	n := copy(p, decrypted)
	if n < len(decrypted) {
		s.readBuf = decrypted[n:]
	}
	return n, nil
}

func ed25519PublicKeyToX25519(edPub ed25519.PublicKey) ([]byte, error) {
	p, err := new(edwards25519.Point).SetBytes(edPub)
	if err != nil {
		return nil, err
	}
	return p.BytesMontgomery(), nil
}

func ed25519PrivateKeyToX25519(edPriv ed25519.PrivateKey) []byte {
	seed := edPriv.Seed()
	h := sha512.Sum512(seed)
	scalar := h[:32]
	scalar[0] &= 248
	scalar[31] &= 127
	scalar[31] |= 64
	return scalar
}

func writeFrame(conn net.Conn, data []byte) error {
	length := len(data)
	if length > 65535 {
		return errors.New("packet too large for Noise framing")
	}
	header := []byte{byte(length >> 8), byte(length)}
	_, err := conn.Write(header)
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}

func readFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}
	length := (int(header[0]) << 8) | int(header[1])
	data := make([]byte, length)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
