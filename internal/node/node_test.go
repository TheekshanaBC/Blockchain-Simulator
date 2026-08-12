package node

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNode_HealthCheckPeers(t *testing.T) {
	cfg := Config{Port: 9999, DataDir: t.TempDir(), MaxTxPerBlock: 10}
	node, err := NewNode(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Create a mock failing peer server
	failedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failedServer.Close()

	// Add peer
	node.PeerManager.AddPeer(failedServer.URL)
	
	// Initial state: 0 failures, healthy
	peers := node.PeerManager.GetAllPeers()
	if len(peers) != 1 || peers[0].Failures != 0 {
		t.Fatalf("Expected 1 healthy peer with 0 failures")
	}

	// Trigger health check 3 times
	for i := 0; i < 3; i++ {
		node.healthCheckPeers()
	}

	// Wait for async health checks to complete
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		peers = node.PeerManager.GetAllPeers()
		if len(peers) == 1 && peers[0].Failures >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	peers = node.PeerManager.GetAllPeers()
	if len(peers) != 1 {
		t.Fatalf("Peer should not be removed yet, just marked unhealthy")
	}
	if peers[0].Failures < 3 {
		t.Fatalf("Expected at least 3 failures, got %d", peers[0].Failures)
	}
	if peers[0].Healthy {
		t.Fatalf("Expected peer to be marked unhealthy")
	}
}

func TestNode_AnnounceToPeer(t *testing.T) {
	cfg := Config{Port: 9998, DataDir: t.TempDir(), MaxTxPerBlock: 10}
	node, err := NewNode(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	receivedAnnounce := false
	// Create a mock peer server that accepts announcements
	mockPeer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/peers/announce" && r.Method == "POST" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["address"] == "http://localhost:9998" {
				receivedAnnounce = true
			}
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockPeer.Close()

	node.PeerManager.AddPeer(mockPeer.URL)
	
	// Call announceToPeer
	node.announceToPeer(mockPeer.URL)
	
	if !receivedAnnounce {
		t.Errorf("Mock peer did not receive the correct announcement payload")
	}
}
