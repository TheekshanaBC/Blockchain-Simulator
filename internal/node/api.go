package node

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// setupAPI registers all the HTTP endpoints for the node
func (n *Node) setupAPI(mux *http.ServeMux) {
	// Chain endpoints (api_chain.go)
	mux.HandleFunc("GET /status", n.handleStatus)
	mux.HandleFunc("GET /chain/height", n.handleChainHeight)
	mux.HandleFunc("GET /chain", n.handleChain)
	mux.HandleFunc("GET /chain/blocks/{height}", n.handleBlockByHeight)
	mux.HandleFunc("GET /chain/blocks/{height}/proof/{txIndex}", n.handleGetMerkleProof)
	mux.HandleFunc("POST /chain/blocks/{height}/verify-proof", n.handleVerifyMerkleProof)
	mux.HandleFunc("GET /balances", n.handleBalances)
	mux.HandleFunc("GET /balances/{address}", n.handleBalanceForAddress)
	mux.HandleFunc("GET /sequence/{address}", n.handleSequence)

	// Transaction & Mempool endpoints (api_tx.go)
	mux.HandleFunc("POST /tx/submit", n.handleSubmitTx)
	mux.HandleFunc("POST /tx/gossip", n.handleGossipTx)
	mux.HandleFunc("POST /block/gossip", n.handleGossipBlock)
	mux.HandleFunc("POST /mine", n.handleMine)
	mux.HandleFunc("POST /faucet", n.handleFaucet)
	mux.HandleFunc("GET /mempool", n.handleGetMempool)

	// Peer endpoints (api_peers.go)
	mux.HandleFunc("GET /peers", n.handleGetPeers)
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

