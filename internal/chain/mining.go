package chain

import (
	"blockchain-simulator/internal/block"
	"fmt"
	"time"
)

func (c *Chain) MinePendingTransactions() error {
	if len(c.pendingPool) == 0 {
		return fmt.Errorf("no pending transactions to mine")
	}
	c.maybeRetarget()

	lastBlock := c.blocks[len(c.blocks)-1]

	// Limit the number of transactions to MaxTxPerBlock
	numMined := len(c.pendingPool)
	if c.MaxTxPerBlock > 0 && numMined > c.MaxTxPerBlock {
		numMined = c.MaxTxPerBlock
	}

	// Take a snapshot of the selected transactions to mine
	txsToMine := make([]block.Transaction, numMined)
	copy(txsToMine, c.pendingPool[:numMined])

	newBlock := &block.Block{
		Height:       lastBlock.Height + 1,
		Transactions: txsToMine,
		Header: block.BlockHeader{
			PrevHash:  lastBlock.Hash,
			Timestamp: time.Now().Unix(),
		},
	}

	newBlock.Mine(c.Difficulty)

	c.blocks = append(c.blocks, newBlock)
	
	// Remove only the transactions we just mined from the pending pool,
	// preserving any new transactions added while mining was in progress.
	if len(c.pendingPool) >= numMined {
		remainingPool := make([]block.Transaction, len(c.pendingPool)-numMined)
		copy(remainingPool, c.pendingPool[numMined:])
		c.pendingPool = remainingPool
	} else {
		c.pendingPool = []block.Transaction{}
	}

	return nil
}
