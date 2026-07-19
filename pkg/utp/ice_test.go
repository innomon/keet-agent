package utp

import (
	"testing"
	"time"
)

func TestICE_CandidatePriority(t *testing.T) {
	c1 := ICECandidate{
		Addr:     "192.168.1.100:12345",
		Type:     CandidateHost,
		Priority: 1000,
	}

	c2 := ICECandidate{
		Addr:     "203.0.113.50:54321",
		Type:     CandidateSrflx,
		Priority: 500,
	}

	session := NewICESession()
	session.AddLocalCandidate(c1)
	session.AddLocalCandidate(c2)

	best := session.GetBestLocalCandidate()
	if best.Addr != c1.Addr {
		t.Errorf("expected best candidate to be host %s, got %s", c1.Addr, best.Addr)
	}
}

func TestICE_ConnectivityChecks(t *testing.T) {
	session := NewICESession()

	// Add local and remote candidates
	session.AddLocalCandidate(ICECandidate{Addr: "127.0.0.1:10001", Type: CandidateHost, Priority: 100})
	session.AddRemoteCandidate(ICECandidate{Addr: "127.0.0.1:10002", Type: CandidateHost, Priority: 100})

	// Simulate connectivity check response
	go func() {
		time.Sleep(10 * time.Millisecond)
		session.ConfirmCheck("127.0.0.1:10001", "127.0.0.1:10002")
	}()

	err := session.WaitForNomination(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("connectivity checks failed to nominate a pair: %v", err)
	}

	local, remote := session.GetNominatedPair()
	if local != "127.0.0.1:10001" || remote != "127.0.0.1:10002" {
		t.Errorf("unexpected nominated pair: local=%s, remote=%s", local, remote)
	}
}

func TestICE_GatherCandidates(t *testing.T) {
	// Create mock STUN connection
	// We verify Candidate Gathering handles non-responsive STUN servers gracefully without crashing
	session := NewICESession()
	err := session.GatherCandidates("127.0.0.1:1", 50*time.Millisecond)
	// Expect no panic, and candidate list should contain at least host candidates
	if err != nil && err.Error() == "" {
		t.Errorf("unexpected error format: %v", err)
	}

	if len(session.localCandidates) == 0 {
		t.Error("expected at least host candidates to be gathered")
	}
}

func TestICE_GetNominatedPair(t *testing.T) {
	session := NewICESession()
	session.ConfirmCheck("127.0.0.1:1000", "127.0.0.1:2000")
	local, remote := session.GetNominatedPair()
	if local != "127.0.0.1:1000" || remote != "127.0.0.1:2000" {
		t.Errorf("expected nominated pair, got %s -> %s", local, remote)
	}
}

