package chain

import (
	"blockchain-simulator/internal/block"
	"fmt"
	"time"
)

func (c *Chain) MinePendingTransactions() error {
	if len(c.PendingPool) == 0 {
		return fmt.Errorf("no pending transactions to mine")
	}
	c.maybeRetarget()

	lastBlock := c.Blocks[len(c.Blocks)-1]

	// Limit the number of transactions to MaxTxPerBlock
	numMined := len(c.PendingPool)
	if c.MaxTxPerBlock > 0 && numMined > c.MaxTxPerBlock {
		numMined = c.MaxTxPerBlock
	}

	// Take a snapshot of the selected transactions to mine
	txsToMine := make([]block.Transaction, numMined)
	copy(txsToMine, c.PendingPool[:numMined])

	newBlock := &block.Block{
		Height:       lastBlock.Height + 1,
		Transactions: txsToMine,
		Header: block.BlockHeader{
			PrevHash:  lastBlock.Hash,
			Timestamp: time.Now().Unix(),
		},
	}

	newBlock.Mine(c.Difficulty)

	c.Blocks = append(c.Blocks, newBlock)
	
	// Remove only the transactions we just mined from the pending pool,
	// preserving any new transactions added while mining was in progress.
	if len(c.PendingPool) >= numMined {
		remainingPool := make([]block.Transaction, len(c.PendingPool)-numMined)
		copy(remainingPool, c.PendingPool[numMined:])
		c.PendingPool = remainingPool
	} else {
		c.PendingPool = []block.Transaction{}
	}

	return nil
}
