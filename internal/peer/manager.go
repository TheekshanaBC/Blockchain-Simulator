package peer

import (
	"strings"
	"sync"
	"time"
)

// normalizeAddress strips http:// and https:// prefixes to ensure consistent peer tracking
func normalizeAddress(addr string) string {
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	return addr
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

	pm.peers[address] = &PeerInfo{
		Address:  address,
		LastSeen: time.Now(),
		Healthy:  true,
		Failures: 0,
	}
	return true
}

// RemovePeer permanently removes a peer from the manager.
func (pm *PeerManager) RemovePeer(address string) {
	address = normalizeAddress(address)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.peers, address)
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
