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
	return chain.NewChain(1, 5, 10, 1, 10)
}

func setupSyncerWithMockPeer(t *testing.T, localChain *chain.Chain, handler http.HandlerFunc) (*Syncer, *httptest.Server) {
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
	
	txs, err := syncer.SyncFromBestPeer()
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
			})
			return
		}
		t.Errorf("Unexpected request to %s", r.URL.Path)
	}

	syncer, server := setupSyncerWithMockPeer(t, localChain, handler)
	defer server.Close()

	txs, err := syncer.SyncFromBestPeer()
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
			Timestamp:  time.Now().Unix(),
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
			})
			return
		}
		if r.URL.Path == "/chain" {
			json.NewEncoder(w).Encode(tallerChain.GetBlocks())
			return
		}
		t.Errorf("Unexpected request to %s", r.URL.Path)
	}

	syncer, server := setupSyncerWithMockPeer(t, localChain, handler)
	defer server.Close()

	_, err := syncer.SyncFromBestPeer()
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

	syncer, server := setupSyncerWithMockPeer(t, localChain, handler)
	defer server.Close()

	_, err := syncer.SyncFromBestPeer()
	if err == nil {
		t.Fatal("Expected sync to fail with invalid candidate chain, but it succeeded")
	}

	// Local chain should still be at genesis
	if localChain.Height() != 0 {
		t.Errorf("Expected local chain to remain at height 0, got %d", localChain.Height())
	}
}

