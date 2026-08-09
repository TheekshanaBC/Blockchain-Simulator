package peer

import (
	"testing"
	"time"
)

func TestPeerManager_PruneUnhealthyPeers(t *testing.T) {
	pm := NewPeerManager("localhost:8080", []string{})
	
	pm.AddPeer("peer1")
	pm.AddPeer("peer2")
	pm.AddPeer("peer3")
	
	// Mark peer1 as failed 3 times (unhealthy), but last seen recently
	pm.MarkFailed("peer1")
	pm.MarkFailed("peer1")
	pm.MarkFailed("peer1")
	
	// Manipulate peer2 to be unhealthy AND last seen 1 hour ago
	pm.mu.Lock()
	pm.peers["peer2"].Healthy = false
	pm.peers["peer2"].LastSeen = time.Now().Add(-1 * time.Hour)
	pm.mu.Unlock()
	
	// peer3 remains healthy
	
	// Prune peers that have been unhealthy for more than 30 minutes
	pm.PruneUnhealthyPeers(30 * time.Minute)
	
	// Check results
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if _, exists := pm.peers["peer1"]; !exists {
		t.Error("Expected peer1 to remain (failed but recent)")
	}
	
	if _, exists := pm.peers["peer2"]; exists {
		t.Error("Expected peer2 to be pruned (failed and old)")
	}
	
	if _, exists := pm.peers["peer3"]; !exists {
		t.Error("Expected peer3 to remain (healthy)")
	}
}
