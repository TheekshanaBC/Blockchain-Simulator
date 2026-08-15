package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// triggerSync signals the background sync loop to perform a sync immediately,
// debouncing multiple concurrent requests.
func (n *Node) triggerSync() {
	select {
	case n.syncTrigger <- struct{}{}:
	default:
	}
}

func (n *Node) runSync() {
	switched, orphanedTxs, err := n.Syncer.SyncFromBestPeer()
	if err != nil {
		n.Logger.Warn("Periodic sync failed", "error", err)
		return
	}
	if !switched {
		return
	}

	// Remove all transactions that are now in the new chain from the mempool
	var minedTxIDs []string
	for _, b := range n.Chain.GetBlocks() {
		for _, tx := range b.Transactions {
			minedTxIDs = append(minedTxIDs, tx.ID)
		}
	}
	n.Mempool.Remove(minedTxIDs)

	for _, tx := range orphanedTxs {
		n.Mempool.Add(tx)
	}
	if len(orphanedTxs) > 0 {
		n.Logger.Info("Returned orphaned transactions to mempool", "count", len(orphanedTxs))
	}
	n.SaveState()
}

func (n *Node) announceToPeer(peerAddr string) {
	peerURL := peerAddr
	if !strings.HasPrefix(peerURL, "http://") && !strings.HasPrefix(peerURL, "https://") {
		peerURL = "http://" + peerURL
	}
	peerURL = peerURL + "/peers/announce"

	announceAddr := n.Config.AnnounceAddr
	if announceAddr == "" {
		announceAddr = fmt.Sprintf("http://localhost:%d", n.Config.Port)
	}

	payload := map[string]interface{}{
		"address": announceAddr,
		"peers":   n.PeerManager.GetPeers(),
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", peerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		n.Logger.Debug("Failed to announce to peer", "peer", peerAddr, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		n.PeerManager.MarkSeen(peerAddr)

		var respData struct {
			Peers []string `json:"peers"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&respData); err == nil {
			for _, p := range respData.Peers {
				cleanP := strings.TrimPrefix(p, "http://")
				cleanP = strings.TrimPrefix(cleanP, "https://")

				// pm.AddPeer will automatically reject our own address based on normalizeAddress matching
				if isNew := n.PeerManager.AddPeer(cleanP); isNew {
					go n.announceToPeer(cleanP)
				}
			}
		}
	}
}

func (n *Node) healthCheckPeers() {
	peers := n.PeerManager.GetAllPeers()
	client := &http.Client{Timeout: 5 * time.Second}

	for _, pInfo := range peers {
		// We check all peers so they can recover if they come back online before being pruned.
		go func(peerAddr string) {
			url := peerAddr
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "http://" + url
			}
			url = url + "/status"

			resp, err := client.Get(url)
			if err != nil {
				n.PeerManager.MarkFailed(peerAddr)
				n.Logger.Debug("Peer health check failed", "peer", peerAddr, "error", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				n.PeerManager.MarkFailed(peerAddr)
				n.Logger.Debug("Peer health check failed (bad status)", "peer", peerAddr, "status", resp.StatusCode)
			} else {
				n.PeerManager.MarkSeen(peerAddr)
			}
		}(pInfo.Address)
	}
}

func (n *Node) runMempoolSync() {
	peers := n.PeerManager.GetPeers()
	for _, p := range peers {
		txs, err := n.Syncer.SyncMempoolFromPeer(p)
		if err != nil {
			n.Logger.Debug("Failed to pull mempool", "peer", p, "error", err)
			continue
		}

		addedCount := 0
		for _, tx := range txs {
			if !n.Mempool.Has(tx.ID) {
				if err := n.Mempool.ValidateAndAdd(tx, n.Chain.GetBlocks()); err == nil {
					addedCount++
				}
			}
		}

		if addedCount > 0 {
			n.Logger.Info("Pulled mempool transactions", "peer", p, "count", addedCount, "mempool_size", n.Mempool.Size())
		}
	}
}
