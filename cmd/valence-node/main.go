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

	flag.IntVar(&port, "port", 3001, "Port to listen on")
	flag.StringVar(&peersStr, "peers", "", "Comma-separated list of peer addresses (e.g. localhost:3002)")
	flag.StringVar(&dataDir, "data-dir", "./data/node1", "Directory to store node data")
	flag.IntVar(&difficulty, "difficulty", 3, "Initial mining difficulty")
	flag.IntVar(&retargetWindow, "retarget-window", 4, "Number of blocks between difficulty retargets")
	flag.Int64Var(&targetBlockTime, "target-block-time", 10, "Target block time in seconds")
	flag.IntVar(&minDiff, "min-diff", 2, "Minimum difficulty")
	flag.IntVar(&maxDiff, "max-diff", 6, "Maximum difficulty")
	flag.Parse()

	var peers []string
	if peersStr != "" {
		peers = strings.Split(peersStr, ",")
	}

	cfg := node.Config{
		Port:            port,
		Peers:           peers,
		DataDir:         dataDir,
		Difficulty:      difficulty,
		RetargetWindow:  retargetWindow,
		TargetBlockTime: targetBlockTime,
		MinDifficulty:   minDiff,
		MaxDifficulty:   maxDiff,
	}

	n := node.NewNode(cfg)

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
