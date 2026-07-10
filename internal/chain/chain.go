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
	if tx.Sender == "COINBASE" {
		return fmt.Errorf("Transaction Rejected: COINBASE is reserved for block mining rewards only")
	}

	balances := ledger.CalculateBalances(c.Blocks, c.PendingPool)

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
		Header: block.BlockHeader{
			PrevHash:  lastBlock.Hash,
			Timestamp: time.Now().Unix(),
			Nonce:     0,
		},
		Height:       lastBlock.Height + 1,
		Transactions: c.PendingPool,
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
	for i := 1; i < len(c.Blocks); i++ {
		currentBlock := c.Blocks[i]
		previousBlock := c.Blocks[i-1]

		if currentBlock.Hash != currentBlock.CalculateHash() {
			return ValidationResult{false, currentBlock.Height, "Hash mismatch"}
		}

		if currentBlock.Header.MerkleRoot != block.CalculateMerkleRoot(currentBlock.Transactions) {
			return ValidationResult{false, currentBlock.Height, "Merkle Root mismatch"}
		}

		if currentBlock.Header.PrevHash != previousBlock.Hash {
			return ValidationResult{false, currentBlock.Height, "Previous Hash mismatch"}
		}

		target := strings.Repeat("0", currentBlock.Header.Difficulty)
		if !strings.HasPrefix(currentBlock.Hash, target) {
			return ValidationResult{false, currentBlock.Height, "Proof of work failed"}
		}
	}

	return ValidationResult{true, -1, "Chain is Valid"}
}
