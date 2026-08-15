package peer

import (
	"strings"
	"sync"
	"time"
)

// normalizeAddress ensures consistent peer tracking by standardizing on a scheme.
// It defaults to http:// if no scheme is provided, and preserves https://.
func normalizeAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	isHttps := strings.HasPrefix(addr, "https://")
	
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	
	// Fix R5: Use HasPrefix to avoid corrupting hostnames that contain "localhost:" as a substring
	// (e.g. "mylocalhost:3001" should not become "my127.0.0.1:3001")
	if strings.HasPrefix(addr, "localhost:") {
		addr = "127.0.0.1:" + addr[len("localhost:"):]
	}
	
	if isHttps {
		return "https://" + addr
	}
	return "http://" + addr
}

type PeerInfo struct {
	Address  string
	LastSeen time.Time
	Healthy  bool
	Failures int // consecutive failures
}

type PeerManager struct {
	mu       sync.RWMutex
	peers    map[string]*PeerInfo // key = address string
	selfAddr string               // our own address (never add self as peer)
}

func NewPeerManager(selfAddr string, initialPeers []string) *PeerManager {
	pm := &PeerManager{
		peers:    make(map[string]*PeerInfo),
		selfAddr: normalizeAddress(selfAddr),
	}

	for _, p := range initialPeers {
		pm.AddPeer(p)
	}

	return pm
}

// AddPeer adds a new peer. Returns false if the peer is ourself or already exists.
func (pm *PeerManager) AddPeer(address string) bool {
	address = normalizeAddress(address)
	if address == pm.selfAddr || address == "" {
		return false
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[address]; exists {
		return false
	}

	// Enforce Max Peer Limit (e.g., 50 peers) to prevent network overload
	if len(pm.peers) >= 50 {
		return false
	}

	pm.peers[address] = &PeerInfo{
		Address:  address,
		LastSeen: time.Now(),
		Healthy:  true,
		Failures: 0,
	}
	return true
}

// GetPeers returns a list of healthy peer addresses.
func (pm *PeerManager) GetPeers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var healthyPeers []string
	for addr, info := range pm.peers {
		if info.Healthy {
			healthyPeers = append(healthyPeers, addr)
		}
	}
	return healthyPeers
}

// GetAllPeers returns a copy of all peer info, including unhealthy ones.
func (pm *PeerManager) GetAllPeers() []*PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	allPeers := make([]*PeerInfo, 0, len(pm.peers))
	for _, info := range pm.peers {
		// Return a copy so the caller can't mutate the manager's state
		copyInfo := *info
		allPeers = append(allPeers, &copyInfo)
	}
	return allPeers
}

// MarkSeen updates the LastSeen timestamp and resets failures to 0.
func (pm *PeerManager) MarkSeen(address string) {
	address = normalizeAddress(address)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if info, exists := pm.peers[address]; exists {
		info.LastSeen = time.Now()
		info.Failures = 0
		info.Healthy = true
	}
}

// MarkFailed increments the failure count and marks the peer as unhealthy if it exceeds a threshold.
func (pm *PeerManager) MarkFailed(address string) {
	address = normalizeAddress(address)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if info, exists := pm.peers[address]; exists {
		info.Failures++
		if info.Failures >= 3 {
			info.Healthy = false
		}
	}
}

// PruneUnhealthyPeers permanently removes peers that have been unhealthy
// and not seen for longer than the provided maximum age.
func (pm *PeerManager) PruneUnhealthyPeers(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for addr, info := range pm.peers {
		if !info.Healthy && info.LastSeen.Before(cutoff) {
			delete(pm.peers, addr)
		}
	}
}

