package dht

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config represents DHT configuration parameters.
type Config struct {
	BootstrapNodes   []string
	LocalID          [32]byte
	Port             int
	Transport        Transport
	AnnounceInterval time.Duration
}

// SwarmRepository is a decoupled interface for querying persisted active swarm topics.
type SwarmRepository interface {
	GetActiveSwarms(ctx context.Context) ([]string, error)
}

// DHTNode represents an active instance of a Kademlia node.
type DHTNode struct {
	localID          [32]byte
	transport        Transport
	dispatcher       *Dispatcher
	routing          *RoutingTable
	localRegistry    *SwarmRegistry
	bootstrapNodes   []string
	announces        map[[32]byte]context.CancelFunc
	announceInterval time.Duration
	mu               sync.Mutex
}

// NewDHTNode instantiates and binds a new DHT Node with the given configuration.
func NewDHTNode(cfg *Config) (*DHTNode, error) {
	var bootstrapNodes []string

	// Check env override
	if envNodes := os.Getenv("DHT_BOOTSTRAP_NODES"); envNodes != "" {
		parts := strings.Split(envNodes, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				bootstrapNodes = append(bootstrapNodes, part)
			}
		}
	} else if cfg != nil && len(cfg.BootstrapNodes) > 0 {
		bootstrapNodes = cfg.BootstrapNodes
	} else {
		// Public Holepunch defaults
		bootstrapNodes = []string{
			"dht1.holepunch.to:49737",
			"dht2.holepunch.to:49737",
			"dht3.holepunch.to:49737",
		}
	}

	var localID [32]byte
	if cfg != nil && cfg.LocalID != [32]byte{} {
		localID = cfg.LocalID
	} else {
		localID = generateLocalID()
	}

	var transport Transport
	if cfg != nil && cfg.Transport != nil {
		transport = cfg.Transport
	} else {
		port := 0
		if cfg != nil {
			port = cfg.Port
		}
		addr := fmt.Sprintf(":%d", port)
		var err error
		transport, err = NewUDPTransport(addr)
		if err != nil {
			return nil, err
		}
	}

	node := &DHTNode{
		localID:          localID,
		transport:        transport,
		bootstrapNodes:   bootstrapNodes,
		routing:          NewRoutingTable(localID),
		localRegistry:    NewSwarmRegistry(),
		announces:        make(map[[32]byte]context.CancelFunc),
		announceInterval: 10 * time.Minute,
	}

	if cfg != nil && cfg.AnnounceInterval > 0 {
		node.announceInterval = cfg.AnnounceInterval
	}

	node.dispatcher = NewDispatcher(transport, node.handleRequest)

	return node, nil
}

// GetBootstrapNodes returns the configured bootstrap node addresses.
func (n *DHTNode) GetBootstrapNodes() []string {
	return n.bootstrapNodes
}

// Start begins processing inbound packets, bootstraps the node, and re-hydrates active DB swarms.
func (n *DHTNode) Start(ctx context.Context, repo SwarmRepository) error {
	n.dispatcher.Start()

	if len(n.bootstrapNodes) == 0 {
		n.rehydrateSwarms(ctx, repo)
		return nil
	}

	var wg sync.WaitGroup
	for _, addrStr := range n.bootstrapNodes {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			resolved, err := n.resolveAddr(addr)
			if err != nil {
				return
			}

			txID := generateTxID()
			req := &Message{
				TxID:     txID,
				Type:     MsgPing,
				SenderID: n.localID,
			}

			resp, err := n.dispatcher.SendRequest(ctx, resolved, req)
			if err != nil {
				return
			}

			n.routing.AddContact(Contact{ID: resp.SenderID, Addr: addr})
		}(addrStr)
	}
	wg.Wait()

	_ = n.iterativeFindNode(ctx, n.localID)

	n.rehydrateSwarms(ctx, repo)

	return nil
}

// SetP2PPort configures the TCP listening port of the PeerManager for this DHT Node.
func (n *DHTNode) SetP2PPort(port uint16) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.localRegistry != nil {
		n.localRegistry.P2PPort = port
	}
}

func (n *DHTNode) rehydrateSwarms(ctx context.Context, repo SwarmRepository) {
	if repo == nil {
		return
	}

	activeKeys, err := repo.GetActiveSwarms(ctx)
	if err != nil {
		return
	}

	for _, keyHex := range activeKeys {
		keyBytes, err := hex.DecodeString(keyHex)
		if err != nil || len(keyBytes) != 32 {
			continue
		}
		var topicKey [32]byte
		copy(topicKey[:], keyBytes)

		go func(tk [32]byte) {
			// Brief delay to let bootstrapping stabilize
			time.Sleep(50 * time.Millisecond)

			_ = n.Announce(ctx, tk, 0)

			peers, err := n.Lookup(ctx, tk)
			if err == nil && n.localRegistry != nil {
				for _, p := range peers {
					n.localRegistry.RegisterPeer(tk, p)
				}
			}
		}(topicKey)
	}
}

// Stop terminates background processes and releases the transport.
func (n *DHTNode) Stop() {
	n.mu.Lock()
	for _, cancel := range n.announces {
		cancel()
	}
	n.announces = make(map[[32]byte]context.CancelFunc)
	n.mu.Unlock()

	n.dispatcher.Stop()
}

func (n *DHTNode) handleRequest(ctx context.Context, req *Message, src net.Addr) (*Message, error) {
	n.routing.AddContact(Contact{ID: req.SenderID, Addr: src.String()})

	switch req.Type {
	case MsgPing:
		return &Message{
			Type:     MsgPong,
			SenderID: n.localID,
		}, nil

	case MsgFindNode:
		closest := n.routing.FindClosestPeers(req.Target, 20)
		return &Message{
			Type:     MsgFindNodeResp,
			SenderID: n.localID,
			Contacts: closest,
		}, nil

	case MsgAnnounce:
		if req.Port == 0 {
			// Interpret port = 0 as LEAVE/Unannounce
			host, _, err := net.SplitHostPort(src.String())
			if err != nil {
				host = src.String()
			}
			peers := n.localRegistry.GetPeers(req.Topic)
			for _, p := range peers {
				pHost, _, err := net.SplitHostPort(p)
				if err != nil {
					pHost = p
				}
				if pHost == host {
					n.localRegistry.UnregisterPeer(req.Topic, p)
				}
			}
		} else {
			var peerAddr string
			if udpAddr, ok := src.(*net.UDPAddr); ok {
				peerAddr = net.JoinHostPort(udpAddr.IP.String(), strconv.Itoa(int(req.Port)))
			} else {
				host, _, err := net.SplitHostPort(src.String())
				if err == nil {
					peerAddr = net.JoinHostPort(host, strconv.Itoa(int(req.Port)))
				} else {
					peerAddr = src.String()
				}
			}
			n.localRegistry.RegisterPeer(req.Topic, peerAddr)
		}
		return &Message{
			Type:     MsgPong,
			SenderID: n.localID,
		}, nil

	case MsgLookup:
		peers := n.localRegistry.GetPeers(req.Topic)
		return &Message{
			Type:     MsgLookupResp,
			SenderID: n.localID,
			Peers:    peers,
		}, nil

	default:
		return nil, fmt.Errorf("unhandled message type: %d", req.Type)
	}
}

func (n *DHTNode) iterativeFindNode(ctx context.Context, target [32]byte) []Contact {
	candidates := n.routing.FindClosestPeers(target, 20)

	queried := make(map[[32]byte]bool)
	var closestList []Contact
	closestMap := make(map[[32]byte]Contact)

	for _, c := range candidates {
		closestList = append(closestList, c)
		closestMap[c.ID] = c
	}

	for {
		sort.Slice(closestList, func(i, j int) bool {
			d1 := xorDistance(closestList[i].ID, target)
			d2 := xorDistance(closestList[j].ID, target)
			return compareDistance(d1, d2) < 0
		})

		var toQuery []Contact
		for _, c := range closestList {
			if !queried[c.ID] {
				toQuery = append(toQuery, c)
				if len(toQuery) >= 3 {
					break
				}
			}
		}

		if len(toQuery) == 0 {
			break
		}

		type queryResult struct {
			contact  Contact
			contacts []Contact
			err      error
		}
		resChan := make(chan queryResult, len(toQuery))

		var wg sync.WaitGroup
		for _, c := range toQuery {
			queried[c.ID] = true
			wg.Add(1)
			go func(contact Contact) {
				defer wg.Done()
				addr, err := n.resolveAddr(contact.Addr)
				if err != nil {
					resChan <- queryResult{contact: contact, err: err}
					return
				}

				txID := generateTxID()
				req := &Message{
					TxID:     txID,
					Type:     MsgFindNode,
					SenderID: n.localID,
					Target:   target,
				}

				resp, err := n.dispatcher.SendRequest(ctx, addr, req)
				if err != nil {
					resChan <- queryResult{contact: contact, err: err}
					return
				}

				resChan <- queryResult{contact: contact, contacts: resp.Contacts, err: nil}
			}(c)
		}
		wg.Wait()
		close(resChan)

		anyCloser := false
		for res := range resChan {
			if res.err != nil {
				for idx, val := range closestList {
					if val.ID == res.contact.ID {
						closestList = append(closestList[:idx], closestList[idx+1:]...)
						break
					}
				}
				continue
			}

			n.routing.AddContact(res.contact)

			for _, nc := range res.contacts {
				if nc.ID == n.localID {
					continue
				}
				if _, ok := closestMap[nc.ID]; !ok {
					closestMap[nc.ID] = nc
					closestList = append(closestList, nc)
					anyCloser = true
				}
			}
		}
		_ = anyCloser
	}

	if len(closestList) > 20 {
		return closestList[:20]
	}
	return closestList
}

// Announce registers local peer port on a given topic key.
func (n *DHTNode) Announce(ctx context.Context, topic [32]byte, port uint16) error {
	if port == 0 {
		n.mu.Lock()
		if n.localRegistry != nil {
			port = n.localRegistry.P2PPort
		}
		n.mu.Unlock()
	}

	var localAddr string
	if _, ok := n.transport.(*UDPTransport); ok {
		localAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port)))
	} else {
		localAddr = n.transport.Addr().String()
	}
	n.localRegistry.RegisterPeer(topic, localAddr)

	closest := n.iterativeFindNode(ctx, topic)
	if len(closest) == 0 {
		n.startReannounce(topic, port)
		return nil
	}

	var wg sync.WaitGroup
	for _, c := range closest {
		wg.Add(1)
		go func(contact Contact) {
			defer wg.Done()
			addr, err := n.resolveAddr(contact.Addr)
			if err != nil {
				return
			}

			txID := generateTxID()
			req := &Message{
				TxID:     txID,
				Type:     MsgAnnounce,
				SenderID: n.localID,
				Topic:    topic,
				Port:     port,
			}

			_, _ = n.dispatcher.SendRequest(ctx, addr, req)
		}(c)
	}
	wg.Wait()

	n.startReannounce(topic, port)

	return nil
}

func (n *DHTNode) startReannounce(topic [32]byte, port uint16) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.announces[topic]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.announces[topic] = cancel

	interval := n.announceInterval
	if interval == 0 {
		interval = 10 * time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 15*time.Second)
				_ = n.Announce(ctxTimeout, topic, port)
				cancelTimeout()
			}
		}
	}()
}

// Lookup searches the network for peers announced on a given topic key.
func (n *DHTNode) Lookup(ctx context.Context, topic [32]byte) ([]string, error) {
	closest := n.iterativeFindNode(ctx, topic)
	if len(closest) == 0 {
		return n.localRegistry.GetPeers(topic), nil
	}

	var mu sync.Mutex
	peerSet := make(map[string]bool)

	var wg sync.WaitGroup
	for _, c := range closest {
		wg.Add(1)
		go func(contact Contact) {
			defer wg.Done()
			addr, err := n.resolveAddr(contact.Addr)
			if err != nil {
				return
			}

			txID := generateTxID()
			req := &Message{
				TxID:     txID,
				Type:     MsgLookup,
				SenderID: n.localID,
				Topic:    topic,
			}

			resp, err := n.dispatcher.SendRequest(ctx, addr, req)
			if err != nil {
				return
			}

			mu.Lock()
			for _, peer := range resp.Peers {
				peerSet[peer] = true
			}
			mu.Unlock()
		}(c)
	}
	wg.Wait()

	localPeers := n.localRegistry.GetPeers(topic)
	for _, p := range localPeers {
		peerSet[p] = true
	}

	peers := make([]string, 0, len(peerSet))
	for p := range peerSet {
		peers = append(peers, p)
	}

	return peers, nil
}

// Leave removes the local peer registration from a given topic and stops periodic re-announce.
func (n *DHTNode) Leave(ctx context.Context, topic [32]byte) error {
	n.mu.Lock()
	cancel, exists := n.announces[topic]
	if exists {
		cancel()
		delete(n.announces, topic)
	}
	n.mu.Unlock()

	// Notify closest peers that we are leaving
	closest := n.iterativeFindNode(ctx, topic)
	if len(closest) > 0 {
		var wg sync.WaitGroup
		for _, c := range closest {
			wg.Add(1)
			go func(contact Contact) {
				defer wg.Done()
				addr, err := n.resolveAddr(contact.Addr)
				if err != nil {
					return
				}

				txID := generateTxID()
				req := &Message{
					TxID:     txID,
					Type:     MsgAnnounce,
					SenderID: n.localID,
					Topic:    topic,
					Port:     0, // Port 0 indicates Leave/Unannounce
				}

				_, _ = n.dispatcher.SendRequest(ctx, addr, req)
			}(c)
		}
		wg.Wait()
	}

	n.localRegistry.ClearSwarm(topic)
	return nil
}

func generateLocalID() [32]byte {
	var id [32]byte
	_, _ = rand.Read(id[:])
	return id
}

func generateTxID() [4]byte {
	var id [4]byte
	_, _ = rand.Read(id[:])
	return id
}

func (n *DHTNode) resolveAddr(addr string) (net.Addr, error) {
	if _, ok := n.transport.(*InProcessTransport); ok {
		return inProcessAddr{addr: addr}, nil
	}
	return net.ResolveUDPAddr("udp", addr)
}
