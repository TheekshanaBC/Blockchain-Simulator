package chain

import (
	"math/big"
	"testing"
	"valence/internal/block"
)

func TestBlockWork(t *testing.T) {
	tests := []struct {
		difficulty int
		expected   string
	}{
		{0, "1"},
		{1, "16"},
		{2, "256"},
		{3, "4096"},
		{5, "1048576"},
	}

	for _, tt := range tests {
		work := BlockWork(tt.difficulty)
		if work.String() != tt.expected {
			t.Errorf("BlockWork(%d) = %s, want %s", tt.difficulty, work.String(), tt.expected)
		}
	}
}

func TestCumulativeWork(t *testing.T) {
	blocks := []*block.Block{
		{Header: block.BlockHeader{Difficulty: 1}}, // work: 16
		{Header: block.BlockHeader{Difficulty: 2}}, // work: 256
		{Header: block.BlockHeader{Difficulty: 3}}, // work: 4096
	}

	total := CumulativeWork(blocks)
	expected := big.NewInt(16 + 256 + 4096)

	if total.Cmp(expected) != 0 {
		t.Errorf("CumulativeWork = %s, want %s", total.String(), expected.String())
	}
}
