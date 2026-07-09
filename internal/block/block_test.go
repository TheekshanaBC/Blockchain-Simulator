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

	if block.Header.PrevHash != GenesisPrevHash {
		t.Errorf("Expected PrevHash %s, got %s", GenesisPrevHash, block.Header.PrevHash)
	}
}

func TestCalculateHash(t *testing.T) {
	block := &Block{
		Header: BlockHeader{
			Timestamp: 1720211552,
			PrevHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Nonce:     12345,
		},
		Height:       1,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}},
	}

	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	hash1 := block.CalculateHash()

	block.Transactions = []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 100}}
	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	hash2 := block.CalculateHash()

	block.Transactions = []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}}
	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	hash3 := block.CalculateHash()

	if hash1 == hash2 {
		t.Errorf("Hashes should be different for different Merkle Roots")
	}

	if hash1 != hash3 {
		t.Errorf("Hashes should be same for same Merkle Roots")
	}
}

func TestMine(t *testing.T) {
	block := &Block{
		Header: BlockHeader{
			PrevHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Timestamp: 1627890123,
			Nonce:     0,
		},
		Height:       1,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 100}},
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
