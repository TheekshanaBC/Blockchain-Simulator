package node

import (
	"fmt"
	"sort"
	"sync"
	"valence/internal/block"
	"valence/internal/ledger"
)

type Mempool struct {
	mu      sync.RWMutex
	txs     map[string]block.Transaction // key = tx.ID
	maxSize int
}

func NewMempool(maxSize int) *Mempool {
	if maxSize <= 0 {
		maxSize = 5000 // default max size
	}
	return &Mempool{
		txs:     make(map[string]block.Transaction),
		maxSize: maxSize,
	}
}

// Add adds a transaction to the mempool. Returns false if duplicate.
func (m *Mempool) Add(tx block.Transaction) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.txs[tx.ID]; exists {
		return false
	}
	if len(m.txs) >= m.maxSize {
		return false // Silently reject if mempool is full (or should we return error? For gossip Add, returning false is fine)
	}
	m.txs[tx.ID] = tx
	return true
}

// ValidateAndAdd atomically validates the transaction against the current ledger state and mempool,
// and adds it if valid. This prevents TOCTOU race conditions where multiple transactions with the
// same sequence could be validated concurrently and then both added.
func (m *Mempool) ValidateAndAdd(tx block.Transaction, chainBlocks []*block.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.txs[tx.ID]; exists {
		return fmt.Errorf("transaction already exists")
	}
	if len(m.txs) >= m.maxSize {
		return fmt.Errorf("mempool is full")
	}

	txList := make([]block.Transaction, 0, len(m.txs))
	for _, t := range m.txs {
		txList = append(txList, t)
	}

	if err := ledger.ValidateTransactions([]block.Transaction{tx}, chainBlocks, txList); err != nil {
		return err
	}

	m.txs[tx.ID] = tx
	return nil
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

