package chain

import (
	"valence/internal/block"
)

// recalculate difficulty every N blocks
func expectedDifficultyAfterWindow(blocks []*block.Block, nextHeight, N int, targetBlockTime int64, prevDifficulty, min, max int) int {
	if nextHeight <= 1 || (nextHeight-1)%N != 0 {
		return prevDifficulty
	}

	lastBlock := blocks[nextHeight-1]

	windowIndex := (nextHeight - 1) / N
	var firstBlockIndex int
	var expectedIntervals int

	if windowIndex == 1 {
		firstBlockIndex = 1
		expectedIntervals = N - 1
	} else {
		firstBlockIndex = nextHeight - 1 - N
		expectedIntervals = N
	}

	firstBlock := blocks[firstBlockIndex]
	actual := lastBlock.Header.Timestamp - firstBlock.Header.Timestamp
	expected := targetBlockTime * 1_000_000_000 * int64(expectedIntervals)

	return adjustDifficulty(prevDifficulty, actual, expected, min, max)
}

// adjust difficulty based on actual time taken to mine previous N blocks
func adjustDifficulty(current int, actual, expected int64, min, max int) int {
	if actual < expected/2 {
		current++
	} else if actual > expected*2 {
		current--
	}
	if current < min {
		current = min
	}
	if current > max {
		current = max
	}
	return current
}


