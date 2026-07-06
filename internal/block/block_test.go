package block

import "testing"

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
		Transactions: []string{"Alice pays Bob 10 USD"},
		PrevHash:     GenesisPrevHash,
		Nonce:        12345,
	}

	hash1 := block.CalculateHash()

	block.Transactions = []string{"Alice pays Bob 100 USD"}
	hash2 := block.CalculateHash()

	block.Transactions = []string{"Alice pays Bob 10 USD"}
	hash3 := block.CalculateHash()

	if hash1 == hash2 {
		t.Errorf("Hashes should be different for different Transactions")
	}

	if hash1 != hash3 {
		t.Errorf("Hashes should be same for same Transactions")
	}
}
