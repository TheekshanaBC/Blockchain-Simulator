package chain

import (
	"math/big"
	"valence/internal/block"
)

// BlockWork returns the expected number of hashes required to find a block at the given difficulty.
// Since the target requires leading zero hex characters, each difficulty level increases work by a factor of 16.
func BlockWork(difficulty int) *big.Int {
	work := big.NewInt(16)
	// Work is 16^difficulty
	return new(big.Int).Exp(work, big.NewInt(int64(difficulty)), nil)
}

// CumulativeWork iterates over a slice of blocks and returns the sum of their individual works.
func CumulativeWork(blocks []*block.Block) *big.Int {
	totalWork := big.NewInt(0)
	for _, b := range blocks {
		totalWork.Add(totalWork, BlockWork(b.Header.Difficulty))
	}
	return totalWork
}
