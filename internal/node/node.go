package node

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"valence/internal/chain"
	"valence/internal/gossip"
	"valence/internal/peer"
	"valence/internal/storage"
	chainsync "valence/internal/sync"
	"valence/internal/wallet"
)

type Config struct {
	Port            int
	Peers           []string // initial peer addresses, e.g., ["localhost:3002"]
	DataDir         string   // per-node data directory, e.g., "./data/node1"
	Difficulty      int
	RetargetWindow  int
	TargetBlockTime int64
	MinDifficulty   int
	MaxDifficulty   int
	MinerAddress    string // address to receive mining rewards (from node's wallet)
}

type Node struct {
	Config      Config
	Chain       *chain.Chain
	Wallet      *wallet.Wallet
	Mempool     *Mempool
	PeerManager *peer.PeerManager
	Gossip      *gossip.Engine
	Syncer      *chainsync.Syncer
	Logger      *slog.Logger
	server      *http.Server
}

func NewNode(cfg Config) (*Node, error) {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0750); err != nil {
		logger.Error("failed to create data directory", "error", err)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load or create wallet
	keystoreFile := fmt.Sprintf("%s/keystore.json", cfg.DataDir)
	var nodeWallet *wallet.Wallet
	wallets, err := wallet.GetAllWallets(keystoreFile)
	if err == nil && len(wallets) > 0 {
		// Pick the first available wallet in a non-deterministic way since map iteration is random.
		// For a single-wallet keystore, this works fine.
		for _, w := range wallets {
			nodeWallet = w
			break
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
	if err != nil {
		logger.Info("No existing chain found, creating genesis chain")
		c = chain.NewChain(cfg.Difficulty, cfg.RetargetWindow, cfg.TargetBlockTime, cfg.MinDifficulty, cfg.MaxDifficulty)
	} else {
		logger.Info("Loaded chain from disk", "height", c.Height(), "file", chainFile)
	}

	selfAddr := fmt.Sprintf("http://localhost:%d", cfg.Port)

	pm := peer.NewPeerManager(selfAddr, cfg.Peers)
	cache := gossip.NewSeenCache(1 * time.Hour)
	engine := gossip.NewEngine(pm, cache, logger)
	syncer := chainsync.NewSyncer(c, pm, logger)

	return &Node{
		Config:      cfg,
		Chain:       c,
		Wallet:      nodeWallet,
		Mempool:     NewMempool(),
		PeerManager: pm,
		Gossip:      engine,
		Syncer:      syncer,
		Logger:      logger,
	}, nil
}

func (n *Node) Start() error {
	n.Logger.Info("Starting Valence Node", "port", n.Config.Port, "data_dir", n.Config.DataDir, "miner_address", n.Config.MinerAddress)

	// Start background goroutine to purge SeenCache
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			n.Gossip.PurgeSeenCache()
		}
	}()

	// Do initial chain sync before starting HTTP to catch up
	n.runSync()

	// Start background periodic sync
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			n.runSync()
		}
	}()

	// Setup HTTP endpoints
	mux := http.NewServeMux()
	n.setupAPI(mux)

	n.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", n.Config.Port),
		Handler: mux,
	}

	return n.server.ListenAndServe()
}

func (n *Node) Stop() {
	n.Logger.Info("Stopping Valence Node...")
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
	orphanedTxs, err := n.Syncer.SyncFromBestPeer()
	if err != nil {
		n.Logger.Warn("Periodic sync failed", "error", err)
		return
	}
	for _, tx := range orphanedTxs {
		n.Mempool.Add(tx)
	}
	if len(orphanedTxs) > 0 {
		n.Logger.Info("Returned orphaned transactions to mempool", "count", len(orphanedTxs))
	}
}
