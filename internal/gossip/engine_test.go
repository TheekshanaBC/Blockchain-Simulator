package gossip

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
	"valence/internal/block"
	"valence/internal/peer"
)

/*
TestEngine_BroadcastTx verifies that the Gossip Engine correctly
sends a transaction to all healthy peers exactly once (due to deduplication).
*/
func TestEngine_BroadcastTx(t *testing.T) {
	var mu sync.Mutex
	receivedCount := 0

	// 1. Create a mock HTTP server to simulate a peer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tx/gossip" {
			mu.Lock()
			receivedCount++
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	// 2. Setup dependencies (PeerManager, SeenCache, Logger)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pm := peer.NewPeerManager("localhost:1000", []string{server.URL})
	cache := NewSeenCache(1 * time.Hour)
	engine := NewEngine(pm, cache, logger)

	// 3. Create a dummy transaction
	tx := block.Transaction{
		ID:        "tx_gossip_test_1",
		Sender:    "Alice",
		Recipient: "Bob",
		Amount:    100,
	}

	// 4. Trigger the broadcast
	engine.BroadcastTx(tx)

	// Wait briefly for asynchronous goroutines to finish
	time.Sleep(100 * time.Millisecond)

	// 5. Verify the peer received the broadcast
	mu.Lock()
	if receivedCount != 1 {
		t.Errorf("Expected 1 broadcast reception, got %d", receivedCount)
	}
	mu.Unlock()

	// 6. Attempt to broadcast the exact same transaction again
	engine.BroadcastTx(tx)
	time.Sleep(100 * time.Millisecond)

	// 7. Verify the cache prevented the second broadcast
	mu.Lock()
	if receivedCount != 1 {
		t.Errorf("Expected deduplication to prevent second broadcast, but got %d receptions", receivedCount)
	}
	mu.Unlock()
}

/*
TestEngine_PeerFailure verifies that if a peer is unreachable,
the Engine correctly invokes PeerManager.MarkFailed().
*/
func TestEngine_PeerFailure(t *testing.T) {
	// 1. Setup an Engine pointing to a dummy (unreachable) peer
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pm := peer.NewPeerManager("localhost:1000", []string{"localhost:9999"})
	cache := NewSeenCache(1 * time.Hour)
	engine := NewEngine(pm, cache, logger)

	// Ensure peer starts with 0 failures
	peerInfo := pm.GetAllPeers()[0]
	if peerInfo.Failures != 0 {
		t.Fatalf("Expected 0 initial failures, got %d", peerInfo.Failures)
	}

	// 2. Broadcast a transaction
	tx := block.Transaction{ID: "tx_fail_test"}
	engine.BroadcastTx(tx)

	// Wait briefly for asynchronous HTTP request to fail
	time.Sleep(100 * time.Millisecond)

	// 3. Verify the peer was marked as failed
	peerInfo = pm.GetAllPeers()[0]
	if peerInfo.Failures != 1 {
		t.Errorf("Expected peer to have 1 failure after unreachable request, got %d", peerInfo.Failures)
	}
}
