package node

import (
	"fmt"
	"sync"
	"testing"
	"valence/internal/block"
)

/*
TestMempoolAddAndHas verifies basic operations of the Mempool,
ensuring transactions can be added, duplicates are rejected,
and existence checks (Has) work correctly.
*/
func TestMempoolAddAndHas(t *testing.T) {
	m := NewMempool()

	tx1 := block.Transaction{ID: "tx1", Sender: "Alice"}
	tx2 := block.Transaction{ID: "tx2", Sender: "Bob"}

	if !m.Add(tx1) {
		t.Error("Expected true when adding new tx1")
	}

	if m.Add(tx1) {
		t.Error("Expected false when adding duplicate tx1")
	}

	m.Add(tx2)

	if !m.Has("tx1") || !m.Has("tx2") {
		t.Error("Mempool should have tx1 and tx2")
	}

	if m.Has("tx3") {
		t.Error("Mempool should not have tx3")
	}

	if m.Size() != 2 {
		t.Errorf("Expected size 2, got %d", m.Size())
	}
}

/*
TestMempoolRemove verifies that transactions can be successfully
removed from the pool using their IDs (e.g., after a block is mined).
*/
func TestMempoolRemove(t *testing.T) {
	m := NewMempool()
	m.Add(block.Transaction{ID: "tx1"})
	m.Add(block.Transaction{ID: "tx2"})
	m.Add(block.Transaction{ID: "tx3"})

	m.Remove([]string{"tx1", "tx3"})

	if m.Has("tx1") || m.Has("tx3") {
		t.Error("tx1 and tx3 should have been removed")
	}

	if !m.Has("tx2") {
		t.Error("tx2 should still be in the mempool")
	}

	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}
}

/*
TestMempoolGetAll verifies that GetAll returns a copy of transactions
sorted correctly by Sequence, then Sender.
*/
func TestMempoolGetAll(t *testing.T) {
	m := NewMempool()
	m.Add(block.Transaction{ID: "tx1", Sender: "B", Sequence: 2})
	m.Add(block.Transaction{ID: "tx2", Sender: "A", Sequence: 1})
	m.Add(block.Transaction{ID: "tx3", Sender: "A", Sequence: 2})

	txs := m.GetAll()

	if len(txs) != 3 {
		t.Fatalf("Expected 3 txs, got %d", len(txs))
	}

	if txs[0].ID != "tx2" || txs[1].ID != "tx3" || txs[2].ID != "tx1" {
		t.Errorf("Transactions are not sorted correctly: %v", txs)
	}
}

/*
TestMempoolConcurrency verifies that the Mempool is thread-safe
when accessed by multiple goroutines concurrently.
This test should be run with the -race flag.
*/
func TestMempoolConcurrency(t *testing.T) {
	m := NewMempool()
	var wg sync.WaitGroup

	// Add transactions concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			txID := fmt.Sprintf("tx%d", id)
			m.Add(block.Transaction{ID: txID})
			m.Has(txID)
			m.GetAll()
		}(i)
	}

	wg.Wait()

	if m.Size() != 100 {
		t.Errorf("Expected 100 transactions, got %d", m.Size())
	}

	// Remove transactions concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Remove([]string{fmt.Sprintf("tx%d", id)})
		}(i)
	}

	wg.Wait()

	if m.Size() != 50 {
		t.Errorf("Expected 50 transactions remaining, got %d", m.Size())
	}
}
