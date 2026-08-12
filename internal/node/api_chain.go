package node

import (
	"net/http"
	"strconv"
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
