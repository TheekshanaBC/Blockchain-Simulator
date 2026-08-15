package sync

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"valence/internal/block"
	"valence/internal/chain"
	"valence/internal/peer"
)

func setupTestChain() *chain.Chain {
	return chain.NewChain(1, 5, 10, 1, 10, 10)
}

func setupSyncerWithMockPeer(localChain *chain.Chain, handler http.HandlerFunc) (*Syncer, *httptest.Server) {
	server := httptest.NewServer(handler)
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Pass the bare host:port without http://
	pm := peer.NewPeerManager("localhost:1000", []string{server.URL})
	
	syncer := NewSyncer(localChain, pm, logger)
	
	return syncer, server
}

/*
TestSyncFromBestPeer_NoPeers verifies that syncing gracefully returns
when the node has no connected peers.
*/
func TestSyncFromBestPeer_NoPeers(t *testing.T) {
	localChain := setupTestChain()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pm := peer.NewPeerManager("localhost:1000", []string{}) // empty peers
	
	syncer := NewSyncer(localChain, pm, logger)
	
	_, txs, err := syncer.SyncFromBestPeer()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if txs != nil {
		t.Errorf("Expected nil transactions, got %v", txs)
	}
}

/*
TestSyncFromBestPeer_AlreadyAtTip verifies that the syncer correctly skips
downloading the chain if the best peer's height is equal to or less than ours.
*/
func TestSyncFromBestPeer_AlreadyAtTip(t *testing.T) {
	localChain := setupTestChain()
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"height": 0,
				"hash":   localChain.GetLastBlock().Hash,
				"work":   "0",
			})
			return
		}
		t.Errorf("Unexpected request to %s", r.URL.Path)
	}

	syncer, server := setupSyncerWithMockPeer(localChain, handler)
	defer server.Close()

	_, txs, err := syncer.SyncFromBestPeer()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if txs != nil {
		t.Errorf("Expected nil transactions, got %v", txs)
	}
}

/*
TestSyncFromBestPeer_SyncsTaller verifies that if a peer has a valid, taller chain,
the syncer will download it completely and successfully swap it out.
*/
func TestSyncFromBestPeer_SyncsTaller(t *testing.T) {
	// Build a valid taller chain
	tallerChain := setupTestChain()
	// Add a block to make it height 1
	b := block.Block{
		Height: 1,
		Header: block.BlockHeader{
			PrevHash:   tallerChain.GetLastBlock().Hash,
			Timestamp:  time.Now().UnixNano(),
			Difficulty: 1,
		},
		Transactions: []block.Transaction{
			{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
		},
	}
	b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
	b.Mine(context.Background(), 1)
	tallerChain.AddBlock(b)

	// Our local chain is just at genesis (height 0)
	localChain := setupTestChain()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"height": tallerChain.Height(),
				"hash":   tallerChain.GetLastBlock().Hash,
				"work":   chain.CumulativeWork(tallerChain.GetBlocks()).String(),
			})
			return
		}
		if r.URL.Path == "/chain" {
			json.NewEncoder(w).Encode(tallerChain.GetBlocks())
			return
		}
		t.Errorf("Unexpected request to %s", r.URL.Path)
	}

	syncer, server := setupSyncerWithMockPeer(localChain, handler)
	defer server.Close()

	_, _, err := syncer.SyncFromBestPeer()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if localChain.Height() != 1 {
		t.Errorf("Expected local chain to sync to height 1, got %d", localChain.Height())
	}
}

/*
TestSyncFromBestPeer_InvalidChain verifies that if the peer returns a broken/invalid chain,
it is rejected and an error is returned.
*/
func TestSyncFromBestPeer_InvalidChain(t *testing.T) {
	localChain := setupTestChain()
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"height": 5,
				"hash":   "fake_hash",
				"work":   "99999999",
			})
			return
		}
		if r.URL.Path == "/chain" {
			// Return a garbage chain that will fail validation
			garbageChain := []*block.Block{
				{Height: 99}, // completely invalid
			}
			json.NewEncoder(w).Encode(garbageChain)
			return
		}
	}

	syncer, server := setupSyncerWithMockPeer(localChain, handler)
	defer server.Close()

	_, _, err := syncer.SyncFromBestPeer()
	if err == nil {
		t.Fatal("Expected sync to fail with invalid candidate chain, but it succeeded")
	}

	// Local chain should still be at genesis
	if localChain.Height() != 0 {
		t.Errorf("Expected local chain to remain at height 0, got %d", localChain.Height())
	}
}

func TestSyncFromBestPeer_SyncsHeaviest(t *testing.T) {
	// Local chain with diff 1, 6 blocks (longer but lighter)
	localChain := chain.NewChain(1, 2, 10, 1, 10, 10)
	baseTime := localChain.GetLastBlock().Header.Timestamp
	for i := 1; i <= 6; i++ {
		lb := block.Block{
			Height: i,
			Header: block.BlockHeader{
				PrevHash:   localChain.GetLastBlock().Hash,
				Timestamp:  baseTime + int64(i)*10*1_000_000_000,
				Difficulty: 1,
			},
			Transactions: []block.Transaction{
				{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
			},
		}
		lb.Header.MerkleRoot = block.CalculateMerkleRoot(lb.Transactions)
		lb.Mine(context.Background(), 1)
		err := localChain.AddBlock(lb)
		if err != nil {
			t.Fatalf("localChain AddBlock failed at height %d: %v", i, err)
		}
	}

	// Heavier chain with diff 3, 5 blocks
	heavierChain := chain.NewChain(1, 2, 10, 1, 10, 10)
	for i := 1; i <= 5; i++ {
		expectedDiff := 1
		if i >= 3 {
			expectedDiff = 2
		}
		if i >= 5 {
			expectedDiff = 3
		}

		b := block.Block{
			Height: i,
			Header: block.BlockHeader{
				PrevHash:   heavierChain.GetLastBlock().Hash,
				Timestamp:  baseTime + int64(i)*1_000_000_000,
				Difficulty: expectedDiff,
			},
			Transactions: []block.Transaction{
				{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
			},
		}
		b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
		b.Mine(context.Background(), expectedDiff)
		err := heavierChain.AddBlock(b)
		if err != nil {
			t.Fatalf("heavierChain AddBlock failed at height %d: %v", i, err)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"height": heavierChain.Height(),
				"hash":   heavierChain.GetLastBlock().Hash,
				"work":   chain.CumulativeWork(heavierChain.GetBlocks()).String(),
			})
			return
		}
		if r.URL.Path == "/chain" {
			json.NewEncoder(w).Encode(heavierChain.GetBlocks())
			return
		}
	}

	syncer, server := setupSyncerWithMockPeer(localChain, handler)
	defer server.Close()

	_, _, err := syncer.SyncFromBestPeer()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if localChain.Height() != 5 {
		t.Errorf("Expected local chain to sync to heavier chain (height 5), got %d", localChain.Height())
	}
}

/*
TestGetPeerHeight_OversizedWorkString is a regression test for the
big.Int.SetString DoS vulnerability in getPeerHeight.

A malicious peer returning a multi-megabyte "work" string in the
/chain/height response would cause O(n²) CPU usage in SetString.
The fix adds http.MaxBytesReader(4096) before json.Decode, so an
oversized response is rejected with an error rather than processed.
*/
func TestGetPeerHeight_OversizedWorkString(t *testing.T) {
	localChain := setupTestChain()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			// Simulate an attacker returning a 10 MB "work" field.
			// Before the fix this would reach big.Int.SetString and peg the CPU.
			oversizedWork := make([]byte, 10*1024*1024) // 10 MB of '1' digits
			for i := range oversizedWork {
				oversizedWork[i] = '1'
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"height":1,"hash":"abc","work":"`))
			w.Write(oversizedWork)
			w.Write([]byte(`"}`))
			return
		}
	}

	syncer, server := setupSyncerWithMockPeer(localChain, handler)
	defer server.Close()

	// getPeerHeight is unexported — exercise it through SyncFromBestPeer,
	// which calls it internally and handles the error gracefully.
	_, _, err := syncer.SyncFromBestPeer()
	// We expect either an error (body cap triggered) or no sync (work rejected).
	// What must NOT happen is the call hanging for tens of seconds on SetString.
	// The test itself acts as the timeout guard — if it hangs, the fix is broken.
	if err != nil {
		t.Logf("SyncFromBestPeer correctly returned error for oversized work: %v", err)
	}
	// Either way, the local chain must be untouched.
	if localChain.Height() != 0 {
		t.Errorf("Expected local chain to remain at genesis (height 0), got %d", localChain.Height())
	}
}

/*
TestSyncFromPeer_OversizedChainBody is a regression test for the memory-exhaustion
DoS in SyncFromPeer. Without the fix, a malicious peer serving a multi-gigabyte
response for GET /chain could OOM the syncing node before any block validation
runs. The fix wraps the body in http.MaxBytesReader(10MB) matching the server-side
cap in handlePushSync.
*/
func TestSyncFromPeer_OversizedChainBody(t *testing.T) {
	localChain := setupTestChain()

	// Build a minimal valid block so the peer's reported work exceeds ours
	// and SyncFromPeer actually proceeds to fetch /chain.
	peerChain := setupTestChain()
	b := block.Block{
		Height: 1,
		Header: block.BlockHeader{
			PrevHash:   peerChain.GetLastBlock().Hash,
			Timestamp:  peerChain.GetLastBlock().Header.Timestamp + 10*1_000_000_000,
			Difficulty: 1,
		},
		Transactions: []block.Transaction{
			{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
		},
	}
	b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
	b.Mine(context.Background(), 1)
	peerChain.AddBlock(b)

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"height": peerChain.Height(),
				"hash":   peerChain.GetLastBlock().Hash,
				"work":   chain.CumulativeWork(peerChain.GetBlocks()).String(),
			})
			return
		}
		if r.URL.Path == "/chain" {
			// Serve a body that far exceeds the 10 MB cap.
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[")) // open a JSON array...
			garbage := make([]byte, 12*1024*1024) // 12 MB of garbage
			for i := range garbage {
				garbage[i] = 'x'
			}
			w.Write(garbage) // ...then stream garbage
			return
		}
	}

	syncer, server := setupSyncerWithMockPeer(localChain, handler)
	defer server.Close()

	_, _, err := syncer.SyncFromBestPeer()
	// Must fail (body cap or JSON decode error) — not succeed and not hang.
	if err == nil {
		t.Error("Expected SyncFromPeer to fail on oversized /chain body, but it succeeded")
	}
	// Local chain must be untouched.
	if localChain.Height() != 0 {
		t.Errorf("Expected local chain to remain at genesis after oversized body, got height %d", localChain.Height())
	}
}

/*
TestSyncMempoolFromPeer_OversizedBody is a regression test for the memory-exhaustion
DoS in SyncMempoolFromPeer. A malicious peer could serve a gigabyte response for
GET /mempool and OOM the node. The fix wraps the body in http.MaxBytesReader(5MB).
*/
func TestSyncMempoolFromPeer_OversizedBody(t *testing.T) {
	localChain := setupTestChain()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mempool" {
			// Serve a body that far exceeds the 5 MB cap.
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[")) // open a JSON array...
			garbage := make([]byte, 6*1024*1024) // 6 MB of garbage
			for i := range garbage {
				garbage[i] = 'x'
			}
			w.Write(garbage) // ...then stream garbage
			return
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	pm := peer.NewPeerManager("localhost:1000", []string{})
	syncer := NewSyncer(localChain, pm, logger)

	// Call SyncMempoolFromPeer directly — it's exported.
	_, err := syncer.SyncMempoolFromPeer(server.URL)
	if err == nil {
		t.Error("Expected SyncMempoolFromPeer to fail on oversized /mempool body, but it succeeded")
	}
}
