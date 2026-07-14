package block

import (
	"fmt"
	"testing"
)

func TestCalculateMerkleRoot_Empty(t *testing.T) {
	var txs []Transaction
	root := CalculateMerkleRoot(txs)
	if root != "" {
		t.Errorf("Expected empty string for empty transactions, got %s", root)
	}
}

func TestCalculateMerkleRoot_Single(t *testing.T) {
	txs := []Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10},
	}
	root := CalculateMerkleRoot(txs)

	// Since there's only 1 tx, it just doubleSHA256s the tx record once
	record := fmt.Sprintf("%s|%s|%d|%d", txs[0].Sender, txs[0].Recipient, txs[0].Amount, txs[0].ExtraNonce)
	expectedHash := doubleSHA256(record)

	if root != expectedHash {
		t.Errorf("Expected %s, got %s", expectedHash, root)
	}
}

func TestCalculateMerkleRoot_Even(t *testing.T) {
	txs := []Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10},
		{Sender: "Bob", Recipient: "Charlie", Amount: 5},
	}
	root := CalculateMerkleRoot(txs)

	record1 := fmt.Sprintf("%s|%s|%d|%d", txs[0].Sender, txs[0].Recipient, txs[0].Amount, txs[0].ExtraNonce)
	record2 := fmt.Sprintf("%s|%s|%d|%d", txs[1].Sender, txs[1].Recipient, txs[1].Amount, txs[1].ExtraNonce)

	hash1 := doubleSHA256(record1)
	hash2 := doubleSHA256(record2)

	expectedRoot := doubleSHA256(hash1 + hash2)

	if root != expectedRoot {
		t.Errorf("Expected %s, got %s", expectedRoot, root)
	}
}

func TestCalculateMerkleRoot_Odd(t *testing.T) {
	txs := []Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10},
		{Sender: "Bob", Recipient: "Charlie", Amount: 5},
		{Sender: "Charlie", Recipient: "Dave", Amount: 2},
	}
	root := CalculateMerkleRoot(txs)

	record1 := fmt.Sprintf("%s|%s|%d|%d", txs[0].Sender, txs[0].Recipient, txs[0].Amount, txs[0].ExtraNonce)
	record2 := fmt.Sprintf("%s|%s|%d|%d", txs[1].Sender, txs[1].Recipient, txs[1].Amount, txs[1].ExtraNonce)
	record3 := fmt.Sprintf("%s|%s|%d|%d", txs[2].Sender, txs[2].Recipient, txs[2].Amount, txs[2].ExtraNonce)

	hash1 := doubleSHA256(record1)
	hash2 := doubleSHA256(record2)
	hash3 := doubleSHA256(record3) // The odd one out

	// Level 1
	hash12 := doubleSHA256(hash1 + hash2)
	hash33 := doubleSHA256(hash3 + hash3) // Duplicated

	// Level 2 (Root)
	expectedRoot := doubleSHA256(hash12 + hash33)

	if root != expectedRoot {
		t.Errorf("Expected %s, got %s", expectedRoot, root)
	}
}
