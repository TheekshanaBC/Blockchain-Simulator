package chain

import "valence/internal/block"

// findOrphanedBlocks identifies blocks that are in the old chain but not in the new chain.
func findOrphanedBlocks(oldBlocks, newBlocks []*block.Block) []*block.Block {
	newHashes := make(map[string]bool)
	for _, b := range newBlocks {
		newHashes[b.Hash] = true
	}

	var orphaned []*block.Block
	for _, b := range oldBlocks {
		if !newHashes[b.Hash] {
			orphaned = append(orphaned, b)
		}
	}
	return orphaned
}

// collectOrphanedTxs extracts transactions from orphaned blocks that haven't been
// mined in the new chain.
func collectOrphanedTxs(orphanedBlocks, newBlocks []*block.Block) []block.Transaction {
	newTxs := make(map[string]bool)
	for _, b := range newBlocks {
		for _, tx := range b.Transactions {
			newTxs[tx.ID] = true
		}
	}

	var orphanedTxs []block.Transaction
	for _, b := range orphanedBlocks {
		for _, tx := range b.Transactions {
			// Skip coinbase transactions, they are block-specific
			if tx.Sender == block.SystemAddressCoinbase {
				continue
			}
			if !newTxs[tx.ID] {
				orphanedTxs = append(orphanedTxs, tx)
			}
		}
	}
	return orphanedTxs
}
