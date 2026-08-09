package chain

import (
	"fmt"
	"strings"
	"time"
	"valence/internal/block"
	"valence/internal/ledger"
)

type ValidationResult struct {
	IsValid        bool   `json:"is_valid"`
	FailedAtHeight int    `json:"failed_at_height"`
	Reason         string `json:"reason,omitempty"`
}

func (c *Chain) Validate() ValidationResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return ValidateBlockSlice(c.blocks, c.InitialDifficulty, c.RetargetWindow, c.TargetBlockTimeSec, c.MinDifficulty, c.MaxDifficulty)
}

func ValidateBlockSlice(blocks []*block.Block, initialDifficulty, retargetWindow int, targetBlockTimeSec int64, minDifficulty, maxDifficulty int) ValidationResult {
	if len(blocks) == 0 {
		return ValidationResult{false, 0, "Chain is empty"}
	}

	balances := make(map[string]int64)
	sequences := make(map[string]uint64)
	faucetReceived := make(map[string]int64)

	res := validateGenesisBlock(blocks[0], balances, sequences, faucetReceived)
	if !res.IsValid {
		return res
	}

	if len(blocks) < 2 {
		return ValidationResult{true, -1, "Chain is Valid"}
	}

	expectedDifficulty := initialDifficulty
	if expectedDifficulty < minDifficulty {
		expectedDifficulty = minDifficulty
	}
	if expectedDifficulty > maxDifficulty {
		expectedDifficulty = maxDifficulty
	}
	for i := 1; i < len(blocks); i++ {
		currentBlock := blocks[i]
		previousBlock := blocks[i-1]

		if currentBlock.Height != previousBlock.Height+1 {
			return ValidationResult{false, currentBlock.Height, "Block Height mismatch"}
		}

		expectedDifficulty = expectedDifficultyAfterWindow(blocks, currentBlock.Height, retargetWindow, targetBlockTimeSec, expectedDifficulty, minDifficulty, maxDifficulty)

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
	
	if currentBlock.Header.Timestamp > time.Now().Unix()+7200 {
		return ValidationResult{false, currentBlock.Height, "Timestamp is too far in the future"}
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

		// Delegate general validation to ledger
		err := ledger.ValidateTransaction(tx, balances, sequences, faucetReceived)
		if err != nil {
			return ValidationResult{false, currentBlock.Height, err.Error()}
		}

		if tx.Sender == block.SystemAddressFaucet {
			faucetReceived[tx.Recipient] += tx.Amount
		} else if !block.IsSystemAddress(tx.Sender) {
			sequences[tx.Sender] = tx.Sequence
			balances[tx.Sender] -= tx.Amount
		}
		balances[tx.Recipient] += tx.Amount
	}
	return ValidationResult{true, -1, ""}
}
