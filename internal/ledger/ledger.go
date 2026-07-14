package ledger

import (
	"blockchain-simulator/internal/block"
	"errors"
	"fmt"
)

// MaxTransactionAmount is the maximum coins allowed in a single transaction
// to prevent int64 overflow vulnerabilities.
const MaxTransactionAmount = 1_000_000_000

func CalculateBalances(chain []*block.Block) map[string]int64 {
	balances := make(map[string]int64)

	for _, b := range chain {
		for _, tx := range b.Transactions {
			if tx.Amount == 0 {
				continue
			}

			if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
				balances[tx.Sender] -= tx.Amount
			}
			balances[tx.Recipient] += tx.Amount
		}
	}
	return balances
}

// CalculateAvailableBalances returns the balance available to spend (ledger minus pending outbounds)
func CalculateAvailableBalances(chain []*block.Block, pendingPool []block.Transaction) map[string]int64 {
	balances := CalculateBalances(chain)

	// deduct the pending pool transactions to prevent double spending
	for _, tx := range pendingPool {
		if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
			balances[tx.Sender] -= tx.Amount
		}
	}
	return balances
}

// CalculateSequences returns the highest sequence number used by each address in the blockchain
func CalculateSequences(chain []*block.Block) map[string]uint64 {
	sequences := make(map[string]uint64)
	for _, b := range chain {
		for _, tx := range b.Transactions {
			if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
				if tx.Sequence > sequences[tx.Sender] {
					sequences[tx.Sender] = tx.Sequence
				}
			}
		}
	}
	return sequences
}

// CalculatePendingSequences returns the highest sequence number considering both blockchain and pending pool
func CalculatePendingSequences(chain []*block.Block, pendingPool []block.Transaction) map[string]uint64 {
	sequences := CalculateSequences(chain)
	for _, tx := range pendingPool {
		if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
			if tx.Sequence > sequences[tx.Sender] {
				sequences[tx.Sender] = tx.Sequence
			}
		}
	}
	return sequences
}

func ValidateTransaction(tx block.Transaction, balances map[string]int64, sequences map[string]uint64) error {
	if tx.Amount <= 0 {
		return errors.New("Amount must be Greater than 0")
	}
	if tx.Amount > MaxTransactionAmount {
		return errors.New("Amount exceeds maximum allowed transaction size")
	}

	if tx.Recipient == "COINBASE" {
		return errors.New("Cannot send funds to COINBASE address.")
	}

	if !tx.Verify() {
		return errors.New("Invalid transaction signature")
	}

	if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
		// Sequence Validation (Replay Protection)
		expectedSeq := sequences[tx.Sender] + 1
		if tx.Sequence != expectedSeq {
			return fmt.Errorf("Invalid sequence number: expected %d, got %d", expectedSeq, tx.Sequence)
		}

		if balances[tx.Sender] < tx.Amount {
			return fmt.Errorf("Insufficent funds: need %d, but have %d", tx.Amount, balances[tx.Sender])
		}
	}
	return nil
}
