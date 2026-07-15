package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
	"strings"
	"time"
)

type Chain struct {
	Blocks             []*block.Block      `json:"blocks"`
	Difficulty         int                 `json:"difficulty"`
	PendingPool        []block.Transaction `json:"pending_pool"`
	RetargetWindow     int                 `json:"retarget_window"`
	TargetBlockTimeSec int64               `json:"target_block_time_sec"`
	MaxDifficulty      int                 `json:"max_difficulty"`
	MinDifficulty      int                 `json:"min_difficulty"`
}

func NewChain(difficulty int, retargetWindow int, targetBlockTimeSec int64, maxDifficulty int, minDifficulty int) *Chain {
	if retargetWindow < 2 {
		retargetWindow = 2
	}
	genesis := block.NewGenesisBlock()
	return &Chain{
		Blocks:             []*block.Block{genesis},
		Difficulty:         difficulty,
		PendingPool:        []block.Transaction{},
		RetargetWindow:     retargetWindow,
		TargetBlockTimeSec: targetBlockTimeSec,
		MaxDifficulty:      maxDifficulty,
		MinDifficulty:      minDifficulty,
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
	if strings.TrimSpace(recipient) == "" {
		return fmt.Errorf("Recipient address cannot be empty")
	}
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

func expectedDifficultyAfterWindow(blocks []*block.Block, nextHeight, N int, targetBlockTime int64, prevDifficulty, min, max int) int {
	if nextHeight <= 1 || (nextHeight-1)%N != 0 {
		return prevDifficulty
	}

	lastBlock := blocks[nextHeight-1]

	windowIndex := (nextHeight - 1) / N
	var firstBlockIndex int
	var expectedIntervals int

	if windowIndex == 1 {
		firstBlockIndex = 1
		expectedIntervals = N - 1
	} else {
		firstBlockIndex = nextHeight - 1 - N
		expectedIntervals = N
	}

	firstBlock := blocks[firstBlockIndex]
	actual := lastBlock.Header.Timestamp - firstBlock.Header.Timestamp
	expected := targetBlockTime * int64(expectedIntervals)

	return adjustDifficulty(prevDifficulty, actual, expected, min, max)
}

func adjustDifficulty(current int, actual, expected int64, min, max int) int {
	if actual < expected {
		current++
	} else if actual > expected {
		current--
	}
	if current < min {
		current = min
	}
	if current > max {
		current = max
	}
	return current
}
func (c *Chain) maybeRetarget() bool {
	nextHeight := len(c.Blocks)
	newDiff := expectedDifficultyAfterWindow(c.Blocks, nextHeight, c.RetargetWindow, c.TargetBlockTimeSec, c.Difficulty, c.MinDifficulty, c.MaxDifficulty)
	if newDiff != c.Difficulty {
		c.Difficulty = newDiff
		return true
	}
	return false
}

func (c *Chain) MinePendingTransactions() error {
	if len(c.PendingPool) == 0 {
		return fmt.Errorf("No pending transactions to mine")
	}
	c.maybeRetarget()

	lastBlock := c.Blocks[len(c.Blocks)-1]

	newBlock := &block.Block{
		Height:       lastBlock.Height + 1,
		Transactions: c.PendingPool,
		Header: block.BlockHeader{
			PrevHash:  lastBlock.Hash,
			Timestamp: time.Now().Unix(),
		},
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
	faucetReceived := make(map[string]int64)
	for _, tx := range genesisBlock.Transactions {
		if tx.Sender == block.SystemAddressFaucet {
			faucetReceived[tx.Recipient] += tx.Amount
		} else if !block.IsSystemAddress(tx.Sender) {
			balances[tx.Sender] -= tx.Amount
			sequences[tx.Sender] = tx.Sequence
		}
		balances[tx.Recipient] += tx.Amount
	}

	// Validate All Other Blocks
	if len(c.Blocks) < 2 {
		return ValidationResult{true, -1, "Chain is Valid"}
	}
	expectedDifficulty := c.Blocks[1].Header.Difficulty
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

		expectedDifficulty = expectedDifficultyAfterWindow(c.Blocks, currentBlock.Height, c.RetargetWindow, c.TargetBlockTimeSec, expectedDifficulty, c.MinDifficulty, c.MaxDifficulty)

		if currentBlock.Header.Difficulty != expectedDifficulty {
			return ValidationResult{false, currentBlock.Height, fmt.Sprintf("Difficulty retarget mismatch: expected %d, got %d", expectedDifficulty, currentBlock.Header.Difficulty)}
		}

		target := strings.Repeat("0", currentBlock.Header.Difficulty)

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
	}

	return ValidationResult{true, -1, "Chain is Valid"}
}
