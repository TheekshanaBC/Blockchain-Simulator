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
	Logger      *slog.Logger
	server *http.Server
	logger *slog.Logger
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
	c := chain.NewChain(cfg.Difficulty, cfg.RetargetWindow, cfg.TargetBlockTime, cfg.MinDifficulty, cfg.MaxDifficulty)

	selfAddr := fmt.Sprintf("http://localhost:%d", cfg.Port)

	pm := peer.NewPeerManager(selfAddr, cfg.Peers)
	cache := gossip.NewSeenCache(1 * time.Hour)
	engine := gossip.NewEngine(pm, cache, logger)

	return &Node{
		Config:      cfg,
		Chain:       c,
		Wallet:      nodeWallet,
		Mempool:     NewMempool(),
		PeerManager: pm,
		Logger:      logger,
		Gossip:      engine,
		logger:      logger,
	}, nil
}

func (n *Node) Start() error {
	n.logger.Info("Starting Valence Node", "port", n.Config.Port, "data_dir", n.Config.DataDir, "miner_address", n.Config.MinerAddress)

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
	n.logger.Info("Stopping Valence Node...")
	if n.server != nil {
		n.server.Shutdown(context.Background())
	}
}
