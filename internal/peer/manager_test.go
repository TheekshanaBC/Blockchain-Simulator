package peer

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

/*
TestPeerManagerAddAndRemove verifies that peers can be added and removed,
and that the self-address is correctly rejected.
*/
func TestPeerManagerAddAndRemove(t *testing.T) {
	pm := NewPeerManager("localhost:3001", []string{"localhost:3002"})

	if pm.AddPeer("localhost:3001") {
		t.Error("Should not be able to add self as peer")
	}

	if !pm.AddPeer("localhost:3003") {
		t.Error("Should be able to add a new peer")
	}

	if pm.AddPeer("localhost:3003") {
		t.Error("Should return false when adding duplicate peer")
	}

	peers := pm.GetPeers()
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}

	pm.RemovePeer("localhost:3002")
	peers = pm.GetPeers()
	if len(peers) != 1 || peers[0] != "localhost:3003" {
		t.Error("RemovePeer failed")
	}
}

/*
TestPeerManagerHealth verifies that MarkFailed and MarkSeen update the
peer's health status correctly.
*/
func TestPeerManagerHealth(t *testing.T) {
	pm := NewPeerManager("localhost:3001", []string{"localhost:3002"})

	// Fail 3 times to mark as unhealthy
	pm.MarkFailed("localhost:3002")
	pm.MarkFailed("localhost:3002")
	pm.MarkFailed("localhost:3002")

	peers := pm.GetPeers()
	if len(peers) != 0 {
		t.Error("Expected 0 healthy peers after 3 failures")
	}

	allPeers := pm.GetAllPeers()
	if len(allPeers) != 1 || allPeers[0].Healthy {
		t.Error("Expected 1 unhealthy peer in GetAllPeers")
	}

	// Mark seen should restore health
	pm.MarkSeen("localhost:3002")
	peers = pm.GetPeers()
	if len(peers) != 1 {
		t.Error("Expected 1 healthy peer after MarkSeen")
	}
}

/*
TestPeerManagerConcurrency verifies thread-safety. Should be run with -race.
*/
func TestPeerManagerConcurrency(t *testing.T) {
	pm := NewPeerManager("localhost:3001", []string{})
	var wg sync.WaitGroup

	// Concurrently add peers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := fmt.Sprintf("localhost:%d", 4000+id)
			pm.AddPeer(addr)
			pm.MarkSeen(addr)
			pm.GetPeers()
		}(i)
	}

	wg.Wait()

	if len(pm.GetPeers()) != 100 {
		t.Errorf("Expected 100 peers, got %d", len(pm.GetPeers()))
	}

	// Concurrently mark as failed
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := fmt.Sprintf("localhost:%d", 4000+id)
			for j := 0; j < 3; j++ {
				pm.MarkFailed(addr)
				time.Sleep(time.Millisecond) // Ensure some interleaving
			}
		}(i)
	}

	wg.Wait()

	if len(pm.GetPeers()) != 0 {
		t.Errorf("Expected 0 healthy peers, got %d", len(pm.GetPeers()))
	}
}
