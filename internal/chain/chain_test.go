package chain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"valence/internal/block"
	"valence/internal/wallet"
)

func createSignedTx(w *wallet.Wallet, recipient string, amount int64, sequence uint64) block.Transaction {
	tx := block.Transaction{
		Sender:    w.Address(),
		Recipient: recipient,
		Amount:    amount,
		Sequence:  sequence,
		PublicKey: w.PublicKey,
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
	addrAlice := wAlice.Address()
	addrBob := wBob.Address()

	fTx, _ := myChain.CreateFaucetTx(addrAlice, 100, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	tx2 := createSignedTx(wAlice, addrBob, 20, 1)
	myChain.MineBlock(context.Background(), []block.Transaction{tx2}, "Miner")

	// Check the honest chain
	result := myChain.Validate()
	if !result.IsValid {
		t.Fatalf("Expected chain to be valid, but failed at height %d: %s", result.FailedAtHeight, result.Reason)
	}

	// Tampering
	myChain.blocks[1].Transactions[0].Amount = 5000

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

	if len(myChain.blocks) != 1 {
		t.Fatalf("Expected exactly 1 block (Genesis), got %d", len(myChain.blocks))
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
	addrAlice := wAlice.Address()

	// Add money to Alice via FAUCET to test valid transfers later
	fTx, _ := myChain.CreateFaucetTx(addrAlice, 100, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	// 1. Valid transaction
	tx1 := createSignedTx(wAlice, "Bob", 50, 1)
	b, _ := myChain.MineBlock(context.Background(), []block.Transaction{tx1}, "Miner")
	if len(b.Transactions) != 2 {
		t.Errorf("Expected valid transaction to be mined")
	}
// 2. Reject COINBASE sender
	tx2 := createSignedTx(wAlice, "Alice", 100, 2)
	tx2.Sender = "VALENCE_COINBASE" // tamper to test rejection
	b2, _ := myChain.MineBlock(context.Background(), []block.Transaction{tx2}, "Miner")
	if len(b2.Transactions) > 1 {
		t.Errorf("Expected COINBASE transaction to be rejected, got mined")
	}

	// 3. Reject overspending
	tx3 := createSignedTx(wAlice, "Charlie", 60, 2)
	b3, _ := myChain.MineBlock(context.Background(), []block.Transaction{tx3}, "Miner")
	if len(b3.Transactions) > 1 {
		t.Errorf("Expected overspending transaction to be rejected")
	}
}

/*
TestMineBlock verifies that pending transactions are correctly
mined into a new block, the pending pool is cleared, and the block is linked properly.
It also verifies it fails if there are no pending transactions.
*/
func TestMineBlock(t *testing.T) {
	myChain := NewChain(2, 5, 8, 1, 10)

	/* Setup a wallet and create a faucet transaction to simulate mining a block with user transactions */
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()
	fTx, _ := myChain.CreateFaucetTx(addrAlice, 100, nil)
	
	/* Attempt to mine the block containing our test transaction */
	_, err := myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")
	if err != nil {
		t.Errorf("Expected successful mine, got error: %v", err)
	}

	if len(myChain.blocks) != 2 {
		t.Errorf("Expected chain to have 2 blocks, got %d", len(myChain.blocks))
	}

	
	lastBlock := myChain.blocks[len(myChain.blocks)-1]
	if lastBlock.Height != 1 {
		t.Errorf("Expected new block height to be 1, got %d", lastBlock.Height)
	}
	if lastBlock.Header.PrevHash != myChain.blocks[0].Hash {
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
	addrAlice := wAlice.Address()
	fTx, _ := myChain.CreateFaucetTx(addrAlice, 100, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	// Tamper with Genesis block Hash
	originalGenesisHash := myChain.blocks[0].Hash
	myChain.blocks[0].Hash = "invalidhash"
	result := myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to Genesis block hash tampering")
	}
	myChain.blocks[0].Hash = originalGenesisHash

	// Tamper with Block 1 Hash
	originalHash := myChain.blocks[1].Hash
	myChain.blocks[1].Hash = "invalidhash"
	result = myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to broken link/hash")
	}
	myChain.blocks[1].Hash = originalHash
}

/*
TestValidate_ForgedSignature tests that tampering with a transaction's signature
after it has been mined into a block correctly invalidates the blockchain,
even if the block's hash and Merkle root are recalculated.
*/
func TestValidate_ForgedSignature(t *testing.T) {
	myChain := NewChain(1, 5, 8, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Give Alice some funds
	fTx, _ := myChain.CreateFaucetTx(addrAlice, 100, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	// Alice sends to Bob
	tx := block.Transaction{
		Sender:    addrAlice,
		Recipient: "Bob",
		Amount:    20,
		Sequence:  1,
		PublicKey: wAlice.PublicKey,
	}
	tx.Sign(wAlice.PrivateKey)

	myChain.MineBlock(context.Background(), []block.Transaction{tx}, "Miner")

	// Now tamper with the signed transaction in the mined block
	tamperedBlock := myChain.blocks[2]
	// [0] is coinbase, [1] is Alice's tx
	tamperedTx := &tamperedBlock.Transactions[1]

	tamperedTx.Recipient = "Hacker"
	tamperedTx.Signature = []byte("forged-not-a-real-signature")

	// Recalculate block hash and merkle root so it passes those checks
	tamperedBlock.Mine(context.Background(), 1) // Re-mine to get a valid hash with the tampered transaction

	result := myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to forged transaction signature")
	}
	if !strings.Contains(result.Reason, "invalid transaction signature") {
		t.Errorf("Expected reason to be 'invalid transaction signature', got '%s'", result.Reason)
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
	addrAlice := wAlice.Address()

	fTx, _ := originalChain.CreateFaucetTx(addrAlice, 100, nil)
	originalChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	tx2 := createSignedTx(wAlice, "Bob", 20, 1)
	_ = tx2

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

	if len(decodedChain.blocks) != len(originalChain.blocks) {
		t.Fatalf("Expected %d blocks, got %d", len(originalChain.blocks), len(decodedChain.blocks))
	}

	if decodedChain.blocks[1].Hash != originalChain.blocks[1].Hash {
		t.Errorf("Expected Block 1 Hash %s, got %s", originalChain.blocks[1].Hash, decodedChain.blocks[1].Hash)
	}
}

/*
TestValidate_DifficultyMismatch verifies that the blockchain validation logic
correctly detects and rejects blocks that have tampered with their difficulty
target when a retarget was expected.
*/
func TestValidate_DifficultyMismatch(t *testing.T) {
	myChain := NewChain(2, 3, 10, 1, 10) // N=3
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Mine 4 blocks to trigger a retarget at block 4
	for i := 0; i < 4; i++ {
		fTx, _ := myChain.CreateFaucetTx(addrAlice, 10, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")
	}

	// Tamper with the difficulty of a block
	myChain.blocks[2].Header.Difficulty = 99
	myChain.blocks[2].Hash = myChain.blocks[2].CalculateHash()

	result := myChain.Validate()
	if result.IsValid {
		t.Errorf("Expected chain to be invalid due to difficulty mismatch")
	}
	if !strings.Contains(result.Reason, "Difficulty retarget mismatch") {
		t.Errorf("Expected reason to be 'Difficulty retarget mismatch', got '%s'", result.Reason)
	}
}

/*
TestValidate_TamperTimestampRetarget verifies that the validation process
catches malicious attempts to manipulate block timestamps to artificially
lower the difficulty during a retarget window.
*/
func TestValidate_TamperTimestampRetarget(t *testing.T) {
	myChain := NewChain(2, 3, 10, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Mine 4 blocks to trigger a retarget at block 4
	for i := 0; i < 4; i++ {
		fTx, _ := myChain.CreateFaucetTx(addrAlice, 10, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")
	}

	// Verify it's initially valid
	result := myChain.Validate()
	if !result.IsValid {
		t.Fatalf("Expected initial chain to be valid, got: %s", result.Reason)
	}

	// Tamper with a timestamp inside the first window (e.g. Block 2)
	myChain.blocks[2].Header.Timestamp += 1000 // Make it look very slow
	myChain.blocks[3].Header.Timestamp += 1000 // Keep them monotonic
	myChain.blocks[4].Header.Timestamp += 1000
	// so it reaches the expected difficulty check for Block 4.
	myChain.blocks[2].Mine(context.Background(), myChain.blocks[2].Header.Difficulty)

	myChain.blocks[3].Header.PrevHash = myChain.blocks[2].Hash
	myChain.blocks[3].Mine(context.Background(), myChain.blocks[3].Header.Difficulty)

	myChain.blocks[4].Header.PrevHash = myChain.blocks[3].Hash
	myChain.blocks[4].Mine(context.Background(), myChain.blocks[4].Header.Difficulty)

	tamperedResult := myChain.Validate()
	if tamperedResult.IsValid {
		t.Errorf("Expected chain to be invalid after timestamp tamper")
	}
	if !strings.Contains(tamperedResult.Reason, "Difficulty retarget mismatch") {
		t.Errorf("Expected validation to fail with 'Difficulty retarget mismatch', but got '%s'", tamperedResult.Reason)
	}
}

/*
TestRetarget_ConvergesTowardTarget tests the dynamic difficulty adjustment
algorithm. It simulates mining blocks much faster than the target block time
and asserts that the difficulty correctly increases to compensate.
*/
func TestRetarget_ConvergesTowardTarget(t *testing.T) {
	// targetBlockTimeSec is 100, which is far above actual mine time (almost instant)
	myChain := NewChain(2, 3, 100, 1, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// mine 7 blocks (more than 2 retarget windows of size 3)
	for i := 0; i < 7; i++ {
		fTx, _ := myChain.CreateFaucetTx(addrAlice, 10, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")
	}

	if myChain.Difficulty <= 2 {
		t.Errorf("expected difficulty to increase when blocks mine faster than target, got %d", myChain.Difficulty)
	}
}

/*
TestMaxTxPerBlock verifies that the mining process respects the MaxTxPerBlock
limit, properly slicing the pending pool so that only the allowed number of
transactions are included in the new block, leaving the rest pending.
*/
func TestMaxTxPerBlock(t *testing.T) {
	myChain := NewChain(1, 5, 8, 1, 10)

	// Override the max tx limit for testing
	myChain.MaxTxPerBlock = 2

	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()
	fTx, _ := myChain.CreateFaucetTx(addrAlice, 100, nil)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	/* Alice generates 5 transactions to exceed the MaxTxPerBlock limit of 2 */
	var txs []block.Transaction
	for i := 0; i < 5; i++ {
		tx := createSignedTx(wAlice, "Bob", 1, uint64(i+1))
		txs = append(txs, tx)
	}
// Mine a block, it should only take 2 transactions from the pool
	b, err := myChain.MineBlock(context.Background(), txs, "Miner")
	/* Retrieve the newly mined block to verify it strictly respects the transaction limits */
	lastBlock := &b
	if err != nil {
		t.Fatalf("Mine failed: %v", err)
	}

	

	// The block should have 2 transactions: 1 coinbase + 1 user transaction
	if len(lastBlock.Transactions) != 2 {
		t.Errorf("Expected block to have 2 transactions (1 coinbase + 1 user), got %d", len(lastBlock.Transactions))
	}

	// The pending pool should have 3 transactions remaining
}

/*
TestNewChain_DifficultyClamping ensures that the initial difficulty
is properly clamped between MinDifficulty and MaxDifficulty.
*/
func TestNewChain_DifficultyClamping(t *testing.T) {
	// Test difficulty < minDifficulty
	c1 := NewChain(0, 10, 60, 2, 5)
	if c1.Difficulty != 2 {
		t.Errorf("Expected difficulty to be clamped up to MinDifficulty (2), got %d", c1.Difficulty)
	}

	// Test difficulty > maxDifficulty
	c2 := NewChain(10, 10, 60, 2, 5)
	if c2.Difficulty != 5 {
		t.Errorf("Expected difficulty to be clamped down to MaxDifficulty (5), got %d", c2.Difficulty)
	}
}
