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

/*
TestBlockWork_NegativeDifficulty verifies that BlockWork clamps negative difficulty
to 0 (returns 1 = 16^0) instead of panicking or producing garbage output.
This guards against corrupt block headers reaching the work computation.
*/
func TestBlockWork_NegativeDifficulty(t *testing.T) {
	work := BlockWork(-1)
	expected := big.NewInt(1) // 16^0
	if work.Cmp(expected) != 0 {
		t.Errorf("BlockWork(-1) = %s, want 1 (16^0)", work.String())
	}

	work = BlockWork(-1000000)
	if work.Cmp(expected) != 0 {
		t.Errorf("BlockWork(-1000000) = %s, want 1 (16^0)", work.String())
	}
}

/*
TestBlockWork_HardCeiling verifies that BlockWork clamps difficulty above 64
to exactly 64, preventing attacker-controlled exponents from allocating
hundreds of megabytes of memory via 16^(attacker_value).
*/
func TestBlockWork_HardCeiling(t *testing.T) {
	attackerDifficulties := []int{65, 1000, 1_000_000, 1_000_000_000}
	expected := BlockWork(64) // the ceiling value

	for _, d := range attackerDifficulties {
		work := BlockWork(d)
		if work.Cmp(expected) != 0 {
			t.Errorf("BlockWork(%d) should be clamped to BlockWork(64)=%s, got %s", d, expected.String(), work.String())
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
