package node

import (
	"context"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"valence/internal/block"
)

func setupTestNode(t *testing.T) *Node {
	cfg := Config{
		Port:            8080,
		DataDir:         t.TempDir(),
		Difficulty:      1,
		RetargetWindow:  4,
		TargetBlockTime: 10,
		MinDifficulty:   1,
		MaxDifficulty:   6,
	}
	n, err := NewNode(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	return n
}

/*
TestAPIStatus verifies the /status endpoint.
It ensures that a newly started node returns HTTP 200 OK
and reports an initial blockchain height of 0.
*/
func TestAPIStatus(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	req, _ := http.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response["height"].(float64) != 0 {
		t.Errorf("Expected height 0, got %v", response["height"])
	}
}

/*
TestAPISubmitTx_InvalidSignature ensures the /tx/submit endpoint
rejects transactions that lack a valid cryptographic signature.
It should return HTTP 400 Bad Request.
*/
func TestAPISubmitTx_InvalidSignature(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Create a transaction but don't sign it properly
	tx := block.Transaction{
		Sender:    "Alice",
		Recipient: "Bob",
		Amount:    100,
	}
	tx.ComputeID()
	// No signature

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/submit", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

/*
TestAPISubmitTx_ValidSignature ensures the /tx/submit endpoint
accepts correctly constructed and signed transactions.
It verifies that the transaction is successfully added to the mempool
and returns HTTP 202 Accepted.
*/
func TestAPISubmitTx_ValidSignature(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Fund the wallet so validation passes
	fTx, _ := n.Chain.CreateFaucetTx(n.Wallet.Address(), 1000, nil)
	n.Chain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner") // Mined into a block

	tx := block.Transaction{
		Sender:    n.Wallet.Address(),
		Recipient: "Bob",
		Amount:    100,
		Sequence:  1, // Expected sequence is 1 for a new address
	}
	tx.ComputeID()
	tx.Sign(n.Wallet.PrivateKey)

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/submit", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v (body: %s)", status, http.StatusAccepted, rr.Body.String())
	}

	if !n.Mempool.Has(tx.ID) {
		t.Error("Transaction was not added to mempool")
	}
}

/*
TestAPIFaucet verifies the /faucet endpoint.
It ensures that a valid request creates a FAUCET transaction
and correctly places it into the node's mempool.
*/
func TestAPIFaucet(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	payload := map[string]interface{}{
		"address": "test_address",
		"amount":  100, // 100 VCN
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/faucet", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
	}

	if n.Mempool.Size() != 1 {
		t.Errorf("Faucet transaction not added to mempool")
	}
	txs := n.Mempool.GetAll()
	if txs[0].Sender != block.SystemAddressFaucet {
		t.Errorf("Incorrect sender, got %s want %s", txs[0].Sender, block.SystemAddressFaucet)
	}
}

/*
TestAPIGossipTx_AlreadySeen tests the deduplication logic in the gossip protocol.
If a transaction is gossiped to a node that already has it in its mempool,
the node should safely ignore it and return an 'already_seen' status.
*/
func TestAPIGossipTx_AlreadySeen(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Fund the wallet so validation passes
	fTx, _ := n.Chain.CreateFaucetTx(n.Wallet.Address(), 1000, nil)
	n.Chain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	tx := block.Transaction{
		Sender:    n.Wallet.Address(),
		Recipient: "Bob",
		Amount:    100,
		Sequence:  1,
	}
	tx.ComputeID()
	tx.Sign(n.Wallet.PrivateKey)

	// Add to mempool manually
	n.Mempool.Add(tx)

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/gossip", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response["status"] != "already_seen" {
		t.Errorf("Expected status already_seen, got %v", response["status"])
	}
}

/*
TestAPIPeersAnnounce verifies the /peers/announce endpoint.
It ensures that when a new peer announces itself to the node,
the node correctly updates its internal PeerManager with the new addresses.
*/
func TestAPIPeersAnnounce(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	payload := map[string]interface{}{
		"address": "node2:3002",
		"peers":   []string{"node3:3003"},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/peers/announce", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	peers := n.PeerManager.GetPeers()
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %v", len(peers))
	}
}

/*
TestAPIGossipBlock verifies the /block/gossip endpoint.
It ensures that a valid block received from a peer is successfully
validated and appended to the local blockchain, increasing the chain height.
*/
func TestAPIGossipBlock(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Create a valid block (e.g. genesis + 1)
	lastBlock := n.Chain.GetLastBlock()
	b := block.Block{
		Header: block.BlockHeader{
			PrevHash:   lastBlock.Hash,
			Difficulty: n.Chain.Difficulty,
			Timestamp:  lastBlock.Header.Timestamp + 1,
		},
		Height: 1,
		Transactions: []block.Transaction{
			{
				Sender:    block.SystemAddressCoinbase,
				Recipient: n.Wallet.Address(),
				Amount:    block.MiningReward,
			},
		},
	}
	b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
	b.Mine(context.Background(), 1)

	body, _ := json.Marshal(b)
	req, _ := http.NewRequest("POST", "/block/gossip", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v (body: %s)", status, http.StatusAccepted, rr.Body.String())
	}

	if n.Chain.Height() != 1 {
		t.Errorf("Expected chain height 1, got %v", n.Chain.Height())
	}
}

/*
TestAPISubmitTx_SystemAddressForge ensures the API prevents clients
from directly forging System Address transactions (like FAUCET or COINBASE)
via the /tx/submit endpoint, returning HTTP 403 Forbidden.
*/
func TestAPISubmitTx_SystemAddressForge(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: n.Wallet.Address(),
		Amount:    100,
	}
	tx.ComputeID()
	// No signature needed since tx.Verify() returns true for system addresses

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/submit", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code for forged system tx: got %v want %v", status, http.StatusForbidden)
	}
}

/*
TestAPIGossipTx_ValidFaucet ensures that valid FAUCET transactions are
allowed to propagate through the gossip network via /tx/gossip,
so that other nodes can receive and mine them.
*/
func TestAPIGossipTx_ValidFaucet(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: n.Wallet.Address(),
		Amount:    100,
	}
	tx.ComputeID()

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/gossip", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code for valid faucet tx in gossip: got %v want %v", status, http.StatusAccepted)
	}
}

/*
TestAPIFaucet_NegativeAmount ensures the /faucet endpoint
rejects requests containing negative amounts, preventing
malicious integer manipulation.
*/
func TestAPIFaucet_NegativeAmount(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	payload := map[string]interface{}{
		"address": "test_address",
		"amount":  -50,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/faucet", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for negative faucet amount: got %v want %v", status, http.StatusBadRequest)
	}
}
