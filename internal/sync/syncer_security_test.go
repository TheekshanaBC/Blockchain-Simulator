package sync

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"valence/internal/block"
	"valence/internal/chain"
	"valence/internal/peer"
)

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
