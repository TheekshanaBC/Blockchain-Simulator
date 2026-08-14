package node

import (
	"encoding/json"
	"net/http"
	"strconv"
	"valence/internal/block"
	"valence/internal/chain"
	"valence/internal/ledger"
)

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

	work := chain.CumulativeWork(n.Chain.GetBlocks())

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"height": n.Chain.Height(),
		"hash":   hash,
		"work":   work.String(),
	})
}

// GET /chain
func (n *Node) handleChain(w http.ResponseWriter, r *http.Request) {
	blocks := n.Chain.GetBlocks()
	limit := 50 // Default limit

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	// Hard limit to prevent memory exhaustion, but large enough for the simulator
	if limit > 100000 {
		limit = 100000
	}

	if len(blocks) > limit {
		blocks = blocks[len(blocks)-limit:]
	}

	respondJSON(w, http.StatusOK, blocks)
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

// GET /balances/{address}
func (n *Node) handleBalanceForAddress(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if address == "" {
		respondError(w, http.StatusBadRequest, "address is required")
		return
	}

	balances := ledger.CalculateBalances(n.Chain.GetBlocks())
	balance := balances[address]
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"address": address,
		"balance": balance,
	})
}

// GET /history/{address}
func (n *Node) handleHistoryForAddress(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if address == "" {
		respondError(w, http.StatusBadRequest, "address is required")
		return
	}

	blocks := n.Chain.GetBlocks()
	var history []map[string]interface{}

	// Traverse blocks in reverse order (newest first)
	for i := len(blocks) - 1; i >= 0; i-- {
		b := blocks[i]
		for _, tx := range b.Transactions {
			if tx.Sender == address || tx.Recipient == address {
				history = append(history, map[string]interface{}{
					"tx_id":     tx.ID,
					"sender":    tx.Sender,
					"recipient": tx.Recipient,
					"amount":    tx.Amount,
					"timestamp": tx.Timestamp,
					"height":    b.Height,
				})
			}
		}
	}

	if history == nil {
		history = make([]map[string]interface{}, 0)
	}

	respondJSON(w, http.StatusOK, history)
}

// GET /sequence/{address}
func (n *Node) handleSequence(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if address == "" {
		respondError(w, http.StatusBadRequest, "address is required")
		return
	}

	sequences := ledger.CalculatePendingSequences(n.Chain.GetBlocks(), n.Mempool.GetAll())
	nextSeq := sequences[address] + 1

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"address":       address,
		"next_sequence": nextSeq,
	})
}

// GET /chain/blocks/{height}/proof/{txIndex}
func (n *Node) handleGetMerkleProof(w http.ResponseWriter, r *http.Request) {
	heightStr := r.PathValue("height")
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid height")
		return
	}

	txIndexStr := r.PathValue("txIndex")
	txIndex, err := strconv.Atoi(txIndexStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid txIndex")
		return
	}

	blocks := n.Chain.GetBlocks()
	if height < 0 || height >= len(blocks) {
		respondError(w, http.StatusNotFound, "block not found")
		return
	}

	b := blocks[height]
	if txIndex < 0 || txIndex >= len(b.Transactions) {
		respondError(w, http.StatusNotFound, "transaction not found in block")
		return
	}

	proof := block.BuildMerkleProof(b.Transactions, txIndex)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"proof":       proof,
		"root":        b.Header.MerkleRoot,
		"transaction": b.Transactions[txIndex],
	})
}

// POST /chain/blocks/{height}/verify-proof
func (n *Node) handleVerifyMerkleProof(w http.ResponseWriter, r *http.Request) {
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

	b := blocks[height]

	var req struct {
		Transaction block.Transaction `json:"transaction"`
		Proof       []block.ProofNode `json:"proof"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	valid := block.VerifyMerkleProof(req.Transaction, req.Proof, b.Header.MerkleRoot)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid": valid,
	})
}
