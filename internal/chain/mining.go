package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"valence/internal/block"
	"valence/internal/ledger"
)

// MineBlock takes a list of candidate transactions, validates them against the current chain state,
// builds a block, and performs Proof-of-Work WITHOUT holding the chain lock.
// MineBlock gathers valid transactions from mempool and attempts to mine a new block.
func (c *Chain) MineBlock(ctx context.Context, txs []block.Transaction, minerAddress string) (block.Block, error) {
	if strings.TrimSpace(minerAddress) == "" {
		return block.Block{}, errors.New("miner address cannot be empty")
	}

	c.mu.RLock()
	
	// Re-validate and filter
	validTxs := ledger.FilterValidTransactions(txs, c.blocks)
	
	// Add Coinbase transaction
	coinbaseTx := block.Transaction{
		Sender:    block.SystemAddressCoinbase,
		Recipient: minerAddress,
		Amount:    block.MiningReward,
		Timestamp: time.Now().UnixNano(),
	}
	coinbaseTx.ComputeID()
	
	finalTxs := []block.Transaction{coinbaseTx}
	
	maxToAdd := 0
	if c.MaxTxPerBlock > 1 {
		maxToAdd = c.MaxTxPerBlock - 1
	}
	if len(validTxs) < maxToAdd {
		maxToAdd = len(validTxs)
	}
	finalTxs = append(finalTxs, validTxs[:maxToAdd]...)
	
	if len(c.blocks) == 0 {
		c.mu.RUnlock()
		return block.Block{}, fmt.Errorf("cannot mine: chain has no blocks")
	}
	lastBlock := c.blocks[len(c.blocks)-1]
	
	expectedDifficulty := expectedDifficultyAfterWindow(c.blocks, lastBlock.Height+1, c.RetargetWindow, c.TargetBlockTimeSec, c.Difficulty, c.MinDifficulty, c.MaxDifficulty)

	newBlock := block.Block{
		Height:       lastBlock.Height + 1,
		Transactions: finalTxs,
		Header: block.BlockHeader{
			PrevHash:   lastBlock.Hash,
			Timestamp:  time.Now().UnixNano(),
		},
	}
	
	c.mu.RUnlock()
	
	// Proof-of-Work is computationally intensive and takes time.
	// By running it here without a lock, we achieve non-blocking mining concurrency!
	newBlock.Mine(ctx, expectedDifficulty)
	
	// Attempt to add the mined block to the chain.
	if newBlock.Hash == "" {
		return block.Block{}, errors.New("mining was cancelled")
	}

	err := c.AddBlock(newBlock)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to add mined block: %w", err)
	}
	
	return newBlock, nil
}
