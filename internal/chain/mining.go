package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
	"time"
)

func (c *Chain) MinePendingTransactions() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.pendingPool) == 0 {
		return fmt.Errorf("no pending transactions to mine")
	}

	// Re-validate and filter the pending pool before mining
	balances := ledger.CalculateAvailableBalances(c.blocks, []block.Transaction{})
	sequences := ledger.CalculatePendingSequences(c.blocks, []block.Transaction{})
	faucetReceived := make(map[string]int64)

	for _, b := range c.blocks {
		for _, tx := range b.Transactions {
			if tx.Sender == block.SystemAddressFaucet {
				faucetReceived[tx.Recipient] += tx.Amount
			}
		}
	}

	var validPool []block.Transaction
	for _, tx := range c.pendingPool {
		if tx.Sender == block.SystemAddressFaucet {
			if tx.Recipient == block.SystemAddressCoinbase {
				continue
			}
			if tx.Amount > MaxFaucetRequest {
				continue
			}
			if faucetReceived[tx.Recipient]+tx.Amount > MaxLifetimeFaucetPerAddress {
				continue
			}
			faucetReceived[tx.Recipient] += tx.Amount
			validPool = append(validPool, tx)
		} else {
			if err := ledger.ValidateTransaction(tx, balances, sequences); err == nil {
				balances[tx.Sender] -= tx.Amount
				balances[tx.Recipient] += tx.Amount
				sequences[tx.Sender] = tx.Sequence
				validPool = append(validPool, tx)
			}
		}
	}
	c.pendingPool = validPool

	if len(c.pendingPool) == 0 {
		return fmt.Errorf("no pending transactions to mine after validation")
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
