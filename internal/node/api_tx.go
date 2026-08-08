package node

import (
	"encoding/json"
	"net/http"
	"valence/internal/block"
	"valence/internal/ledger"
)

// POST /tx/submit
func (n *Node) handleSubmitTx(w http.ResponseWriter, r *http.Request) {
	var tx block.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Always compute ID server-side to prevent client spoofing/empty ID collisions
	tx.ComputeID()

	// Prevent forging system transactions via client API
	if block.IsSystemAddress(tx.Sender) {
		respondError(w, http.StatusForbidden, "cannot submit transactions from system addresses")
		return
	}

	if n.Mempool.Has(tx.ID) {
		respondError(w, http.StatusConflict, "transaction already exists")
		return
	}

	if err := ledger.ValidateTransactions([]block.Transaction{tx}, n.Chain.GetBlocks(), n.Mempool.GetAll()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	n.Mempool.Add(tx)
	n.Gossip.BroadcastTx(tx)

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

	// Always compute ID server-side
	tx.ComputeID()

	// Prevent gossiping raw system transactions
	if block.IsSystemAddress(tx.Sender) {
		respondError(w, http.StatusForbidden, "cannot gossip system transactions directly")
		return
	}

	if n.Mempool.Has(tx.ID) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "already_seen"})
		return
	}

	if err := ledger.ValidateTransactions([]block.Transaction{tx}, n.Chain.GetBlocks(), n.Mempool.GetAll()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	n.Mempool.Add(tx)
	n.Gossip.BroadcastTx(tx)

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
	n.Gossip.BroadcastBlock(b)

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

	if req.Amount <= 0 {
		respondError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if req.Amount > 1000 { // Check raw VCN before converting to prevent overflow
		respondError(w, http.StatusBadRequest, "amount exceeds faucet limit of 1000 VCN")
		return
	}

	electrons := req.Amount * block.ElectronsPerVCN

	// Re-use the existing Faucet Logic from internal/chain/faucet.go
	tx, err := n.Chain.RequestFaucetFunds(req.Address, electrons)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Move the newly created Faucet transaction directly to the Mempool
	n.Mempool.Add(tx)

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /mempool
func (n *Node) handleGetMempool(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, n.Mempool.GetAll())
}
