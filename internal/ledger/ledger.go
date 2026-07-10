package ledger

import (
	"blockchain-simulator/internal/block"
	"errors"
	"fmt"
)

func CalculateBalances(chain []*block.Block) map[string]uint64 {
	balances := make(map[string]uint64)

	for _, b := range chain {
		for _, tx := range b.Transactions {
			if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
				balances[tx.Sender] -= tx.Amount
			}
			balances[tx.Recipient] += tx.Amount
		}
	}
	return balances
}

// CalculateAvailableBalances returns the balance available to spend (ledger minus pending outbounds)
func CalculateAvailableBalances(chain []*block.Block, pendingPool []block.Transaction) map[string]uint64 {
	balances := CalculateBalances(chain)

	// deduct the pending pool transactions to prevent double spending
	for _, tx := range pendingPool {
		if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
			balances[tx.Sender] -= tx.Amount
		}
	}
	return balances
}

func ValidateTransaction(tx block.Transaction, balances map[string]uint64) error {
	if tx.Amount <= 0 {
		return errors.New("Amount must be Greater than 0")
	}

	if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
		if balances[tx.Sender] < tx.Amount {
			return fmt.Errorf("Insufficent funds: need %d, but have %d", tx.Amount, balances[tx.Sender])
		}
	}
	return nil
}
