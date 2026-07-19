package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"encoding/json"
	"fmt"
	"sync"
)

type Chain struct {
	mu                 sync.RWMutex
	blocks             []*block.Block
	Difficulty         int                 `json:"difficulty"`
	pendingPool        []block.Transaction
	RetargetWindow     int                 `json:"retarget_window"`
	TargetBlockTimeSec int64               `json:"target_block_time_sec"`
	MaxDifficulty      int                 `json:"max_difficulty"`
	MinDifficulty      int                 `json:"min_difficulty"`
	InitialDifficulty  int                 `json:"initial_difficulty"`
	MaxTxPerBlock      int                 `json:"max_tx_per_block"` // Size limit per block
}

func (c *Chain) GetBlocks() []*block.Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	blocksCopy := make([]*block.Block, len(c.blocks))
	copy(blocksCopy, c.blocks)
	return blocksCopy
}

func (c *Chain) GetPendingPool() []block.Transaction {
	c.mu.RLock()
	defer c.mu.RUnlock()
	poolCopy := make([]block.Transaction, len(c.pendingPool))
	copy(poolCopy, c.pendingPool)
	return poolCopy
}

type chainAlias Chain

func (c *Chain) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(&struct {
		Blocks      []*block.Block      `json:"blocks"`
		PendingPool []block.Transaction `json:"pending_pool"`
		*chainAlias
	}{
		Blocks:      c.blocks,
		PendingPool: c.pendingPool,
		chainAlias:  (*chainAlias)(c),
	})
}

func (c *Chain) UnmarshalJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	aux := &struct {
		Blocks      []*block.Block      `json:"blocks"`
		PendingPool []block.Transaction `json:"pending_pool"`
		*chainAlias
	}{
		chainAlias: (*chainAlias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.blocks = aux.Blocks
	c.pendingPool = aux.PendingPool
	return nil
}

func NewChain(difficulty int, retargetWindow int, targetBlockTimeSec int64, minDifficulty int, maxDifficulty int) *Chain {
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
		pendingPool:        []block.Transaction{},
		RetargetWindow:     retargetWindow,
		TargetBlockTimeSec: targetBlockTimeSec,
		MaxDifficulty:      maxDifficulty,
		MinDifficulty:      minDifficulty,
		InitialDifficulty:  difficulty,
		MaxTxPerBlock:      10, // Default size limit per block
	}
}

func (c *Chain) AddTransaction(tx block.Transaction) error {
	if block.IsSystemAddress(tx.Sender) {
		return fmt.Errorf("transaction rejected: %s is reserved for system use only", tx.Sender)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	balances := ledger.CalculateAvailableBalances(c.blocks, c.pendingPool)
	sequences := ledger.CalculatePendingSequences(c.blocks, c.pendingPool)

	err := ledger.ValidateTransaction(tx, balances, sequences)
	if err != nil {
		return fmt.Errorf("transaction rejected: %w", err)
	}

	c.pendingPool = append(c.pendingPool, tx)
	return nil
}
