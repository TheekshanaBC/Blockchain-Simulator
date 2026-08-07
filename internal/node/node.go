package node

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"valence/internal/chain"
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
	mu     sync.RWMutex
	Config Config
	Chain  *chain.Chain
	Wallet  *wallet.Wallet
	Mempool *Mempool
	// PeerManager *peer.PeerManager   // defined in Task 2.3
	// Gossip      *gossip.Engine      // defined in Sprint 3 (nil initially)
	server *http.Server
	logger *slog.Logger
}

func NewNode(cfg Config) *Node {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0750); err != nil {
		logger.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	// Load or create wallet
	keystoreFile := fmt.Sprintf("%s/keystore.json", cfg.DataDir)
	var nodeWallet *wallet.Wallet
	wallets, err := wallet.GetAllWallets(keystoreFile)
	if err == nil && len(wallets) > 0 {
		// Just take the first one
		for _, w := range wallets {
			nodeWallet = w
			break
		}
	} else {
		nodeWallet = wallet.NewWallet()
		err = wallet.SaveToKeystore(keystoreFile, "node_wallet", nodeWallet)
		if err != nil {
			logger.Error("failed to save new wallet", "error", err)
			os.Exit(1)
		}
	}

	if cfg.MinerAddress == "" {
		cfg.MinerAddress = nodeWallet.Address()
	}

	// Initialize chain
	c := chain.NewChain(cfg.Difficulty, cfg.RetargetWindow, cfg.TargetBlockTime, cfg.MinDifficulty, cfg.MaxDifficulty)

	return &Node{
		Config:  cfg,
		Chain:   c,
		Wallet:  nodeWallet,
		Mempool: NewMempool(),
		logger:  logger,
	}
}

func (n *Node) Start() error {
	n.logger.Info("Starting Valence Node", "port", n.Config.Port, "data_dir", n.Config.DataDir, "miner_address", n.Config.MinerAddress)

	// Create a dummy HTTP handler for now
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

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
