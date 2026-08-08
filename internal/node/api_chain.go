package node

import (
	"net/http"
	"strconv"
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
