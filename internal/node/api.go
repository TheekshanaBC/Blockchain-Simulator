package node

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"valence/internal/block"
	"valence/internal/ledger"
)

const FaucetMaxBalance = 10_000 * 1_000_000_000 // 10,000 VCN

// setupAPI registers all the HTTP endpoints for the node
func (n *Node) setupAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /status", n.handleStatus)
	mux.HandleFunc("GET /chain/height", n.handleChainHeight)
	mux.HandleFunc("GET /chain", n.handleChain)
	mux.HandleFunc("GET /chain/blocks/{height}", n.handleBlockByHeight)
	mux.HandleFunc("GET /balances", n.handleBalances)
	mux.HandleFunc("GET /mempool", n.handleGetMempool)
	mux.HandleFunc("GET /peers", n.handleGetPeers)

	mux.HandleFunc("POST /tx/submit", n.handleSubmitTx)
	mux.HandleFunc("POST /tx/gossip", n.handleGossipTx)
	mux.HandleFunc("POST /block/gossip", n.handleGossipBlock)
	mux.HandleFunc("POST /mine", n.handleMine)
	mux.HandleFunc("POST /faucet", n.handleFaucet)
	mux.HandleFunc("POST /peers/announce", n.handlePeersAnnounce)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// GET /status
func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {

	lastBlock := n.Chain.GetLastBlock()
	hash := ""
	if lastBlock != nil {
		hash = lastBlock.Hash
	}

	status := map[string]interface{}{
		"height":       n.Chain.Height(),
		"head_hash":    hash,
		"peers":        len(n.PeerManager.GetPeers()),
		"mempool_size": n.Mempool.Size(),
		"mining":       false, // TODO: Update when mining worker is implemented
		"address":      n.Wallet.Address(),
	}
	respondJSON(w, http.StatusOK, status)
}

// GET /chain/height
func (n *Node) handleChainHeight(w http.ResponseWriter, r *http.Request) {

	lastBlock := n.Chain.GetLastBlock()
	hash := ""
	if lastBlock != nil {
		hash = lastBlock.Hash
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"height": n.Chain.Height(),
		"hash":   hash,
	})
}

// GET /chain
func (n *Node) handleChain(w http.ResponseWriter, r *http.Request) {

	respondJSON(w, http.StatusOK, n.Chain.GetBlocks())
}

// GET /chain/blocks/{height}
func (n *Node) handleBlockByHeight(w http.ResponseWriter, r *http.Request) {
	heightStr := r.PathValue("height")
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid height")
		return
	}

	blocks := n.Chain.GetBlocks()
	if height < 0 || height >= len(blocks) {
		respondError(w, http.StatusNotFound, "block not found")
		return
	}

	respondJSON(w, http.StatusOK, blocks[height])
}

// GET /balances
func (n *Node) handleBalances(w http.ResponseWriter, r *http.Request) {

	balances := ledger.CalculateBalances(n.Chain.GetBlocks())
	respondJSON(w, http.StatusOK, balances)
}

// GET /mempool
func (n *Node) handleGetMempool(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, n.Mempool.GetAll())
}

// GET /peers
func (n *Node) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, n.PeerManager.GetPeers())
}

// POST /tx/submit
func (n *Node) handleSubmitTx(w http.ResponseWriter, r *http.Request) {
	var tx block.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Task 2.5: Verify signature at API boundary
	if !tx.Verify() {
		respondError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	if added := n.Mempool.Add(tx); !added {
		respondError(w, http.StatusConflict, "transaction already exists")
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"tx_id":  tx.ID,
	})
}

// POST /tx/gossip
func (n *Node) handleGossipTx(w http.ResponseWriter, r *http.Request) {
	var tx block.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify signature at API boundary
	if !tx.Verify() {
		respondError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	if added := n.Mempool.Add(tx); !added {
		respondJSON(w, http.StatusOK, map[string]string{"status": "already_seen"})
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// POST /block/gossip
func (n *Node) handleGossipBlock(w http.ResponseWriter, r *http.Request) {
	var b block.Block
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify block and its transactions (AddBlock takes its own lock)
	err := n.Chain.AddBlock(b)
	if err != nil {
		respondError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// Block accepted, remove its transactions from mempool
	var txIDs []string
	for _, tx := range b.Transactions {
		txIDs = append(txIDs, tx.ID)
	}
	n.Mempool.Remove(txIDs)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// POST /mine
func (n *Node) handleMine(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusAccepted, map[string]string{"status": "mining_started"})
}

// POST /faucet
func (n *Node) handleFaucet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
		Amount  int64  `json:"amount"` // In VCN
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	electrons := req.Amount * 1e9

	balances := ledger.CalculateBalances(n.Chain.GetBlocks())

	currentBalance := balances[req.Address]
	if currentBalance+electrons > FaucetMaxBalance {
		respondError(w, http.StatusBadRequest, "faucet limit exceeded")
		return
	}

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: req.Address,
		Amount:    electrons,
		Sequence:  uint64(time.Now().UnixNano()),
		Timestamp: time.Now().UnixNano(),
	}
	tx.ComputeID()
	
	n.Mempool.Add(tx)

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /peers/announce
func (n *Node) handlePeersAnnounce(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string   `json:"address"`
		Peers   []string `json:"peers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	n.PeerManager.AddPeer(req.Address)
	n.PeerManager.MarkSeen(req.Address)

	for _, p := range req.Peers {
		n.PeerManager.AddPeer(p)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"peers": n.PeerManager.GetPeers(),
	})
}
