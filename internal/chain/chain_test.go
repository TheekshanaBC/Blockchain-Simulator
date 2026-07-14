package chain

import (
	"blockchain-simulator/internal/block"
	"encoding/json"
	"strings"
	"testing"
)

/*
TestValidationAndTamperDetection verifies that the blockchain can correctly
identify when data has been altered. It first checks if a sequence of valid
mined blocks passes validation. Then, it intentionally modifies a transaction
in an already mined block and asserts that the blockchain becomes invalid
and successfully pinpoints the exact block height where the tampering occurred.
*/
func TestValidationAndTamperDetection(t *testing.T) {
	myChain := NewChain(2)

	tx1 := block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100}
	myChain.AddTransaction(tx1)
	myChain.MinePendingTransactions()

	tx2 := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 20}
	myChain.AddTransaction(tx2)
	myChain.MinePendingTransactions()

	// Check the honest chain
	result := myChain.Validate()
	if !result.IsValid {
		t.Fatalf("Expected chain to be valid, but failed at height %d: %s", result.FailedAtHeight, result.Reason)
	}

	// Tampering
	myChain.Blocks[1].Transactions[0].Amount = 5000

	tamperedResult := myChain.Validate()

	if tamperedResult.IsValid {
		t.Fatalf("Expected chain to be INVALID after tampering, but it passed!")
	}

	if tamperedResult.FailedAtHeight != 1 {
		t.Errorf("Expected failure at height 1, but failed at %d", tamperedResult.FailedAtHeight)
	}

	t.Logf("Tamper caught successfully! Reason: %s", tamperedResult.Reason)
}

/*
TestNewChain verifies that a new blockchain is initialized correctly with
the given difficulty, an empty pending pool, and exactly one valid Genesis block.
*/
func TestNewChain(t *testing.T) {
	difficulty := 2
	myChain := NewChain(difficulty)

	if myChain.Difficulty != difficulty {
		t.Errorf("Expected difficulty %d, got %d", difficulty, myChain.Difficulty)
	}

	if len(myChain.Blocks) != 1 {
		t.Fatalf("Expected exactly 1 block (Genesis), got %d", len(myChain.Blocks))
	}

	if len(myChain.PendingPool) != 0 {
		t.Errorf("Expected pending pool to be empty, got %d", len(myChain.PendingPool))
	}

	result := myChain.Validate()
	if !result.IsValid {
		t.Errorf("Expected new chain to be valid, but got error: %s", result.Reason)
	}
}

/*
TestAddTransaction verifies the logic for adding new transactions to the pending pool.
It checks for successful additions, rejection of reserved COINBASE sender, and
rejection of invalid transactions (like overspending).
*/
func TestAddTransaction(t *testing.T) {
	myChain := NewChain(2)

	// Add money to Alice via FAUCET to test valid transfers later
	myChain.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	myChain.MinePendingTransactions()

	// 1. Valid transaction
	tx1 := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 50}
	err := myChain.AddTransaction(tx1)
	if err != nil {
		t.Errorf("Expected valid transaction to succeed, got error: %v", err)
	}

	if len(myChain.PendingPool) != 1 {
		t.Errorf("Expected 1 pending transaction, got %d", len(myChain.PendingPool))
	}

	// 2. Reject COINBASE sender
	tx2 := block.Transaction{Sender: "COINBASE", Recipient: "Alice", Amount: 100}
	err = myChain.AddTransaction(tx2)
	if err == nil || !strings.Contains(err.Error(), "COINBASE is reserved") {
		t.Errorf("Expected COINBASE transaction to be rejected, got: %v", err)
	}

	// 3. Reject overspending
	tx3 := block.Transaction{Sender: "Alice", Recipient: "Charlie", Amount: 60}
	err = myChain.AddTransaction(tx3)
	if err == nil {
		t.Errorf("Expected overspending transaction to be rejected, but it succeeded")
	}
}

/*
TestMinePendingTransactions verifies that pending transactions are correctly
mined into a new block, the pending pool is cleared, and the block is linked properly.
It also verifies it fails if there are no pending transactions.
*/
func TestMinePendingTransactions(t *testing.T) {
	myChain := NewChain(2)

	// 1. Fail when no pending transactions
	err := myChain.MinePendingTransactions()
	if err == nil || err.Error() != "No pending transactions to mine" {
		t.Errorf("Expected error when mining empty pool, got: %v", err)
	}

	// 2. Successful mine
	myChain.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	err = myChain.MinePendingTransactions()
	if err != nil {
		t.Errorf("Expected successful mine, got error: %v", err)
	}

	if len(myChain.Blocks) != 2 {
		t.Errorf("Expected chain to have 2 blocks, got %d", len(myChain.Blocks))
	}

	if len(myChain.PendingPool) != 0 {
		t.Errorf("Expected pending pool to be cleared, got %d", len(myChain.PendingPool))
	}

	lastBlock := myChain.Blocks[len(myChain.Blocks)-1]
	if lastBlock.Height != 1 {
		t.Errorf("Expected new block height to be 1, got %d", lastBlock.Height)
	}
	if lastBlock.Header.PrevHash != myChain.Blocks[0].Hash {
		t.Errorf("Expected new block PrevHash to match Genesis hash")
	}
}

/*
TestValidate_InvalidLinks tests that tampering with block hashes or links
(e.g., breaking the PrevHash chain) correctly invalidates the blockchain.
*/
func TestValidate_InvalidLinks(t *testing.T) {
	myChain := NewChain(1)
	myChain.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	myChain.MinePendingTransactions()

	// Tamper with Genesis block Hash
	originalGenesisHash := myChain.Blocks[0].Hash
	myChain.Blocks[0].Hash = "invalidhash"
	result := myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to Genesis block hash tampering")
	}
	myChain.Blocks[0].Hash = originalGenesisHash

	// Tamper with Block 1 Hash
	originalHash := myChain.Blocks[1].Hash
	myChain.Blocks[1].Hash = "invalidhash"
	result = myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to broken link/hash")
	}
	myChain.Blocks[1].Hash = originalHash
}

/*
TestChain_JSONSerialization verifies that an entire Blockchain (Chain struct),
including its blocks, headers, transactions, and pending pool, can be safely
converted to JSON and restored without losing structural integrity.
*/
func TestChain_JSONSerialization(t *testing.T) {
	originalChain := NewChain(3)
	originalChain.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	originalChain.MinePendingTransactions()
	originalChain.AddTransaction(block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 20}) // leave in pending pool

	jsonData, err := json.Marshal(originalChain)
	if err != nil {
		t.Fatalf("Failed to marshal chain to JSON: %v", err)
	}

	var decodedChain Chain
	err = json.Unmarshal(jsonData, &decodedChain)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to chain: %v", err)
	}

	if decodedChain.Difficulty != originalChain.Difficulty {
		t.Errorf("Expected Difficulty %d, got %d", originalChain.Difficulty, decodedChain.Difficulty)
	}

	if len(decodedChain.Blocks) != len(originalChain.Blocks) {
		t.Fatalf("Expected %d blocks, got %d", len(originalChain.Blocks), len(decodedChain.Blocks))
	}

	if decodedChain.Blocks[1].Hash != originalChain.Blocks[1].Hash {
		t.Errorf("Expected Block 1 Hash %s, got %s", originalChain.Blocks[1].Hash, decodedChain.Blocks[1].Hash)
	}

	if len(decodedChain.PendingPool) != len(originalChain.PendingPool) {
		t.Fatalf("Expected %d pending transactions, got %d", len(originalChain.PendingPool), len(decodedChain.PendingPool))
	}

	if decodedChain.PendingPool[0].Amount != originalChain.PendingPool[0].Amount {
		t.Errorf("Expected Pending Transaction Amount %d, got %d", originalChain.PendingPool[0].Amount, decodedChain.PendingPool[0].Amount)
	}
}
