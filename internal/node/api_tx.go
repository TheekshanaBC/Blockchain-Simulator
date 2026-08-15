package node

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"valence/internal/block"
	"valence/internal/ledger"
)

// POST /tx/submit
func (n *Node) handleSubmitTx(w http.ResponseWriter, r *http.Request) {
	var tx block.Transaction
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
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

	if err := n.Mempool.ValidateAndAdd(tx, n.Chain.GetBlocks()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	n.Logger.Info("TX received and added to mempool", "tx_id", tx.ID, "mempool_size", n.Mempool.Size())
	n.Gossip.BroadcastTx(tx)

	respondJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"tx_id":  tx.ID,
	})
}

// POST /tx/gossip
func (n *Node) handleGossipTx(w http.ResponseWriter, r *http.Request) {
	var tx block.Transaction
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Always compute ID server-side
	tx.ComputeID()

	// Prevent gossiping coinbase transactions directly
	if tx.Sender == block.SystemAddressCoinbase {
		respondError(w, http.StatusForbidden, "cannot gossip coinbase transactions directly")
		return
	}

	if n.Mempool.Has(tx.ID) {
		n.Logger.Info("TX already seen, skipping", "tx_id", tx.ID)
		respondJSON(w, http.StatusOK, map[string]string{"status": "already_seen"})
		return
	}

	if err := n.Mempool.ValidateAndAdd(tx, n.Chain.GetBlocks()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	n.Logger.Info("TX received and added to mempool", "tx_id", tx.ID, "mempool_size", n.Mempool.Size())
	n.Gossip.BroadcastTx(tx)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// POST /block/gossip
func (n *Node) handleGossipBlock(w http.ResponseWriter, r *http.Request) {
	var b block.Block
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	myHeight := n.Chain.Height()
	myHeadHash := ""
	if last := n.Chain.GetLastBlock(); last != nil {
		myHeadHash = last.Hash
	}

	if b.Height == myHeight+1 && b.Header.PrevHash != myHeadHash {
		n.Logger.Warn("Fork detected, requesting push sync", "peer_block", b.Hash, "my_head", myHeadHash)
		n.triggerSync() // Debounced sync trigger
		respondJSON(w, http.StatusConflict, map[string]interface{}{
			"error": "sync_required",
			"my_height": myHeight,
		})
		return
	} else if b.Height > myHeight+1 {
		n.Logger.Info("Received block in the future, requesting push sync", "block_height", b.Height, "my_height", myHeight)
		n.triggerSync() // Debounced sync trigger
		respondJSON(w, http.StatusConflict, map[string]interface{}{
			"error": "sync_required",
			"my_height": myHeight,
		})
		return
	} else if b.Height <= myHeight {
		respondJSON(w, http.StatusOK, map[string]string{"status": "already_seen_or_stale"})
		return
	}

	// Verify block and its transactions (AddBlock takes its own lock)
	n.Logger.Info("Block received from gossip", "height", b.Height, "hash", b.Hash)
	err := n.Chain.AddBlock(b)
	if err != nil {
		n.Logger.Warn("Block rejected", "hash", b.Hash, "error", err)
		respondError(w, http.StatusNotAcceptable, err.Error())
		return
	}

	// Block accepted, remove its transactions from mempool
	var txIDs []string
	for _, tx := range b.Transactions {
		txIDs = append(txIDs, tx.ID)
	}
	n.Mempool.Remove(txIDs)
	n.SaveState()
	
	n.Logger.Info("Block validated and appended", "height", b.Height, "hash", b.Hash)
	n.Gossip.BroadcastBlock(b)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// POST /mine
func (n *Node) handleMine(w http.ResponseWriter, r *http.Request) {
	if !n.isMining.CompareAndSwap(false, true) {
		respondError(w, http.StatusConflict, "mining already in progress")
		return
	}
	defer n.isMining.Store(false)

	txs := n.Mempool.GetAll()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	newBlock, err := n.Chain.MineBlock(ctx, txs, n.Config.MinerAddress)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	var txIDs []string
	for _, tx := range newBlock.Transactions {
		txIDs = append(txIDs, tx.ID)
	}
	n.Mempool.Remove(txIDs)
	n.SaveState()
	
	n.Logger.Info("Block mined successfully", "height", newBlock.Height, "hash", newBlock.Hash, "tx_count", len(newBlock.Transactions))
	n.Gossip.BroadcastBlock(newBlock)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "mined",
		"block":  newBlock,
	})
}

// POST /faucet
func (n *Node) handleFaucet(w http.ResponseWriter, r *http.Request) {
	if n.FaucetWallet == nil {
		respondError(w, http.StatusNotImplemented, "This node is not configured as a faucet")
		return
	}

	var req struct {
		Address string `json:"address"`
		Amount    int64  `json:"amount"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount <= 0 {
		respondError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if req.Amount > ledger.MaxFaucetRequest { // Check raw electrons
		respondError(w, http.StatusBadRequest, "amount exceeds faucet limit")
		return
	}

	// Re-use the existing Faucet Logic from internal/chain/faucet.go
	electrons := req.Amount

	// Lock the faucet strictly to prevent sequence collision between concurrent requests
	n.faucetMu.Lock()
	defer n.faucetMu.Unlock()

	mempoolTxs := n.Mempool.GetAll()
	blocks := n.Chain.GetBlocks()
	sequences := ledger.CalculatePendingSequences(blocks, mempoolTxs)
	balances := ledger.CalculateAvailableBalances(blocks, mempoolTxs)
	
	faucetSeq := sequences[n.FaucetWallet.Address()] + 1
	faucetBal := balances[n.FaucetWallet.Address()]

	tx, err := n.Chain.CreateFaucetTx(req.Address, electrons, n.FaucetWallet, faucetSeq, faucetBal, mempoolTxs)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := n.Mempool.ValidateAndAdd(tx, n.Chain.GetBlocks()); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	n.Logger.Info("TX received and added to mempool", "tx_id", tx.ID, "mempool_size", n.Mempool.Size())
	n.Gossip.BroadcastTx(tx)

	respondJSON(w, http.StatusAccepted, map[string]string{
		"status": "ok",
		"tx_id":  tx.ID,
	})
}

// GET /mempool
func (n *Node) handleGetMempool(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, n.Mempool.GetAll())
}
