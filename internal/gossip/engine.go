package gossip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"valence/internal/block"
	"valence/internal/peer"
)

// Engine is responsible for broadcasting blocks and transactions to the network.
type Engine struct {
	peerManager *peer.PeerManager
	seenCache   *SeenCache
	httpClient  *http.Client
	logger      *slog.Logger
}

// NewEngine creates a new Gossip Engine.
func NewEngine(pm *peer.PeerManager, cache *SeenCache, logger *slog.Logger) *Engine {
	return &Engine{
		peerManager: pm,
		seenCache:   cache,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// BroadcastTx sends a transaction to all healthy peers.
// It deduplicates internally using the SeenCache and runs asynchronously.
func (e *Engine) BroadcastTx(tx block.Transaction) {
	if !e.seenCache.AddIfNotSeen(tx.ID) {
		return // We've already broadcasted this tx recently
	}

	payload, err := json.Marshal(tx)
	if err != nil {
		e.logger.Error("failed to marshal transaction for gossip", "txID", tx.ID, "error", err)
		return
	}

	e.broadcast("/tx/gossip", payload)
}

// BroadcastBlock sends a newly mined block to all healthy peers.
// It deduplicates internally using the SeenCache and runs asynchronously.
func (e *Engine) BroadcastBlock(b block.Block) {
	// Deduplicate based on Block Hash
	if !e.seenCache.AddIfNotSeen(b.Hash) {
		return
	}

	payload, err := json.Marshal(b)
	if err != nil {
		e.logger.Error("failed to marshal block for gossip", "blockHash", b.Hash, "error", err)
		return
	}

	e.broadcast("/block/gossip", payload)
}

// broadcast is an internal helper that sends the payload to all peers concurrently.
func (e *Engine) broadcast(endpoint string, payload []byte) {
	peers := e.peerManager.GetPeers()
	if len(peers) == 0 {
		return // No peers to gossip to
	}

	for _, p := range peers {
		// Launch a goroutine for each peer so one slow peer doesn't block the others
		go e.sendToPeer(p, endpoint, payload)
	}
}

func (e *Engine) sendToPeer(peerAddr string, endpoint string, payload []byte) {
	// Construct the full URL. peerAddr doesn't have "http://" prefixed if it was normalized.
	url := fmt.Sprintf("http://%s%s", peerAddr, endpoint)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		e.logger.Error("failed to create gossip request", "url", url, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		e.logger.Debug("gossip request failed", "peer", peerAddr, "endpoint", endpoint, "error", err)
		e.peerManager.MarkFailed(peerAddr)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		e.peerManager.MarkSeen(peerAddr)
	} else if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		e.logger.Warn("gossip request rejected by peer due to server error", "peer", peerAddr, "status", resp.StatusCode)
		e.peerManager.MarkFailed(peerAddr)
	} else {
		// 4xx client errors: peer is healthy but rejected our payload
		e.logger.Debug("gossip request rejected by peer (client error)", "peer", peerAddr, "status", resp.StatusCode)
	}
}

// PurgeSeenCache triggers the cleanup of expired entries in the seen cache.
func (e *Engine) PurgeSeenCache() {
	e.seenCache.PurgeOldItems()
}
