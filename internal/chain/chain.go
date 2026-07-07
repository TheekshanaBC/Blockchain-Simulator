package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
	"strings"
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

type ValidationResult struct {
	IsValid        bool
	FailedAtHeight int
	Reason         string
}

func (c *Chain) Validate() ValidationResult {
	target := strings.Repeat("0", c.Difficulty)
	for i := 1; i < len(c.Blocks); i++ {
		currentBlock := c.Blocks[i]
		previousBlock := c.Blocks[i-1]

		if currentBlock.Hash != currentBlock.CalculateHash() {
			return ValidationResult{false, currentBlock.Height, "Hash mismatch"}
		}

		if currentBlock.PrevHash != previousBlock.Hash {
			return ValidationResult{false, currentBlock.Height, "Previous Hash mismatch"}
		}

		if !strings.HasPrefix(currentBlock.Hash, target) {
			return ValidationResult{false, currentBlock.Height, "Proof of work failed"}
		}
	}

	return ValidationResult{true, -1, "Chain is Valid"}
}
