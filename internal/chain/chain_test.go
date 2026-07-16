package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/wallet"
	"encoding/json"
	"strings"
	"testing"
)

func createSignedTx(w *wallet.Wallet, recipient string, amount int64, sequence uint64) block.Transaction {
	tx := block.Transaction{
		Sender:    wallet.AddressFromPublicKey(w.PublicKeyBytes),
		Recipient: recipient,
		Amount:    amount,
		Sequence:  sequence,
		PublicKey: w.PublicKeyBytes,
	}
	tx.Sign(w.PrivateKey)
	return tx
}

/*
TestValidationAndTamperDetection verifies that the blockchain can correctly
identify when data has been altered. It first checks if a sequence of valid
mined blocks passes validation. Then, it intentionally modifies a transaction
in an already mined block and asserts that the blockchain becomes invalid
and successfully pinpoints the exact block height where the tampering occurred.
*/
func TestValidationAndTamperDetection(t *testing.T) {
	myChain := NewChain(2, 5, 8, 1, 10)
	wAlice := wallet.NewWallet()
	wBob := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)
	addrBob := wallet.AddressFromPublicKey(wBob.PublicKeyBytes)

	myChain.RequestFaucetFunds(addrAlice, 100)
	myChain.MinePendingTransactions()

	tx2 := createSignedTx(wAlice, addrBob, 20, 1)
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
	myChain := NewChain(difficulty, 5, 8, 1, 10)

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
	myChain := NewChain(2, 5, 8, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	// Add money to Alice via FAUCET to test valid transfers later
	myChain.RequestFaucetFunds(addrAlice, 100)
	myChain.MinePendingTransactions()

	// 1. Valid transaction
	tx1 := createSignedTx(wAlice, "Bob", 50, 1)
	err := myChain.AddTransaction(tx1)
	if err != nil {
		t.Errorf("Expected valid transaction to succeed, got error: %v", err)
	}

	if len(myChain.PendingPool) != 1 {
		t.Errorf("Expected 1 pending transaction, got %d", len(myChain.PendingPool))
	}

	// 2. Reject COINBASE sender
	tx2 := createSignedTx(wAlice, "Alice", 100, 2)
	tx2.Sender = "COINBASE" // tamper to test rejection
	err = myChain.AddTransaction(tx2)
	if err == nil || !strings.Contains(err.Error(), "COINBASE is reserved") {
		t.Errorf("Expected COINBASE transaction to be rejected, got: %v", err)
	}

	// 3. Reject overspending
	tx3 := createSignedTx(wAlice, "Charlie", 60, 2)
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
	myChain := NewChain(2, 5, 8, 1, 10)

	// 1. Fail when no pending transactions
	err := myChain.MinePendingTransactions()
	if err == nil || err.Error() != "No pending transactions to mine" {
		t.Errorf("Expected error when mining empty pool, got: %v", err)
	}

	// 2. Successful mine
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)
	myChain.RequestFaucetFunds(addrAlice, 100)
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
	myChain := NewChain(1, 5, 8, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)
	myChain.RequestFaucetFunds(addrAlice, 100)
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
TestValidate_ForgedSignature tests that tampering with a transaction's signature
after it has been mined into a block correctly invalidates the blockchain,
even if the block's hash and Merkle root are recalculated.
*/
func TestValidate_ForgedSignature(t *testing.T) {
	myChain := NewChain(1, 5, 8, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	// Give Alice some funds
	myChain.RequestFaucetFunds(addrAlice, 100)
	myChain.MinePendingTransactions()

	// Alice sends to Bob
	tx := block.Transaction{
		Sender:    addrAlice,
		Recipient: "Bob",
		Amount:    20,
		Sequence:  1,
		PublicKey: wAlice.PublicKeyBytes,
	}
	tx.Sign(wAlice.PrivateKey)

	myChain.AddTransaction(tx)
	myChain.MinePendingTransactions()

	// Now tamper with the signed transaction in the mined block
	tamperedBlock := myChain.Blocks[2]
	// [0] is coinbase, [1] is Alice's tx
	tamperedTx := &tamperedBlock.Transactions[1]

	tamperedTx.Recipient = "Hacker"
	tamperedTx.Signature = []byte("forged-not-a-real-signature")

	// Recalculate block hash and merkle root so it passes those checks
	tamperedBlock.Mine(1) // Re-mine to get a valid hash with the tampered transaction

	result := myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to forged transaction signature")
	}
	if result.Reason != "Invalid transaction signature" {
		t.Errorf("Expected reason to be 'Invalid transaction signature', got '%s'", result.Reason)
	}
}

/*
TestChain_JSONSerialization verifies that an entire Blockchain (Chain struct),
including its blocks, headers, transactions, and pending pool, can be safely
converted to JSON and restored without losing structural integrity.
*/
func TestChain_JSONSerialization(t *testing.T) {
	originalChain := NewChain(3, 5, 8, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	originalChain.RequestFaucetFunds(addrAlice, 100)
	originalChain.MinePendingTransactions()

	tx2 := createSignedTx(wAlice, "Bob", 20, 1)
	originalChain.AddTransaction(tx2) // leave in pending pool

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

func TestValidate_DifficultyMismatch(t *testing.T) {
	myChain := NewChain(2, 3, 10, 1, 10) // N=3
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	// Mine 4 blocks to trigger a retarget at block 4
	for i := 0; i < 4; i++ {
		myChain.RequestFaucetFunds(addrAlice, 10)
		myChain.MinePendingTransactions()
	}

	// Tamper with the difficulty of a block
	myChain.Blocks[2].Header.Difficulty = 99
	myChain.Blocks[2].Hash = myChain.Blocks[2].CalculateHash()

	result := myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to difficulty mismatch")
	}
	if !strings.Contains(result.Reason, "Difficulty retarget mismatch") {
		t.Errorf("Expected reason to be 'Difficulty retarget mismatch', got '%s'", result.Reason)
	}
}

func TestValidate_TamperTimestampRetarget(t *testing.T) {
	myChain := NewChain(2, 3, 10, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	// Mine 4 blocks to trigger a retarget at block 4
	for i := 0; i < 4; i++ {
		myChain.RequestFaucetFunds(addrAlice, 10)
		myChain.MinePendingTransactions()
	}

	// Verify it's initially valid
	result := myChain.Validate()
	if !result.IsValid {
		t.Fatalf("Expected initial chain to be valid, got: %s", result.Reason)
	}

	// Tamper with a timestamp inside the first window (e.g. Block 2)
	myChain.Blocks[2].Header.Timestamp += 1000 // Make it look very slow
	myChain.Blocks[3].Header.Timestamp += 1000 // Keep them monotonic
	myChain.Blocks[4].Header.Timestamp += 1000
	// so it reaches the expected difficulty check for Block 4.
	myChain.Blocks[2].Mine(myChain.Blocks[2].Header.Difficulty)

	myChain.Blocks[3].Header.PrevHash = myChain.Blocks[2].Hash
	myChain.Blocks[3].Mine(myChain.Blocks[3].Header.Difficulty)

	myChain.Blocks[4].Header.PrevHash = myChain.Blocks[3].Hash
	myChain.Blocks[4].Mine(myChain.Blocks[4].Header.Difficulty)

	tamperedResult := myChain.Validate()
	if tamperedResult.IsValid {
		t.Errorf("Expected chain to be invalid after timestamp tamper")
	}
	if !strings.Contains(tamperedResult.Reason, "Difficulty retarget mismatch") {
		t.Errorf("Expected validation to fail with 'Difficulty retarget mismatch', but got '%s'", tamperedResult.Reason)
	}
}

func TestRetarget_ConvergesTowardTarget(t *testing.T) {
	// targetBlockTimeSec is 100, which is far above actual mine time (almost instant)
	myChain := NewChain(2, 3, 100, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	// mine 7 blocks (more than 2 retarget windows of size 3)
	for i := 0; i < 7; i++ {
		myChain.RequestFaucetFunds(addrAlice, 10)
		myChain.MinePendingTransactions()
	}

	if myChain.Difficulty <= 2 {
		t.Errorf("expected difficulty to increase when blocks mine faster than target, got %d", myChain.Difficulty)
	}
}
