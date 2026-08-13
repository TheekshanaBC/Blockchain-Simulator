package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"valence/internal/block"
	"valence/internal/ledger"
)

type Chain struct {
	mu         sync.RWMutex
	blocks     []*block.Block
	Difficulty int `json:"difficulty"`

	RetargetWindow     int   `json:"retarget_window"`
	TargetBlockTimeSec int64 `json:"target_block_time_sec"`
	MaxDifficulty      int   `json:"max_difficulty"`
	MinDifficulty      int   `json:"min_difficulty"`
	InitialDifficulty  int   `json:"initial_difficulty"`
	MaxTxPerBlock      int   `json:"max_tx_per_block"` // Size limit per block
}

func (c *Chain) GetBlocks() []*block.Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	blocksCopy := make([]*block.Block, len(c.blocks))
	copy(blocksCopy, c.blocks)
	return blocksCopy
}

func (c *Chain) GetLastBlock() *block.Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.blocks) == 0 {
		return nil
	}
	return c.blocks[len(c.blocks)-1]
}

func (c *Chain) Height() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.blocks) - 1
}

// AddBlock manually appends a block to the chain, used for gossip/syncing.
// It validates the block against the current chain before appending.
func (c *Chain) AddBlock(b block.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.blocks) == 0 {
		return fmt.Errorf("chain is empty")
	}

	previousBlock := c.blocks[len(c.blocks)-1]

	if b.Height != previousBlock.Height+1 {
		return fmt.Errorf("block height mismatch: expected %d, got %d", previousBlock.Height+1, b.Height)
	}

	expectedDifficulty := expectedDifficultyAfterWindow(c.blocks, b.Height, c.RetargetWindow, c.TargetBlockTimeSec, c.Difficulty, c.MinDifficulty, c.MaxDifficulty)

	res := validateBlockAgainstPrevious(&b, previousBlock, expectedDifficulty)
	if !res.IsValid {
		return errors.New(res.Reason)
	}

	balances := ledger.CalculateAvailableBalances(c.blocks, []block.Transaction{})
	sequences := ledger.CalculatePendingSequences(c.blocks, []block.Transaction{})
	
	res = validateBlockTransactions(&b, balances, sequences, c.MaxTxPerBlock)
	if !res.IsValid {
		return errors.New(res.Reason)
	}

	c.blocks = append(c.blocks, &b)
	c.Difficulty = expectedDifficulty
	return nil
}

// SwitchToChain safely replaces the current chain with a new one if it is valid and longer.
// It returns a slice of orphaned transactions that should be returned to the mempool.
func (c *Chain) SwitchToChain(newBlocks []*block.Block) ([]block.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Check cumulative work
	currentWork := CumulativeWork(c.blocks)
	newWork := CumulativeWork(newBlocks)
	if newWork.Cmp(currentWork) <= 0 {
		return nil, fmt.Errorf("candidate chain does not have more cumulative work than current chain")
	}

	// 2. Validate the entire new chain from genesis
	result := ValidateBlockSlice(newBlocks, c.InitialDifficulty, c.RetargetWindow, c.TargetBlockTimeSec, c.MinDifficulty, c.MaxDifficulty, c.MaxTxPerBlock)
	if !result.IsValid {
		return nil, fmt.Errorf("candidate chain invalid: %s", result.Reason)
	}

	// 3. Find orphaned blocks and transactions
	orphanedBlocks := findOrphanedBlocks(c.blocks, newBlocks)
	orphanedTxs := collectOrphanedTxs(orphanedBlocks, newBlocks)

	// 4. Switch to new chain
	c.blocks = newBlocks
	c.Difficulty = newBlocks[len(newBlocks)-1].Header.Difficulty

	return orphanedTxs, nil
}


type chainAlias Chain

func (c *Chain) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(&struct {
		Blocks      []*block.Block      `json:"blocks"`
		*chainAlias
	}{
		Blocks:      c.blocks,
		chainAlias:  (*chainAlias)(c),
	})
}

func (c *Chain) UnmarshalJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	aux := &struct {
		Blocks      []*block.Block      `json:"blocks"`
		*chainAlias
	}{
		chainAlias: (*chainAlias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.blocks = aux.Blocks
	return nil
}

func NewChain(difficulty int, retargetWindow int, targetBlockTimeSec int64, minDifficulty int, maxDifficulty int, maxTxPerBlock int) *Chain {
	if retargetWindow < 2 {
		retargetWindow = 2
	}
	if difficulty < minDifficulty {
		difficulty = minDifficulty
	}
	if difficulty > maxDifficulty {
		difficulty = maxDifficulty
	}
	genesis := block.NewGenesisBlock()
	return &Chain{
		blocks:             []*block.Block{genesis},
		Difficulty:         difficulty,
		RetargetWindow:     retargetWindow,
		TargetBlockTimeSec: targetBlockTimeSec,
		MaxDifficulty:      maxDifficulty,
		MinDifficulty:      minDifficulty,
		InitialDifficulty:  difficulty,
		MaxTxPerBlock:      maxTxPerBlock,
	}
}


