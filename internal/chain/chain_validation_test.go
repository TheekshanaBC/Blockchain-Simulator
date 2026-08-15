package chain

import (
	"context"
	"strings"
	"testing"
	"valence/internal/block"
	"valence/internal/wallet"
)

/*
TestValidationAndTamperDetection verifies that the blockchain can correctly
identify when data has been altered. It first checks if a sequence of valid
mined blocks passes validation. Then, it intentionally modifies a transaction
in an already mined block and asserts that the blockchain becomes invalid
and successfully pinpoints the exact block height where the tampering occurred.
*/
func TestValidationAndTamperDetection(t *testing.T) {
	myChain := NewChain(2, 5, 8, 1, 10, 10)
	wAlice := wallet.NewWallet()
	wBob := wallet.NewWallet()
	addrAlice := wAlice.Address()
	addrBob := wBob.Address()

	fTx := createTestFaucetTx(myChain, addrAlice, 100)
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
TestValidate_InvalidLinks tests that tampering with block hashes or links
(e.g., breaking the PrevHash chain) correctly invalidates the blockchain.
*/
func TestValidate_InvalidLinks(t *testing.T) {
	myChain := NewChain(1, 5, 8, 1, 10, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()
	fTx := createTestFaucetTx(myChain, addrAlice, 100)
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
	myChain := NewChain(1, 5, 8, 1, 10, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Give Alice some funds
	fTx := createTestFaucetTx(myChain, addrAlice, 100)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	// Alice sends to Bob
	tx := block.Transaction{
		Sender:    addrAlice,
		Recipient: "Bob",
		Amount:    20,
		Sequence:  1,
		PublicKey: wAlice.PublicKey,
	}
	tx.ComputeID()
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
TestValidate_DifficultyMismatch verifies that the blockchain validation logic
correctly detects and rejects blocks that have tampered with their difficulty
target when a retarget was expected.
*/
func TestValidate_DifficultyMismatch(t *testing.T) {
	myChain := NewChain(2, 3, 10, 1, 10, 10) // N=3
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Mine 4 blocks to trigger a retarget at block 4
	for i := 0; i < 4; i++ {
		fTx := createTestFaucetTx(myChain, addrAlice, 10)
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
	myChain := NewChain(2, 3, 10, 1, 10, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Mine 4 blocks to trigger a retarget at block 4
	for i := 0; i < 4; i++ {
		fTx := createTestFaucetTx(myChain, addrAlice, 10)
		myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")
	}

	// Verify it's initially valid
	result := myChain.Validate()
	if !result.IsValid {
		t.Fatalf("Expected initial chain to be valid, got: %s", result.Reason)
	}

	// Tamper with a timestamp inside the first window (e.g. Block 2)
	myChain.blocks[2].Header.Timestamp += 800 * 1_000_000_000 // Make it look very slow
	myChain.blocks[3].Header.Timestamp += 800 * 1_000_000_000 // Keep them monotonic
	myChain.blocks[4].Header.Timestamp += 800 * 1_000_000_000
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
	myChain := NewChain(2, 3, 100, 1, 10, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// mine 7 blocks (more than 2 retarget windows of size 3)
	for i := 0; i < 7; i++ {
		fTx := createTestFaucetTx(myChain, addrAlice, 10)
		myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")
	}

	if myChain.Difficulty <= 2 {
		t.Errorf("expected difficulty to increase when blocks mine faster than target, got %d", myChain.Difficulty)
	}
}
