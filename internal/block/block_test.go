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
