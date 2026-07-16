package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
	"strings"
)

type ValidationResult struct {
	IsValid        bool   `json:"is_valid"`
	FailedAtHeight int    `json:"failed_at_height"`
	Reason         string `json:"reason,omitempty"`
}

func (c *Chain) Validate() ValidationResult {
	if len(c.Blocks) == 0 {
		return ValidationResult{false, 0, "Chain is empty"}
	}

	balances := make(map[string]int64)
	sequences := make(map[string]uint64)
	faucetReceived := make(map[string]int64)

	res := validateGenesisBlock(c.Blocks[0], balances, sequences, faucetReceived)
	if !res.IsValid {
		return res
	}

	if len(c.Blocks) < 2 {
		return ValidationResult{true, -1, "Chain is Valid"}
	}

	expectedDifficulty := c.Blocks[1].Header.Difficulty
	for i := 1; i < len(c.Blocks); i++ {
		currentBlock := c.Blocks[i]
		previousBlock := c.Blocks[i-1]

		expectedDifficulty = expectedDifficultyAfterWindow(c.Blocks, currentBlock.Height, c.RetargetWindow, c.TargetBlockTimeSec, expectedDifficulty, c.MinDifficulty, c.MaxDifficulty)

		res = validateBlockAgainstPrevious(currentBlock, previousBlock, expectedDifficulty)
		if !res.IsValid {
			return res
		}

		res = validateBlockTransactions(currentBlock, balances, sequences, faucetReceived)
		if !res.IsValid {
			return res
		}
	}

	return ValidationResult{true, -1, "Chain is Valid"}
}

func validateGenesisBlock(genesisBlock *block.Block, balances map[string]int64, sequences map[string]uint64, faucetReceived map[string]int64) ValidationResult {
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

	for _, tx := range genesisBlock.Transactions {
		if tx.Sender == block.SystemAddressFaucet {
			faucetReceived[tx.Recipient] += tx.Amount
		} else if !block.IsSystemAddress(tx.Sender) {
			balances[tx.Sender] -= tx.Amount
			sequences[tx.Sender] = tx.Sequence
		}
		balances[tx.Recipient] += tx.Amount
	}
	return ValidationResult{true, -1, ""}
}

func validateBlockAgainstPrevious(currentBlock, previousBlock *block.Block, expectedDifficulty int) ValidationResult {
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

	if currentBlock.Header.Difficulty != expectedDifficulty {
		return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Difficulty retarget mismatch: expected %d, got %d", expectedDifficulty, currentBlock.Header.Difficulty)}
	}

	target := strings.Repeat("0", currentBlock.Header.Difficulty)

	if !strings.HasPrefix(currentBlock.Hash, target) {
		return ValidationResult{false, currentBlock.Height, "Proof of work failed"}
	}
	return ValidationResult{true, -1, ""}
}

func validateBlockTransactions(currentBlock *block.Block, balances map[string]int64, sequences map[string]uint64, faucetReceived map[string]int64) ValidationResult {
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

		if strings.TrimSpace(tx.Recipient) == "" {
			return ValidationResult{false, currentBlock.Height, "Recipient address cannot be empty"}
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

		if tx.Sender == block.SystemAddressFaucet {
			if tx.Amount > MaxFaucetRequest {
				return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Faucet request exceeds maximum allowed limit per request (%d)", MaxFaucetRequest)}
			}
			if faucetReceived[tx.Recipient]+tx.Amount > MaxLifetimeFaucetPerAddress {
				return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Lifetime faucet limit exceeded for address (Max: %d)", MaxLifetimeFaucetPerAddress)}
			}
			faucetReceived[tx.Recipient] += tx.Amount
		} else if !block.IsSystemAddress(tx.Sender) {
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
	return ValidationResult{true, -1, ""}
}
