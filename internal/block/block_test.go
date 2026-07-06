package block

import (
	"strings"
	"testing"
)

func TestNewGenesisBlock(t *testing.T) {
	block := NewGenesisBlock()

	if block.Height != 0 {
		t.Errorf("Expected Height 0, got %d", block.Height)
	}

	if block.PrevHash != GenesisPrevHash {
		t.Errorf("Expected PrevHash %s, got %s", GenesisPrevHash, block.PrevHash)
	}
}

func TestCalculateHash(t *testing.T) {
	block := &Block{
		Height:       1,
		Timestamp:    1720211552,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}},
		PrevHash:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Nonce:        12345,
	}

	hash1 := block.CalculateHash()

	block.Transactions = []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 100}}
	hash2 := block.CalculateHash()

	block.Transactions = []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}}
	hash3 := block.CalculateHash()

	if hash1 == hash2 {
		t.Errorf("Hashes should be different for different Transactions")
	}

	if hash1 != hash3 {
		t.Errorf("Hashes should be same for same Transactions")
	}
}

func TestMine(t *testing.T) {
	block := &Block{
		Height:       1,
		Timestamp:    1627890123,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 100}},
		PrevHash:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Nonce:        0,
	}

	difficulty := 4
	block.Mine(difficulty)

	// check the hash starts with {difficulty} zeros after mining
	target := strings.Repeat("0", difficulty) // target = 0000
	if !strings.HasPrefix(block.Hash, target) {
		t.Errorf("Mined hash does not meet difficulty target. Hash: %s", block.Hash)
	}

	// now nonce is set to a value where blocks hash is starting with {difficulty} zeros.
	// recalculate the hash with the found Nonce amd check is it same with the Mined value
	expectedHash := block.CalculateHash()
	if block.Hash != expectedHash {
		t.Errorf("Hash does not match recalculated hash with found nonce")
	}

}
