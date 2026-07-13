package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
	"strings"
	"time"
)

type Chain struct {
	Blocks      []*block.Block      `json:"blocks"`
	Difficulty  int                 `json:"difficulty"`
	PendingPool []block.Transaction `json:"pending_pool"`
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

	balances := ledger.CalculateAvailableBalances(c.Blocks, c.PendingPool)

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
	IsValid        bool   `json:"is_valid"`
	FailedAtHeight int    `json:"failed_at_height"`
	Reason         string `json:"reason,omitempty"`
}

func (c *Chain) Validate() ValidationResult {

	// Validate Genesis Block
	genesisBlock := c.Blocks[0]
	expectedGenesisHash := "03884eeed6b6115380b8084d617e0927dca73860421d7d9c1ff63adbb9a66e55"

	if genesisBlock.Header.MerkleRoot != block.CalculateMerkleRoot(genesisBlock.Transactions) {
		return ValidationResult{false, 0, "Genesis Merkle Root mismatch"}
	}

	if genesisBlock.CalculateHash() != expectedGenesisHash {
		return ValidationResult{false, 0, "Genesis Hash mismatch"}
	}

	if genesisBlock.Hash != genesisBlock.CalculateHash() {
		return ValidationResult{false, 0, "Genesis Stored Hash Mismatch"}
	}

	// Validate All Other Blocks
	for i := 1; i < len(c.Blocks); i++ {
		currentBlock := c.Blocks[i]
		previousBlock := c.Blocks[i-1]

		if currentBlock.Height != previousBlock.Height+1 {
			return ValidationResult{false, currentBlock.Height, "Block Height mismatch"}
		}

		if currentBlock.Header.Timestamp < previousBlock.Header.Timestamp {
			return ValidationResult{false, currentBlock.Height, "Timestamp is earlier than the previous block"}
		}

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
