package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"fmt"
)

type Chain struct {
	Blocks             []*block.Block      `json:"blocks"`
	Difficulty         int                 `json:"difficulty"`
	PendingPool        []block.Transaction `json:"pending_pool"`
	RetargetWindow     int                 `json:"retarget_window"`
	TargetBlockTimeSec int64               `json:"target_block_time_sec"`
	MaxDifficulty      int                 `json:"max_difficulty"`
	MinDifficulty      int                 `json:"min_difficulty"`
}

func NewChain(difficulty int, retargetWindow int, targetBlockTimeSec int64, minDifficulty int, maxDifficulty int) *Chain {
	if retargetWindow < 2 {
		retargetWindow = 2
	}
	genesis := block.NewGenesisBlock()
	return &Chain{
		Blocks:             []*block.Block{genesis},
		Difficulty:         difficulty,
		PendingPool:        []block.Transaction{},
		RetargetWindow:     retargetWindow,
		TargetBlockTimeSec: targetBlockTimeSec,
		MaxDifficulty:      maxDifficulty,
		MinDifficulty:      minDifficulty,
	}
}

func (c *Chain) AddTransaction(tx block.Transaction) error {
	if block.IsSystemAddress(tx.Sender) {
		return fmt.Errorf("Transaction Rejected: %s is reserved for system use only", tx.Sender)
	}

	balances := ledger.CalculateAvailableBalances(c.Blocks, c.PendingPool)
	sequences := ledger.CalculatePendingSequences(c.Blocks, c.PendingPool)

	err := ledger.ValidateTransaction(tx, balances, sequences)
	if err != nil {
		return fmt.Errorf("Transaction Rejected: %w", err)
	}

	c.PendingPool = append(c.PendingPool, tx)
	return nil
}
