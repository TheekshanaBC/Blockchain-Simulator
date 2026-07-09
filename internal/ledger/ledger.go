package ledger

import (
	"blockchain-simulator/internal/block"
	"errors"
	"fmt"
)

func CalculateBalances(chain []*block.Block) map[string]float64 {
	balances := make(map[string]float64)

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

func ValidateTransaction(tx block.Transaction, balances map[string]float64) error {
	if tx.Amount <= 0 {
		return errors.New("Amount must be Greater than 0!")
	}

	if tx.Sender != "FAUCET" && tx.Sender != "COINBASE" {
		if balances[tx.Sender] < tx.Amount {
			return fmt.Errorf("Insufficent funds!: need %f, but have %f", tx.Amount, balances[tx.Sender])
		}
	}
	return nil
}
