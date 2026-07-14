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
	if block.IsSystemAddress(tx.Sender) {
		return fmt.Errorf("Transaction Rejected: %s is reserved for system use only", tx.Sender)
	}

	balances := ledger.CalculateAvailableBalances(c.Blocks, c.PendingPool)
	sequences := ledger.CalculatePendingSequences(c.Blocks, c.PendingPool)

	err := ledger.ValidateTransaction(tx, balances, sequences)
	if err != nil {
		return fmt.Errorf("Transaction Rejected: %w", err)
	}

	c.PendingPool = append(c.PendingPool, tx)
	return nil
}

const MaxFaucetRequest int64 = 1000
const MaxLifetimeFaucetPerAddress int64 = 5000

// RequestFaucetFunds creates and adds a system-approved FAUCET transaction to the pending pool.
// This bypasses the AddTransaction sender check but enforces its own limits.
func (c *Chain) RequestFaucetFunds(recipient string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("Faucet amount must be strictly positive")
	}
	if amount > MaxFaucetRequest {
		return fmt.Errorf("Faucet request exceeds maximum allowed limit per request (%d)", MaxFaucetRequest)
	}

	// Calculate total amount already given to this address by the faucet
	var totalReceived int64 = 0
	for _, b := range c.Blocks {
		for _, tx := range b.Transactions {
			if tx.Sender == block.SystemAddressFaucet && tx.Recipient == recipient {
				totalReceived += tx.Amount
			}
		}
	}
	for _, tx := range c.PendingPool {
		if tx.Sender == block.SystemAddressFaucet && tx.Recipient == recipient {
			totalReceived += tx.Amount
		}
	}

	if totalReceived+amount > MaxLifetimeFaucetPerAddress {
		return fmt.Errorf("Lifetime faucet limit exceeded for address (Max: %d, Already Received: %d)", MaxLifetimeFaucetPerAddress, totalReceived)
	}

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: recipient,
		Amount:    amount,
		// FAUCET transactions don't need a signature or sequence
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

	if len(c.Blocks) == 0 {
		return ValidationResult{false, 0, "Chain is empty"}
	}

	// Validate Genesis Block
	genesisBlock := c.Blocks[0]
	expectedGenesisHash := block.NewGenesisBlock().Hash

	if genesisBlock.Header.MerkleRoot != block.CalculateMerkleRoot(genesisBlock.Transactions) {
		return ValidationResult{false, 0, "Genesis Merkle Root mismatch"}
	}

	if genesisBlock.CalculateHash() != expectedGenesisHash {
		return ValidationResult{false, 0, "Genesis Hash mismatch"}
	}

	if genesisBlock.Hash != genesisBlock.CalculateHash() {
		return ValidationResult{false, 0, "Genesis Stored Hash Mismatch"}
	}

	if genesisBlock.Header.Difficulty != 0 {
		return ValidationResult{false, 0, "Genesis difficulty should be 0"}
	}

	balances := make(map[string]int64)
	sequences := make(map[string]uint64)
	for _, tx := range genesisBlock.Transactions {
		if !block.IsSystemAddress(tx.Sender) {
			balances[tx.Sender] -= tx.Amount
			sequences[tx.Sender] = tx.Sequence
		}
		balances[tx.Recipient] += tx.Amount
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

		if currentBlock.Header.Difficulty != c.Difficulty {
			return ValidationResult{false, currentBlock.Height, "Block difficulty does not match chain difficulty"}
		}

		target := strings.Repeat("0", c.Difficulty)

		if !strings.HasPrefix(currentBlock.Hash, target) {
			return ValidationResult{false, currentBlock.Height, "Proof of work failed"}
		}

		if len(currentBlock.Transactions) == 0 {
			return ValidationResult{false, currentBlock.Height, "Block must contain at least one transaction (COINBASE)"}
		}

		for i, tx := range currentBlock.Transactions {
			// Enforce strict COINBASE rules
			if i == 0 {
				if tx.Sender != block.SystemAddressCoinbase {
					return ValidationResult{false, currentBlock.Height, "First transaction must be COINBASE"}
				}
				if tx.Amount != block.MiningReward {
					return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Invalid COINBASE reward: expected %d, got %d", block.MiningReward, tx.Amount)}
				}
			} else {
				if tx.Sender == block.SystemAddressCoinbase {
					return ValidationResult{false, currentBlock.Height, "COINBASE transaction can only be the first transaction in a block"}
				}
			}

			if tx.Recipient == block.SystemAddressCoinbase {
				return ValidationResult{false, currentBlock.Height, "Cannot send funds to COINBASE address."}
			}

			if tx.Amount <= 0 {
				return ValidationResult{false, currentBlock.Height, "Transaction amount must be strictly positive"}
			}
			if tx.Amount > ledger.MaxTransactionAmount {
				return ValidationResult{false, currentBlock.Height, "Transaction amount exceeds maximum allowed limit"}
			}
			if !tx.Verify() {
				return ValidationResult{false, currentBlock.Height, "Invalid transaction signature"}
			}
			if !block.IsSystemAddress(tx.Sender) {
				expectedSeq := sequences[tx.Sender] + 1
				if tx.Sequence != expectedSeq {
					return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Ledger replay failed: invalid sequence for %s (expected %d, got %d)", tx.Sender, expectedSeq, tx.Sequence)}
				}
				sequences[tx.Sender] = tx.Sequence

				balances[tx.Sender] -= tx.Amount
				if balances[tx.Sender] < 0 {
					return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Ledger replay failed: negative balance for %s", tx.Sender)}
				}
			}
			balances[tx.Recipient] += tx.Amount
		}
	}

	return ValidationResult{true, -1, "Chain is Valid"}
}
