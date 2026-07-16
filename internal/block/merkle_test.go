package block

import (
	"fmt"
	"testing"
)

/*
TestCalculateMerkleRoot_Empty verifies that calculating a Merkle Root for
an empty list of transactions returns an empty string.
*/
func TestCalculateMerkleRoot_Empty(t *testing.T) {
	var txs []Transaction
	root := CalculateMerkleRoot(txs)
	if root != "" {
		t.Errorf("Expected empty string for empty transactions, got %s", root)
	}
}

/*
TestCalculateMerkleRoot_Single tests that when there is only one transaction,
the Merkle Root is simply the double SHA-256 hash of that single transaction.
*/
func TestCalculateMerkleRoot_Single(t *testing.T) {
	txs := []Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10},
	}
	root := CalculateMerkleRoot(txs)

	// Since there's only 1 tx, it just doubleSHA256s the tx record once
	record := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[0].Sender), txs[0].Sender, len(txs[0].Recipient), txs[0].Recipient, txs[0].Amount, len(txs[0].PublicKey), txs[0].PublicKey, len(txs[0].Signature), txs[0].Signature)
	expectedHash := doubleSHA256(record)

	if root != expectedHash {
		t.Errorf("Expected %s, got %s", expectedHash, root)
	}
}

/*
TestCalculateMerkleRoot_Even verifies the correct calculation of the Merkle Root
for an even number of transactions (e.g., 2), ensuring their individual hashes
are concatenated and hashed together.
*/
func TestCalculateMerkleRoot_Even(t *testing.T) {
	txs := []Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10},
		{Sender: "Bob", Recipient: "Charlie", Amount: 5},
	}
	root := CalculateMerkleRoot(txs)

	record1 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[0].Sender), txs[0].Sender, len(txs[0].Recipient), txs[0].Recipient, txs[0].Amount, len(txs[0].PublicKey), txs[0].PublicKey, len(txs[0].Signature), txs[0].Signature)
	record2 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[1].Sender), txs[1].Sender, len(txs[1].Recipient), txs[1].Recipient, txs[1].Amount, len(txs[1].PublicKey), txs[1].PublicKey, len(txs[1].Signature), txs[1].Signature)

	hash1 := doubleSHA256(record1)
	hash2 := doubleSHA256(record2)

	expected := doubleSHA256("\x01" + hash1 + hash2)
	if root != expected {
		t.Errorf("Expected %s, got %s", expected, root)
	}
}

/*
TestCalculateMerkleRoot_Odd tests the Merkle Root calculation for an odd number
of transactions (e.g., 3). It ensures that the last transaction's hash is duplicated
to pair it up before the final hashing step.
*/
func TestCalculateMerkleRoot_Odd(t *testing.T) {
	txs := []Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10},
		{Sender: "Bob", Recipient: "Charlie", Amount: 5},
		{Sender: "Charlie", Recipient: "Dave", Amount: 2},
	}
	root := CalculateMerkleRoot(txs)

	record1 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[0].Sender), txs[0].Sender, len(txs[0].Recipient), txs[0].Recipient, txs[0].Amount, len(txs[0].PublicKey), txs[0].PublicKey, len(txs[0].Signature), txs[0].Signature)
	record2 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[1].Sender), txs[1].Sender, len(txs[1].Recipient), txs[1].Recipient, txs[1].Amount, len(txs[1].PublicKey), txs[1].PublicKey, len(txs[1].Signature), txs[1].Signature)
	record3 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[2].Sender), txs[2].Sender, len(txs[2].Recipient), txs[2].Recipient, txs[2].Amount, len(txs[2].PublicKey), txs[2].PublicKey, len(txs[2].Signature), txs[2].Signature)

	hash1 := doubleSHA256(record1)
	hash2 := doubleSHA256(record2)
	hash3 := doubleSHA256(record3) // The odd one out

	// Level 1
	hash12 := doubleSHA256("\x01" + hash1 + hash2)
	hash33 := hash3 // Promoted unchanged (CVE-2012-2459 fix)

	// Level 2 (Root)
	expectedRoot := doubleSHA256("\x01" + hash12 + hash33)

	if root != expectedRoot {
		t.Errorf("Expected %s, got %s", expectedRoot, root)
	}
}

/*
TestDoubleSHA256 verifies that the internal hashing function produces
deterministic, consistent 64-character hex strings for identical inputs,
and different outputs for different inputs.
*/
func TestDoubleSHA256(t *testing.T) {
	input1 := "hello"
	input2 := "world"

	hash1 := doubleSHA256(input1)
	hash2 := doubleSHA256(input1)
	hash3 := doubleSHA256(input2)

	if hash1 != hash2 {
		t.Errorf("Expected deterministic hashes, but got different results for the same input")
	}

	if hash1 == hash3 {
		t.Errorf("Expected different hashes for different inputs")
	}

	if len(hash1) != 64 {
		t.Errorf("Expected 64-character hex string, got length %d", len(hash1))
	}
}

/*
TestCalculateMerkleRoot_Deep tests a deeper Merkle tree (e.g., 5 transactions)
to ensure the loop correctly collapses multiple levels (5 -> 3 -> 2 -> 1).
*/
func TestCalculateMerkleRoot_Deep(t *testing.T) {
	txs := []Transaction{
		{Sender: "A", Recipient: "B", Amount: 1},
		{Sender: "B", Recipient: "C", Amount: 2},
		{Sender: "C", Recipient: "D", Amount: 3},
		{Sender: "D", Recipient: "E", Amount: 4},
		{Sender: "E", Recipient: "F", Amount: 5},
	}
	root := CalculateMerkleRoot(txs)

	record1 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[0].Sender), txs[0].Sender, len(txs[0].Recipient), txs[0].Recipient, txs[0].Amount, len(txs[0].PublicKey), txs[0].PublicKey, len(txs[0].Signature), txs[0].Signature)
	record2 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[1].Sender), txs[1].Sender, len(txs[1].Recipient), txs[1].Recipient, txs[1].Amount, len(txs[1].PublicKey), txs[1].PublicKey, len(txs[1].Signature), txs[1].Signature)
	record3 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[2].Sender), txs[2].Sender, len(txs[2].Recipient), txs[2].Recipient, txs[2].Amount, len(txs[2].PublicKey), txs[2].PublicKey, len(txs[2].Signature), txs[2].Signature)
	record4 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[3].Sender), txs[3].Sender, len(txs[3].Recipient), txs[3].Recipient, txs[3].Amount, len(txs[3].PublicKey), txs[3].PublicKey, len(txs[3].Signature), txs[3].Signature)
	record5 := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x", len(txs[4].Sender), txs[4].Sender, len(txs[4].Recipient), txs[4].Recipient, txs[4].Amount, len(txs[4].PublicKey), txs[4].PublicKey, len(txs[4].Signature), txs[4].Signature)

	h1 := doubleSHA256(record1)
	h2 := doubleSHA256(record2)
	h3 := doubleSHA256(record3)
	h4 := doubleSHA256(record4)
	h5 := doubleSHA256(record5)

	// Level 1: 5 nodes -> 3 nodes
	h12 := doubleSHA256("\x01" + h1 + h2)
	h34 := doubleSHA256("\x01" + h3 + h4)
	h55 := h5 // Promoted unchanged

	// Level 2: 3 nodes -> 2 nodes
	h1234 := doubleSHA256("\x01" + h12 + h34)
	h5555 := h55 // Promoted unchanged

	// Level 3: 2 nodes -> 1 root
	expectedRoot := doubleSHA256("\x01" + h1234 + h5555)

	if root != expectedRoot {
		t.Errorf("Expected %s, got %s", expectedRoot, root)
	}
}
