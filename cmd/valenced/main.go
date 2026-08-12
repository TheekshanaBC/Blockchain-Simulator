package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"valence/internal/node"
)

func main() {
	var port int
	var peersStr string
	var dataDir string
	var difficulty int
	var retargetWindow int
	var targetBlockTime int64
	var minDiff int
	var maxDiff int
	var minerAddress string
	var maxTxPerBlock int

	flag.IntVar(&port, "port", 3001, "Port to listen on")
	flag.StringVar(&peersStr, "peers", "", "Comma-separated list of peer addresses (e.g. localhost:3002)")
	flag.StringVar(&dataDir, "data-dir", "./data/node1", "Directory to store node data")
	flag.IntVar(&difficulty, "difficulty", 3, "Initial mining difficulty")
	flag.IntVar(&retargetWindow, "retarget-window", 4, "Number of blocks between difficulty retargets")
	flag.Int64Var(&targetBlockTime, "target-block-time", 10, "Target block time in seconds")
	flag.IntVar(&minDiff, "min-diff", 2, "Minimum difficulty")
	flag.IntVar(&maxDiff, "max-diff", 6, "Maximum difficulty")
	flag.StringVar(&minerAddress, "miner-address", "", "Address to receive mining rewards")
	flag.IntVar(&maxTxPerBlock, "max-tx-per-block", 10, "Maximum transactions per block")
	var announceAddr string
	flag.StringVar(&announceAddr, "announce-addr", "", "Address to announce to peers (e.g. http://192.168.1.100:3001)")
	flag.Parse()

	// Fix R3: Fail fast on invalid flag combinations rather than silently stalling later.
	// A negative min-diff causes strings.Repeat to panic; max < min causes an impossible
	// difficulty range; bad ports are caught here before the OS gives an unhelpful error.
	if port < 1 || port > 65535 {
		slog.Error("invalid -port: must be in range 1–65535", "port", port)
		os.Exit(1)
	}
	if minDiff < 0 {
		slog.Error("invalid -min-diff: must be >= 0", "min-diff", minDiff)
		os.Exit(1)
	}
	if maxDiff < minDiff {
		slog.Error("invalid -max-diff: must be >= -min-diff", "min-diff", minDiff, "max-diff", maxDiff)
		os.Exit(1)
	}
	if difficulty < minDiff || difficulty > maxDiff {
		slog.Error("invalid -difficulty: must be in [min-diff, max-diff]", "difficulty", difficulty, "min-diff", minDiff, "max-diff", maxDiff)
		os.Exit(1)
	}
	if maxTxPerBlock < 1 {
		slog.Error("invalid -max-tx-per-block: must be >= 1", "max-tx-per-block", maxTxPerBlock)
		os.Exit(1)
	}

	var peers []string
	if peersStr != "" {
		peers = strings.Split(peersStr, ",")
	}

	cfg := node.Config{
		Port:            port,
		AnnounceAddr:    announceAddr,
		Peers:           peers,
		DataDir:         dataDir,
		Difficulty:      difficulty,
		RetargetWindow:  retargetWindow,
		TargetBlockTime: targetBlockTime,
		MinDifficulty:   minDiff,
		MaxDifficulty:   maxDiff,
		MinerAddress:    minerAddress,
		MaxTxPerBlock:   maxTxPerBlock,
	}

	n, err := node.NewNode(cfg)
	if err != nil {
		slog.Error("failed to initialize node", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		n.Stop()
	}()

	if err := n.Start(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
