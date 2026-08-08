package chain

import (
	"fmt"
	"time"
	"valence/internal/block"
	"valence/internal/ledger"
)

// MineBlock takes a list of candidate transactions, validates them against the current chain state,
// builds a block, and performs Proof-of-Work WITHOUT holding the chain lock.
// This allows the node to continue processing network requests while mining.
func (c *Chain) MineBlock(txs []block.Transaction, minerAddress string) (block.Block, error) {
	c.mu.RLock()
	
	// Re-validate and filter
	validTxs := ledger.FilterValidTransactions(txs, c.blocks)
	
	// Add Coinbase transaction
	coinbaseTx := block.Transaction{
		Sender:    block.SystemAddressCoinbase,
		Recipient: minerAddress,
		Amount:    block.ElectronsPerVCN * 50, // 50 VCN block reward
	}
	coinbaseTx.ComputeID()
	
	finalTxs := []block.Transaction{coinbaseTx}
	
	maxToAdd := c.MaxTxPerBlock - 1
	if len(validTxs) < maxToAdd {
		maxToAdd = len(validTxs)
	}
	finalTxs = append(finalTxs, validTxs[:maxToAdd]...)
	
	lastBlock := c.blocks[len(c.blocks)-1]
	
	expectedDifficulty := expectedDifficultyAfterWindow(c.blocks, lastBlock.Height+1, c.RetargetWindow, c.TargetBlockTimeSec, c.Difficulty, c.MinDifficulty, c.MaxDifficulty)

	newBlock := block.Block{
		Height:       lastBlock.Height + 1,
		Transactions: finalTxs,
		Header: block.BlockHeader{
			PrevHash:   lastBlock.Hash,
			Timestamp:  time.Now().Unix(),
		},
	}
	
	c.mu.RUnlock()
	
	// Proof-of-Work is computationally intensive and takes time.
	// By running it here without a lock, we achieve non-blocking mining concurrency!
	newBlock.Mine(expectedDifficulty)
	
	// Attempt to add the mined block to the chain.
	err := c.AddBlock(newBlock)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to add mined block: %w", err)
	}
	
	return newBlock, nil
}
