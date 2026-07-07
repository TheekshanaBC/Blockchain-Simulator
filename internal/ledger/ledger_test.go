package ledger

import (
	"testing"

	"blockchain-simulator/internal/block"
)

func TestValidateTransaction(t *testing.T) {
	balances := map[string]float64{
		"Alice": 100.0,
		"Bob":   50.0,
	}

	// Overspending
	overTx := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 150}
	err := ValidateTransaction(overTx, balances)
	if err == nil {
		t.Errorf("Expected an error for overspending, but got nil")
	}

	// Zero amount transaction
	zeroTx := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 0}
	err = ValidateTransaction(zeroTx, balances)
	if err == nil {
		t.Errorf("Expected an error for zero amount transaction, but got nil")
	}

	// Good Transaction
	goodTx := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 20}
	err = ValidateTransaction(goodTx, balances)
	if err != nil {
		t.Errorf("Did not expect an error for valid transaction, but got: %v", err)
	}

}
