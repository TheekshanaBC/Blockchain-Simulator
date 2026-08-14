package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"github.com/lmittmann/tint"
	"valence/internal/chain"
	"valence/internal/gossip"
	"valence/internal/peer"
	"valence/internal/storage"
	chainsync "valence/internal/sync"
	"valence/internal/wallet"
)

type Config struct {
	Port            int
	AnnounceAddr    string // Address to announce to peers (e.g., "http://192.168.1.100:3001")
	Peers           []string // initial peer addresses, e.g., ["localhost:3002"]
	DataDir         string   // per-node data directory, e.g., "./data/node1"
	Difficulty      int
	RetargetWindow  int
	TargetBlockTime int64
	MinDifficulty   int
	MaxDifficulty   int
	MinerAddress    string // address to receive mining rewards (from node's wallet)
	MaxTxPerBlock   int    // maximum transactions allowed per block
	FaucetKey       string // base64 private key for the Faucet wallet
}

type Node struct {
	Config       Config
	Chain        *chain.Chain
	Wallet       *wallet.Wallet
	FaucetWallet *wallet.Wallet
	Mempool      *Mempool
	PeerManager  *peer.PeerManager
	Gossip       *gossip.Engine
	Syncer       *chainsync.Syncer
	Logger       *slog.Logger
	server       *http.Server
	stopChan     chan struct{}
	faucetMu     sync.Mutex
}

func NewNode(cfg Config) (*Node, error) {
	// Setup logger
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.Kitchen,
	}))

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0750); err != nil {
		logger.Error("failed to create data directory", "error", err)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Clean up any stale temp files from previous crashes
	if err := storage.CleanupTempFiles(cfg.DataDir); err != nil {
		logger.Warn("failed to clean up temp files", "error", err)
	}

	var faucetWallet *wallet.Wallet
	if cfg.FaucetKey != "" {
		fw, err := wallet.WalletFromBase64(cfg.FaucetKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse faucet key: %w", err)
		}
		faucetWallet = fw
		logger.Info("Faucet wallet loaded", "address", faucetWallet.Address())
	}

	// Load or create wallet
	keystoreFile := fmt.Sprintf("%s/keystore.json", cfg.DataDir)
	var nodeWallet *wallet.Wallet
	wallets, err := wallet.GetAllWallets(keystoreFile)
	if err == nil && len(wallets) > 0 {
		// Prioritize "node_wallet" if it exists, otherwise pick deterministically
		if w, exists := wallets["node_wallet"]; exists {
			nodeWallet = w
		} else {
			var names []string
			for name := range wallets {
				names = append(names, name)
			}
			sort.Strings(names)
			nodeWallet = wallets[names[0]]
		}
	} else {
		nodeWallet = wallet.NewWallet()
		err = wallet.SaveToKeystore(keystoreFile, "node_wallet", nodeWallet)
		if err != nil {
			logger.Error("failed to save new wallet", "error", err)
			return nil, fmt.Errorf("failed to save new wallet: %w", err)
		}
	}

	if cfg.MinerAddress == "" {
		cfg.MinerAddress = nodeWallet.Address()
	}

	// Initialize chain
	chainFile := fmt.Sprintf("%s/chain.json", cfg.DataDir)
	var c *chain.Chain
	c, err = storage.LoadChain(chainFile)
	if err == nil {
		res := c.Validate()
		if !res.IsValid {
			logger.Warn("Loaded chain is invalid, falling back to genesis", "reason", res.Reason)
			err = fmt.Errorf("invalid chain")
		}
	}
	if err != nil {
		logger.Info("No valid existing chain found, creating genesis chain")
		c = chain.NewChain(cfg.Difficulty, cfg.RetargetWindow, cfg.TargetBlockTime, cfg.MinDifficulty, cfg.MaxDifficulty, cfg.MaxTxPerBlock)
	} else {
		logger.Info("Loaded chain from disk", "height", c.Height(), "file", chainFile)
	}

	selfAddr := cfg.AnnounceAddr
	if selfAddr == "" {
		selfAddr = fmt.Sprintf("http://localhost:%d", cfg.Port)
	}

	pm := peer.NewPeerManager(selfAddr, cfg.Peers)
	cache := gossip.NewSeenCache(1 * time.Hour)
	engine := gossip.NewEngine(pm, cache, logger)
	syncer := chainsync.NewSyncer(c, pm, logger)

	return &Node{
		Config:       cfg,
		Chain:        c,
		Wallet:       nodeWallet,
		FaucetWallet: faucetWallet,
		Mempool:      NewMempool(5000), // Max 5000 transactions in mempool
		PeerManager: pm,
		Gossip:      engine,
		Syncer:      syncer,
		Logger:      logger,
		stopChan:    make(chan struct{}),
	}, nil
}

func (n *Node) Start() error {
	n.Logger.Info("Starting Valence Node", "port", n.Config.Port, "data_dir", n.Config.DataDir, "miner_address", n.Config.MinerAddress)

	// Start background goroutine to purge caches and dead peers
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n.Gossip.PurgeSeenCache()
				n.PeerManager.PruneUnhealthyPeers(1 * time.Hour)
			case <-n.stopChan:
				return
			}
		}
	}()

	// Do initial chain sync before starting HTTP to catch up
	n.runSync()

	// Initial peer announcement
	go func() {
		for _, p := range n.Config.Peers {
			n.announceToPeer(p)
		}
	}()

	// Start background periodic sync
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n.runSync()
			case <-n.stopChan:
				return
			}
		}
	}()

	// Start health check loop
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n.healthCheckPeers()
			case <-n.stopChan:
				return
			}
		}
	}()

	// Setup HTTP endpoints
	mux := http.NewServeMux()
	n.setupAPI(mux)

	n.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", n.Config.Port),
		Handler: corsMiddleware(mux),
	}

	return n.server.ListenAndServe()
}

func (n *Node) Stop() {
	n.Logger.Info("Stopping Valence Node...")
	
	select {
	case <-n.stopChan:
		// already closed
	default:
		close(n.stopChan)
	}

	n.SaveState()

	if n.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := n.server.Shutdown(ctx); err != nil {
			n.Logger.Error("server shutdown error", "error", err)
		}
	}
}

// SaveState saves the current blockchain state to disk
func (n *Node) SaveState() {
	chainFile := fmt.Sprintf("%s/chain.json", n.Config.DataDir)
	if err := storage.SaveChain(n.Chain, chainFile); err != nil {
		n.Logger.Error("Failed to save chain state", "error", err)
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
		// Only check peers that haven't failed too many times, or maybe check all of them?
		// We'll check all of them so they can recover if they come back online before being pruned.
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
