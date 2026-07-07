package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
	"time"
)

type Chain struct {
	Blocks      []*block.Block
	Difficulty  int
	PendingPool []block.Transaction
}

func NewChain(difficulty int) *Chain {
	genesis := block.NewGenesisBlock()
	return &Chain{
		Blocks:      []*block.Block{genesis},
		Difficulty:  difficulty,
		PendingPool: []block.Transaction{},
	}
}

func (c *Chain) AddTransaction(tx block.Transaction) error {
	balances := ledger.CalculateBalances(c.Blocks)

	err := ledger.ValidateTransaction(tx, balances)
	if err != nil {
		return fmt.Errorf("Transaction Rejected: %w", err)
	}

	c.PendingPool = append(c.PendingPool, tx)
	return nil
}

func (c *Chain) MinePendingTransactions() error {
	if len(c.PendingPool) == 0 {
		return fmt.Errorf("No pending transactions to mine")
	}

	lastBlock := c.Blocks[len(c.Blocks)-1]

	newBlock := &block.Block{
		Height:       lastBlock.Height + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: c.PendingPool,
		PrevHash:     lastBlock.Hash,
		Nonce:        0,
	}

	newBlock.Mine(c.Difficulty)

	c.Blocks = append(c.Blocks, newBlock)
	c.PendingPool = []block.Transaction{}

	return nil

}
