package chain

import (
	"blockchain-simulator/internal/block"
	"fmt"
	"time"
)

func (c *Chain) MinePendingTransactions() error {
	if len(c.PendingPool) == 0 {
		return fmt.Errorf("No pending transactions to mine")
	}
	c.maybeRetarget()

	lastBlock := c.Blocks[len(c.Blocks)-1]

	newBlock := &block.Block{
		Height:       lastBlock.Height + 1,
		Transactions: c.PendingPool,
		Header: block.BlockHeader{
			PrevHash:  lastBlock.Hash,
			Timestamp: time.Now().Unix(),
		},
	}

	newBlock.Mine(c.Difficulty)

	c.Blocks = append(c.Blocks, newBlock)
	c.PendingPool = []block.Transaction{}

	return nil
}
