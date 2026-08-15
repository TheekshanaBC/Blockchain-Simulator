package chain

import (
	"context"
	"strings"
	"testing"
	"time"
	"valence/internal/block"
	"valence/internal/wallet"
)

/*
TestMaxTxPerBlock verifies that the mining process respects the MaxTxPerBlock
limit, properly slicing the pending pool so that only the allowed number of
transactions are included in the new block, leaving the rest pending.
*/
func TestMaxTxPerBlock(t *testing.T) {
	myChain := NewChain(1, 5, 8, 1, 10, 10)

	// Override the max tx limit for testing
	myChain.MaxTxPerBlock = 2

	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()
	fTx := createTestFaucetTx(myChain, addrAlice, 100)
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
}

func TestSwitchToChain_HeaviestChain(t *testing.T) {
	// Retarget window of 2, target block time 10s
	myChain := NewChain(1, 2, 10, 1, 10, 10)

	baseTime := myChain.blocks[0].Header.Timestamp

	// Create 6 blocks with normal time spacing on myChain, so difficulty stays 1
	for i := 1; i <= 6; i++ {
		b := block.Block{
			Height: i,
			Header: block.BlockHeader{
				PrevHash:   myChain.GetLastBlock().Hash,
				Timestamp:  baseTime + int64(i)*10*1_000_000_000, // Exactly target block time
				Difficulty: 1,
			},
			Transactions: []block.Transaction{
				{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
			},
		}
		b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
		b.Mine(context.Background(), 1)
		err := myChain.AddBlock(b)
		if err != nil {
			t.Fatalf("AddBlock failed for myChain at height %d: %v", i, err)
		}
	}

	// Create a heavier chain starting from the same genesis
	heavierChain := NewChain(1, 2, 10, 1, 10, 10)

	// Mine 5 blocks very fast so difficulty increases
	// Genesis: H=0, Diff=1, T=baseTime
	// Block 1: H=1, Diff=1, T=baseTime+1
	// Block 2: H=2, Diff=1, T=baseTime+2
	// Block 3: H=3, Diff=2, T=baseTime+3 (Retargets here because T2-T1 = 1 < 10)
	// Block 4: H=4, Diff=2, T=baseTime+4
	// Block 5: H=5, Diff=3, T=baseTime+5 (Retargets here because T4-T2 = 2 < 20)

	for i := 1; i <= 5; i++ {
		// Calculate what the difficulty SHOULD be
		expectedDiff := 1
		if i >= 3 {
			expectedDiff = 2
		}
		if i >= 5 {
			expectedDiff = 3
		}

		b := block.Block{
			Height: i,
			Header: block.BlockHeader{
				PrevHash:   heavierChain.GetLastBlock().Hash,
				Timestamp:  baseTime + int64(i)*1_000_000_000, // Fast blocks, 1 second apart
				Difficulty: expectedDiff,
			},
			Transactions: []block.Transaction{
				{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
			},
		}
		b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
		b.Mine(context.Background(), expectedDiff)
		err := heavierChain.AddBlock(b)
		if err != nil {
			t.Fatalf("AddBlock failed for heavierChain at height %d: %v", i, err)
		}
	}

	// myChain has 6 blocks of diff 1. Cumulative work = 16 * 7 = 112
	// heavierChain has 5 blocks. Works: 16 (gen) + 16 (B1) + 16 (B2) + 256 (B3) + 256 (B4) + 4096 (B5) = 4656
	// heavierChain is shorter (height 5 vs 6) but heavier.

	_, err := myChain.SwitchToChain(heavierChain.GetBlocks())
	if err != nil {
		t.Fatalf("SwitchToChain failed, heavier chain should win: %v", err)
	}

	if myChain.Height() != 5 {
		t.Errorf("Expected height to be 5, got %d", myChain.Height())
	}
}

func TestValidate_OversizedBlock(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5, 2) // MaxTxPerBlock = 2

	// Manually construct an oversized block
	coinbaseTx := block.Transaction{
		Sender:    block.SystemAddressCoinbase,
		Recipient: "Miner",
		Amount:    block.MiningReward,
		Timestamp: time.Now().UnixNano(),
	}
	fTx1 := createTestFaucetTx(c, "Alice", 100)
	fTx2 := createTestFaucetTx(c, "Bob", 100)

	newBlock := block.Block{
		Height:       1,
		Transactions: []block.Transaction{coinbaseTx, fTx1, fTx2}, // 3 txs > 2
		Header: block.BlockHeader{
			PrevHash:   c.GetBlocks()[0].Hash,
			Difficulty: c.Difficulty,
			Timestamp:  time.Now().UnixNano(),
		},
	}
	newBlock.Header.MerkleRoot = block.CalculateMerkleRoot(newBlock.Transactions)
	// Perform minimal PoW
	target := strings.Repeat("0", c.Difficulty)
	for {
		newBlock.Header.Nonce++
		newBlock.Hash = newBlock.CalculateHash()
		if strings.HasPrefix(newBlock.Hash, target) {
			break
		}
	}

	err := c.AddBlock(newBlock)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum allowed transactions") {
		t.Errorf("Expected error for oversized block, got: %v", err)
	}
}

/*
TestSwitchToChain_RejectsInflatedDifficultyBeforeWork is a regression test for
the CPU/memory DoS vulnerability where CumulativeWork was called with
attacker-controlled block.Header.Difficulty values before validation.

A block carrying difficulty=1_000_000_000 would cause big.Int.Exp(16, 1e9)
— a ~500 MB allocation — before ValidateBlockSlice ever ran.

After the fix, ValidateBlockSlice runs first and rejects the bogus difficulty
(it doesn't match the retarget schedule), so BlockWork is never called with
the giant exponent.
*/
func TestSwitchToChain_RejectsInflatedDifficultyBeforeWork(t *testing.T) {
	c := NewChain(1, 2, 10, 1, 6, 10)

	// Build a fake chain where block 1 claims an absurd difficulty.
	// This would cause 16^1_000_000_000 if work is computed before validation.
	genesis := c.GetBlocks()[0]
	maliciousBlock := &block.Block{
		Height: 1,
		Header: block.BlockHeader{
			PrevHash:   genesis.Hash,
			Difficulty: 1_000_000_000, // attacker-supplied, way above MaxDifficulty=6
			Timestamp:  genesis.Header.Timestamp + 1,
		},
		Transactions: []block.Transaction{
			{Sender: block.SystemAddressCoinbase, Recipient: "Attacker", Amount: block.MiningReward},
		},
	}
	maliciousBlock.Header.MerkleRoot = block.CalculateMerkleRoot(maliciousBlock.Transactions)
	// Give it a valid-looking hash (no real PoW needed — validation rejects it
	// on difficulty mismatch before the PoW check matters).
	maliciousBlock.Hash = maliciousBlock.CalculateHash()

	_, err := c.SwitchToChain([]*block.Block{genesis, maliciousBlock})
	if err == nil {
		t.Fatal("Expected SwitchToChain to reject a chain with inflated difficulty, but it accepted it")
	}
	// Validation must be the reason — not a "lower cumulative work" reason,
	// which would only be possible if CumulativeWork ran first on the bad input.
	if !strings.Contains(err.Error(), "candidate chain invalid") {
		t.Errorf("Expected rejection reason to be validation failure, got: %v", err)
	}
}

/*
TestSwitchToChain_ValidChainStillAccepted verifies that the validate-before-work
reorder does not break the normal happy path: a legitimately heavier chain
must still be accepted.
*/
func TestSwitchToChain_ValidChainStillAccepted(t *testing.T) {
	c := NewChain(1, 2, 10, 1, 6, 10)
	baseTime := c.GetBlocks()[0].Header.Timestamp

	// Build a competing chain: 2 blocks, both at difficulty=1 with correct PoW.
	other := NewChain(1, 2, 10, 1, 6, 10)
	for i := 1; i <= 2; i++ {
		b := block.Block{
			Height: i,
			Header: block.BlockHeader{
				PrevHash:   other.GetLastBlock().Hash,
				Difficulty: 1,
				Timestamp:  baseTime + int64(i)*10*1_000_000_000,
			},
			Transactions: []block.Transaction{
				{Sender: block.SystemAddressCoinbase, Recipient: "Miner", Amount: block.MiningReward},
			},
		}
		b.Header.MerkleRoot = block.CalculateMerkleRoot(b.Transactions)
		b.Mine(context.Background(), 1)
		if err := other.AddBlock(b); err != nil {
			t.Fatalf("setup: AddBlock height %d failed: %v", i, err)
		}
	}

	_, err := c.SwitchToChain(other.GetBlocks())
	if err != nil {
		t.Fatalf("SwitchToChain rejected a valid heavier chain: %v", err)
	}
	if c.Height() != 2 {
		t.Errorf("Expected chain height 2 after switch, got %d", c.Height())
	}
}
