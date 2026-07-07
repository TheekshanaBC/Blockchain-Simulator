package chain

import (
	"blockchain-simulator/internal/block"
	"testing"
)

func TestValidationAndTamperDetection(t *testing.T) {
	myChain := NewChain(2)

	tx1 := block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100}
	myChain.AddTransaction(tx1)
	myChain.MinePendingTransactions()

	tx2 := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 20}
	myChain.AddTransaction(tx2)
	myChain.MinePendingTransactions()

	// Check the honest chain
	result := myChain.Validate()
	if !result.IsValid {
		t.Fatalf("Expected chain to be valid, but failed at height %d: %s", result.FailedAtHeight, result.Reason)
	}

	// Tampering
	myChain.Blocks[1].Transactions[0].Amount = 5000

	tamperedResult := myChain.Validate()

	if tamperedResult.IsValid {
		t.Fatalf("Expected chain to be INVALID after tampering, but it passed!")
	}

	if tamperedResult.FailedAtHeight != 1 {
		t.Errorf("Expected failure at height 1, but failed at %d", tamperedResult.FailedAtHeight)
	}

	t.Logf("Tamper caught successfully! Reason: %s", tamperedResult.Reason)
}
