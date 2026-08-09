package sync

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"valence/internal/block"
	"valence/internal/chain"
	"valence/internal/peer"
)

type Syncer struct {
	chain      *chain.Chain
	peerMgr    *peer.PeerManager
	httpClient *http.Client
	logger     *slog.Logger
}

func NewSyncer(c *chain.Chain, pm *peer.PeerManager, logger *slog.Logger) *Syncer {
	return &Syncer{
		chain:   c,
		peerMgr: pm,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// SyncFromBestPeer queries all healthy peers to find the one with the longest chain,
// and then synchronizes from that peer.
func (s *Syncer) SyncFromBestPeer() (bool, []block.Transaction, error) {
	peers := s.peerMgr.GetPeers()
	if len(peers) == 0 {
		return false, nil, nil // No peers to sync from
	}

	bestPeer := ""
	maxHeight := s.chain.Height()

	for _, p := range peers {
		height, _, err := s.getPeerHeight(p)
		if err != nil {
			s.logger.Debug("Failed to get height from peer", "peer", p, "error", err)
			continue
		}

		if height > maxHeight {
			maxHeight = height
			bestPeer = p
		}
	}

	if bestPeer == "" {
		s.logger.Debug("Already at the highest chain height", "height", s.chain.Height())
		return false, nil, nil // We are at the highest height
	}

	s.logger.Info("Found peer with longer chain, initiating sync", "peer", bestPeer, "target_height", maxHeight)
	return s.SyncFromPeer(bestPeer)
}

// SyncFromPeer downloads the full chain from the peer and attempts to switch to it.
func (s *Syncer) SyncFromPeer(peerAddr string) (bool, []block.Transaction, error) {
	peerHeight, _, err := s.getPeerHeight(peerAddr)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get peer height: %w", err)
	}

	ourHeight := s.chain.Height()
	if peerHeight <= ourHeight {
		return false, nil, nil // Nothing to sync
	}

	s.logger.Info("Starting chain sync (full download)", "peer", peerAddr, "our_height", ourHeight, "peer_height", peerHeight)

	// Fetch the entire candidate chain
	url := fmt.Sprintf("http://%s/chain?limit=100000", peerAddr)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return false, nil, fmt.Errorf("failed to fetch chain from peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	var candidateChain []*block.Block
	if err := json.NewDecoder(resp.Body).Decode(&candidateChain); err != nil {
		return false, nil, fmt.Errorf("failed to decode candidate chain: %w", err)
	}

	// Try to switch to the new chain
	orphanedTxs, err := s.chain.SwitchToChain(candidateChain)
	if err != nil {
		s.logger.Warn("Sync failed: candidate chain invalid or shorter", "peer", peerAddr, "error", err)
		s.peerMgr.MarkFailed(peerAddr)
		return false, nil, err
	}

	s.logger.Info("Chain sync/reorg completed successfully", "new_height", s.chain.Height())
	return true, orphanedTxs, nil
}

func (s *Syncer) getPeerHeight(peerAddr string) (int, string, error) {
	url := fmt.Sprintf("http://%s/chain/height", peerAddr)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	var result struct {
		Height int    `json:"height"`
		Hash   string `json:"hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", err
	}

	return result.Height, result.Hash, nil
}

