package hypercore

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/db"
)

type SyncSession struct {
	conn        net.Conn
	storage     *Storage
	blockRepo   *db.BlockRepository
	feedKey     string
	remotePub   ed25519.PublicKey
	localPriv   ed25519.PrivateKey
	isInitiator bool
	writeMu     sync.Mutex
	OnAppendBlock func(index uint64, value []byte)
}

func NewSyncSession(conn net.Conn, storage *Storage, blockRepo *db.BlockRepository, feedKey string, localPriv ed25519.PrivateKey, remotePub ed25519.PublicKey, isInitiator bool) *SyncSession {
	return &SyncSession{
		conn:        conn,
		storage:     storage,
		blockRepo:   blockRepo,
		feedKey:     feedKey,
		localPriv:   localPriv,
		remotePub:   remotePub,
		isInitiator: isInitiator,
	}
}

func (s *SyncSession) NotifyHave(length uint64) error {
	haveMsg := &Have{
		Start: 0,
		Len:   length,
	}
	haveBytes, err := EncodeHave(haveMsg)
	if err != nil {
		return err
	}
	return s.writeFrame(haveBytes)
}

func (s *SyncSession) Run(ctx context.Context) error {
	// 1. Send Handshake
	localHandshake := &Handshake{
		Protocol: "hypercore/v10",
		Key:      []byte(s.feedKey),
	}
	hb, err := EncodeHandshake(localHandshake)
	if err != nil {
		return fmt.Errorf("encode handshake: %w", err)
	}
	if err := s.writeFrame(hb); err != nil {
		return fmt.Errorf("write handshake: %w", err)
	}

	// 2. Read Handshake
	rb, err := s.readFrame()
	if err != nil {
		return fmt.Errorf("read handshake: %w", err)
	}
	remoteHandshake, err := DecodeHandshake(rb)
	if err != nil {
		return fmt.Errorf("decode handshake: %w", err)
	}
	if remoteHandshake.Protocol != "hypercore/v10" {
		return fmt.Errorf("unsupported remote protocol: %q", remoteHandshake.Protocol)
	}

	// 3. Exchange Have & Want
	localLen := s.storage.Len()
	haveMsg := &Have{
		Start: 0,
		Len:   localLen,
	}
	haveBytes, err := EncodeHave(haveMsg)
	if err != nil {
		return fmt.Errorf("encode have: %w", err)
	}
	if err := s.writeFrame(haveBytes); err != nil {
		return fmt.Errorf("write have: %w", err)
	}

	// Initially want any new blocks the remote peer has
	wantMsg := &Want{
		Start: localLen,
		Len:   1000000,
	}
	wantBytes, err := EncodeWant(wantMsg)
	if err != nil {
		return fmt.Errorf("encode want: %w", err)
	}
	if err := s.writeFrame(wantBytes); err != nil {
		return fmt.Errorf("write want: %w", err)
	}

	// Setup channels and context
	errChan := make(chan error, 2)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 4. Start concurrent read loop
	go func() {
		errChan <- s.readLoop(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func (s *SyncSession) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read message frame from connection
			b, err := s.readFrame()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("read loop frame: %w", err)
			}

			if len(b) < 1 {
				continue
			}

			msgType := b[0]
			switch msgType {
			case 2: // Have
				have, err := DecodeHave(b)
				if err != nil {
					return fmt.Errorf("decode have: %w", err)
				}
				slog.Debug("P2P Have block details received", "start", have.Start, "len", have.Len)
				localLen := s.storage.Len()
				if have.Len > localLen {
					// We need to request missing blocks sequentially
					for i := localLen; i < have.Len; i++ {
						reqMsg := &Request{Index: i}
						reqBytes, err := EncodeRequest(reqMsg)
						if err != nil {
							return err
						}
						if err := s.writeFrame(reqBytes); err != nil {
							return err
						}
					}
				}
			case 3: // Want
				want, err := DecodeWant(b)
				if err != nil {
					return fmt.Errorf("decode want: %w", err)
				}
				slog.Debug("P2P Want block range received", "start", want.Start, "len", want.Len)
				// Send whatever matching blocks we have locally
				localLen := s.storage.Len()
				for i := want.Start; i < want.Start+want.Len && i < localLen; i++ {
					if err := s.sendBlockData(ctx, i); err != nil {
						return err
					}
				}
			case 4: // Request
				req, err := DecodeRequest(b)
				if err != nil {
					return fmt.Errorf("decode request: %w", err)
				}
				if err := s.sendBlockData(ctx, req.Index); err != nil {
					return err
				}
			case 5: // Data
				dataMsg, err := DecodeData(b)
				if err != nil {
					return fmt.Errorf("decode data: %w", err)
				}

				// Only append if it's the next expected block index
				expectedIndex := s.storage.Len()
				if dataMsg.Index == expectedIndex {
					// Load all existing blocks to calculate new root hash
					leaves := make([][]byte, expectedIndex+1)
					for i := uint64(0); i < expectedIndex; i++ {
						val, err := s.storage.Get(i)
						if err != nil {
							return fmt.Errorf("read local block for signature verification: %w", err)
						}
						leaves[i] = val
					}
					leaves[expectedIndex] = dataMsg.Value

					rootHash, err := ComputeRootHash(leaves)
					if err != nil {
						return fmt.Errorf("compute root hash for block: %w", err)
					}

					// Verify cryptographic signature
					if !VerifySignature(s.remotePub, rootHash, dataMsg.Signature) {
						return fmt.Errorf("cryptographic verification failed for block %d", dataMsg.Index)
					}

					// Append to storage
					if err := s.storage.Append(dataMsg.Value); err != nil {
						return fmt.Errorf("append block: %w", err)
					}

					// Save in database repository if available
					if s.blockRepo != nil {
						if err := s.blockRepo.PutBlock(ctx, s.feedKey, dataMsg.Index, dataMsg.Value, dataMsg.Signature); err != nil {
							slog.Error("Failed to save replicated block in DB repository", "index", dataMsg.Index, "err", err)
						}
					}
					if s.OnAppendBlock != nil {
						s.OnAppendBlock(dataMsg.Index, dataMsg.Value)
					}

					slog.Info("Successfully verified and appended replicated block", "index", dataMsg.Index)
				}
			}
		}
	}
}

func (s *SyncSession) sendBlockData(ctx context.Context, index uint64) error {
	var value, signature []byte
	var err error

	// Load from database if repo is available
	if s.blockRepo != nil {
		value, signature, err = s.blockRepo.GetBlock(ctx, s.feedKey, index)
	}

	// Fallback to flat file storage
	if err != nil || value == nil {
		value, err = s.storage.Get(index)
		if err != nil {
			return nil // We don't have this block, ignore
		}
		// If we don't have the signature (flat-file only), sign it locally
		leaves := make([][]byte, index+1)
		for i := uint64(0); i <= index; i++ {
			val, _ := s.storage.Get(i)
			leaves[i] = val
		}
		rootHash, _ := ComputeRootHash(leaves)
		signature = SignRootHash(s.localPriv, rootHash)
	}

	dataMsg := &Data{
		Index:     index,
		Value:     value,
		Signature: signature,
	}
	dataBytes, err := EncodeData(dataMsg)
	if err != nil {
		return err
	}
	return s.writeFrame(dataBytes)
}

func (s *SyncSession) writeFrame(payload []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	length := uint32(len(payload))
	if err := binary.Write(s.conn, binary.BigEndian, length); err != nil {
		return err
	}
	_, err := s.conn.Write(payload)
	return err
}

func (s *SyncSession) readFrame() ([]byte, error) {
	s.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var length uint32
	if err := binary.Read(s.conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length > MaxMessageSize {
		return nil, fmt.Errorf("frame size %d exceeds safety limit", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(s.conn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
