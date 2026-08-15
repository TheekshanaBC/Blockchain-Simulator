package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
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
	maxWork := chain.CumulativeWork(s.chain.GetBlocks())

	for _, p := range peers {
		_, _, work, err := s.getPeerHeight(p)
		if err != nil {
			s.logger.Debug("Failed to get height/work from peer", "peer", p, "error", err)
			continue
		}

		if work.Cmp(maxWork) > 0 {
			maxWork = work
			bestPeer = p
		}
	}

	if bestPeer == "" {
		s.logger.Debug("Already at the heaviest chain", "work", maxWork.String())
		return false, nil, nil // We are at the highest work
	}

	s.logger.Info("Found peer with heavier chain, initiating sync", "peer", bestPeer, "target_work", maxWork.String())
	return s.SyncFromPeer(bestPeer)
}

// SyncFromPeer downloads the full chain from the peer and attempts to switch to it.
func (s *Syncer) SyncFromPeer(peerAddr string) (bool, []block.Transaction, error) {
	peerHeight, _, peerWork, err := s.getPeerHeight(peerAddr)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get peer height: %w", err)
	}

	ourWork := chain.CumulativeWork(s.chain.GetBlocks())
	if peerWork.Cmp(ourWork) <= 0 {
		return false, nil, nil // Nothing to sync
	}

	s.logger.Info("Starting chain sync (full download)", "peer", peerAddr, "our_work", ourWork.String(), "peer_work", peerWork.String(), "peer_height", peerHeight)

	// Fetch the entire candidate chain
	url := fmt.Sprintf("%s/chain?limit=100000", peerAddr)
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

func (s *Syncer) getPeerHeight(peerAddr string) (int, string, *big.Int, error) {
	url := fmt.Sprintf("%s/chain/height", peerAddr)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", nil, fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	// Cap the response body to prevent a malicious peer from returning a
	// multi-megabyte "work" string that would cause O(n²) CPU burn in
	// big.Int.SetString. 4 KB is orders of magnitude more than any legitimate
	// /chain/height response needs.
	resp.Body = http.MaxBytesReader(nil, resp.Body, 4096)

	var result struct {
		Height int    `json:"height"`
		Hash   string `json:"hash"`
		Work   string `json:"work"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", nil, err
	}

	work, ok := new(big.Int).SetString(result.Work, 10)
	if !ok {
		work = big.NewInt(0)
	}

	return result.Height, result.Hash, work, nil
}

// SyncMempoolFromPeer fetches the unconfirmed transactions from a peer's mempool
func (s *Syncer) SyncMempoolFromPeer(peerAddr string) ([]block.Transaction, error) {
	url := fmt.Sprintf("%s/mempool", peerAddr)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mempool from peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned status %d", resp.StatusCode)
	}

	var txs []block.Transaction
	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, fmt.Errorf("failed to decode mempool txs: %w", err)
	}

	return txs, nil
}

// PushChainToPeer pushes the current node's entire chain to a peer that is behind.
func (s *Syncer) PushChainToPeer(peerAddr string) error {
	s.logger.Info("Pushing chain to peer", "peer", peerAddr, "height", s.chain.Height())

	blocks := s.chain.GetBlocks()
	payload, err := json.Marshal(blocks)
	if err != nil {
		return fmt.Errorf("failed to marshal chain for push sync: %w", err)
	}

	url := fmt.Sprintf("%s/chain/sync", peerAddr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create push sync request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.peerMgr.MarkFailed(peerAddr)
		return fmt.Errorf("failed to send push sync request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer rejected push sync, status: %d", resp.StatusCode)
	}

	s.logger.Info("Successfully pushed chain to peer", "peer", peerAddr)
	return nil
}
