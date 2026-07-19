package utp

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

// CandidateType represents the type of an ICE candidate.
type CandidateType string

const (
	// CandidateHost represents a local interface candidate.
	CandidateHost CandidateType = "host"
	// CandidateSrflx represents a server-reflexive candidate (discovered via STUN).
	CandidateSrflx CandidateType = "srflx"
	// CandidateRelay represents a relayed candidate (discovered via TURN).
	CandidateRelay CandidateType = "relay"
)

// ICECandidate holds connection information and classification for NAT traversal.
type ICECandidate struct {
	Addr     string
	Type     CandidateType
	Priority uint32
}

// ICESession coordinates candidate gathering, prioritization, and connectivity checks.
type ICESession struct {
	localCandidates  []ICECandidate
	remoteCandidates []ICECandidate
	nominatedLocal   string
	nominatedRemote  string
	nominatedChan    chan struct{}
	once             sync.Once
	mu               sync.Mutex
}

// NewICESession instantiates a new ICESession.
func NewICESession() *ICESession {
	return &ICESession{
		nominatedChan: make(chan struct{}),
	}
}

// AddLocalCandidate adds a candidate to the local candidate list.
func (s *ICESession) AddLocalCandidate(c ICECandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.localCandidates = append(s.localCandidates, c)
}

// AddRemoteCandidate adds a candidate to the remote candidate list.
func (s *ICESession) AddRemoteCandidate(c ICECandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteCandidates = append(s.remoteCandidates, c)
}

// GetBestLocalCandidate returns the local candidate with the highest priority.
func (s *ICESession) GetBestLocalCandidate() ICECandidate {
	s.mu.Lock()
	defer s.mu.Unlock()

	var best ICECandidate
	for _, c := range s.localCandidates {
		if c.Priority > best.Priority {
			best = c
		}
	}
	return best
}

// ConfirmCheck sets the nominated local-remote address pair.
func (s *ICESession) ConfirmCheck(localAddr, remoteAddr string) {
	s.mu.Lock()
	s.nominatedLocal = localAddr
	s.nominatedRemote = remoteAddr
	s.mu.Unlock()

	s.once.Do(func() {
		close(s.nominatedChan)
	})
}

// WaitForNomination blocks until a candidate pair is nominated or times out.
func (s *ICESession) WaitForNomination(timeout time.Duration) error {
	select {
	case <-s.nominatedChan:
		return nil
	case <-time.After(timeout):
		return errors.New("ICE nomination timed out")
	}
}

// GetNominatedPair returns the chosen local and remote connection addresses.
func (s *ICESession) GetNominatedPair() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nominatedLocal, s.nominatedRemote
}

// GatherCandidates collects local interface host candidates and queries STUN for reflexive ones.
func (s *ICESession) GatherCandidates(stunServer string, timeout time.Duration) error {
	// 1. Gather host candidates (local interface IPs)
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && !ipNet.IP.IsLoopback() {
				ipv4 := ipNet.IP.To4()
				if ipv4 != nil {
					s.AddLocalCandidate(ICECandidate{
						Addr:     net.JoinHostPort(ipv4.String(), "0"),
						Type:     CandidateHost,
						Priority: 1000,
					})
				}
			}
		}
	}

	// Always ensure at least one loopback host candidate for tests/local setups
	s.AddLocalCandidate(ICECandidate{
		Addr:     "127.0.0.1:0",
		Type:     CandidateHost,
		Priority: 100,
	})

	// 2. Query STUN server (non-blocking/graceful timeout handling)
	if stunServer != "" {
		go func() {
			conn, err := net.DialTimeout("udp", stunServer, timeout)
			if err != nil {
				return
			}
			defer conn.Close()

			req, txID, err := BuildSTUNBindingRequest()
			if err != nil {
				return
			}

			_ = conn.SetDeadline(time.Now().Add(timeout))
			_, err = conn.Write(req)
			if err != nil {
				return
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				return
			}

			ip, port, err := ParseSTUNBindingResponse(buf[:n], txID)
			if err == nil {
				s.AddLocalCandidate(ICECandidate{
					Addr:     net.JoinHostPort(ip.String(), strconv.Itoa(port)),
					Type:     CandidateSrflx,
					Priority: 500,
				})
			}
		}()
	}

	return nil
}
