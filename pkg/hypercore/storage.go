package hypercore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type indexEntry struct {
	offset uint64
	length uint64
}

type Storage struct {
	mu        sync.RWMutex
	dataF     *os.File
	indexF    *os.File
	entries   []indexEntry
	dataSize  int64
}

func NewStorage(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	dataPath := filepath.Join(dir, "data.log")
	indexPath := filepath.Join(dir, "index.log")

	dataF, err := os.OpenFile(dataPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open data file: %w", err)
	}

	indexF, err := os.OpenFile(indexPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		dataF.Close()
		return nil, fmt.Errorf("open index file: %w", err)
	}

	// Get data file size
	dataInfo, err := dataF.Stat()
	if err != nil {
		dataF.Close()
		indexF.Close()
		return nil, err
	}
	dataSize := dataInfo.Size()

	// Read index entries
	var entries []indexEntry
	var buf [16]byte
	for {
		_, err := io.ReadFull(indexF, buf[:])
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			dataF.Close()
			indexF.Close()
			return nil, fmt.Errorf("read index: %w", err)
		}
		offset := binary.BigEndian.Uint64(buf[0:8])
		length := binary.BigEndian.Uint64(buf[8:16])
		entries = append(entries, indexEntry{offset: offset, length: length})
	}

	// Seek data file to end
	if _, err := dataF.Seek(0, io.SeekEnd); err != nil {
		dataF.Close()
		indexF.Close()
		return nil, err
	}

	return &Storage{
		dataF:    dataF,
		indexF:   indexF,
		entries:  entries,
		dataSize: dataSize,
	}, nil
}

func (s *Storage) Append(block []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	offset := uint64(s.dataSize)
	length := uint64(len(block))

	// Write payload
	n, err := s.dataF.Write(block)
	if err != nil {
		return fmt.Errorf("write data block: %w", err)
	}
	s.dataSize += int64(n)

	// Write index
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], offset)
	binary.BigEndian.PutUint64(buf[8:16], length)

	if _, err := s.indexF.Write(buf[:]); err != nil {
		return fmt.Errorf("write index block: %w", err)
	}

	s.entries = append(s.entries, indexEntry{offset: offset, length: length})
	return nil
}

func (s *Storage) Get(index uint64) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index >= uint64(len(s.entries)) {
		return nil, errors.New("index out of bounds")
	}

	entry := s.entries[index]
	buf := make([]byte, entry.length)

	_, err := s.dataF.ReadAt(buf, int64(entry.offset))
	if err != nil {
		return nil, fmt.Errorf("read data block: %w", err)
	}

	return buf, nil
}

func (s *Storage) Len() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return uint64(len(s.entries))
}

func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	if s.dataF != nil {
		if err := s.dataF.Close(); err != nil {
			errs = append(errs, err)
		}
		s.dataF = nil
	}
	if s.indexF != nil {
		if err := s.indexF.Close(); err != nil {
			errs = append(errs, err)
		}
		s.indexF = nil
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
