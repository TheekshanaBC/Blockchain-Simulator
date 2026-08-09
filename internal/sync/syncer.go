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
func (s *Syncer) SyncFromBestPeer() error {
	peers := s.peerMgr.GetPeers()
	if len(peers) == 0 {
		return nil // No peers to sync from
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
		return nil // We are at the highest height
	}

	s.logger.Info("Found peer with longer chain, initiating sync", "peer", bestPeer, "target_height", maxHeight)
	return s.SyncFromPeer(bestPeer)
}

// SyncFromPeer downloads missing blocks from the given peer address.
func (s *Syncer) SyncFromPeer(peerAddr string) error {
	peerHeight, _, err := s.getPeerHeight(peerAddr)
	if err != nil {
		return fmt.Errorf("failed to get peer height: %w", err)
	}

	ourHeight := s.chain.Height()
	if peerHeight <= ourHeight {
		return nil // Nothing to sync
	}

	s.logger.Info("Starting chain sync", "peer", peerAddr, "our_height", ourHeight, "peer_height", peerHeight)

	// Fetch each missing block sequentially
	for height := ourHeight + 1; height <= peerHeight; height++ {
		b, err := s.getPeerBlock(peerAddr, height)
		if err != nil {
			s.logger.Warn("Sync failed: could not fetch block", "peer", peerAddr, "height", height, "error", err)
			return err
		}

		// Try to add the block. The Chain.AddBlock method handles validation.
		err = s.chain.AddBlock(*b)
		if err != nil {
			s.logger.Warn("Sync failed: invalid block received", "peer", peerAddr, "height", height, "error", err)
			s.peerMgr.MarkFailed(peerAddr)
			return err
		}

		s.logger.Info("Synced block", "height", height, "hash", b.Hash)
	}

	s.logger.Info("Chain sync completed successfully", "new_height", s.chain.Height())
	return nil
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

func (s *Syncer) getPeerBlock(peerAddr string, height int) (*block.Block, error) {
	url := fmt.Sprintf("http://%s/chain/blocks/%d", peerAddr, height)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	var b block.Block
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}

	return &b, nil
}
