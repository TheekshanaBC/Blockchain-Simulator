package node

import (
	"sort"
	"sync"
	"valence/internal/block"
)

type Mempool struct {
	mu  sync.RWMutex
	txs map[string]block.Transaction // key = tx.ID
}

func NewMempool() *Mempool {
	return &Mempool{
		txs: make(map[string]block.Transaction),
	}
}

// Add adds a transaction to the mempool. Returns false if duplicate.
func (m *Mempool) Add(tx block.Transaction) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.txs[tx.ID]; exists {
		return false
	}
	m.txs[tx.ID] = tx
	return true
}

// Remove removes transactions from the mempool by their IDs.
// Called after a block is accepted to clear confirmed transactions.
func (m *Mempool) Remove(txIDs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range txIDs {
		delete(m.txs, id)
	}
}

// Has checks if a transaction ID exists in the mempool.
// Useful for gossip deduplication.
func (m *Mempool) Has(txID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.txs[txID]
	return exists
}

// GetAll returns a copy of all pending transactions sorted by sequence.
func (m *Mempool) GetAll() []block.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txList := make([]block.Transaction, 0, len(m.txs))
	for _, tx := range m.txs {
		txList = append(txList, tx)
	}

	// Sort transactions by Sequence (and Sender/ID as secondary/tertiary to keep deterministic)
	sort.Slice(txList, func(i, j int) bool {
		if txList[i].Sequence == txList[j].Sequence {
			if txList[i].Sender == txList[j].Sender {
				return txList[i].ID < txList[j].ID
			}
			return txList[i].Sender < txList[j].Sender
		}
		return txList[i].Sequence < txList[j].Sequence
	})

	return txList
}

// Size returns the number of pending transactions.
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.txs)
}

// Clear removes all transactions from the mempool.
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.txs = make(map[string]block.Transaction)
}
