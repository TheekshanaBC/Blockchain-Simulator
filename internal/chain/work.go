package chain

import (
	"math/big"
	"valence/internal/block"
)

// BlockWork returns the expected number of hashes required to find a block at the given difficulty.
// Since the target requires leading zero hex characters, each difficulty level increases work by a factor of 16.
func BlockWork(difficulty int) *big.Int {
	// Clamp difficulty before exponentiation (defense-in-depth).
	// Negative values can come from corrupt blocks; absurdly large values could
	// allocate gigabytes even for a validated chain if MaxDifficulty is ever
	// raised. 16^64 already represents ~10^77 expected hashes — far beyond any
	// realistic need — so 64 is a safe hard ceiling with no caller-visible impact.
	if difficulty < 0 {
		difficulty = 0
	}
	if difficulty > 64 {
		difficulty = 64
	}
	return new(big.Int).Exp(big.NewInt(16), big.NewInt(int64(difficulty)), nil)
}

// CumulativeWork iterates over a slice of blocks and returns the sum of their individual works.
func CumulativeWork(blocks []*block.Block) *big.Int {
	totalWork := big.NewInt(0)
	for _, b := range blocks {
		totalWork.Add(totalWork, BlockWork(b.Header.Difficulty))
	}
	return totalWork
}
