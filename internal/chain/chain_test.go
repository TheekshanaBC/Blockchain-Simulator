package chain

import (
	"context"
	"encoding/json"
	"testing"
	"valence/internal/block"
	"valence/internal/wallet"
)

var testFaucetWallet *wallet.Wallet
var seqCache = make(map[*Chain]uint64)

func init() {
	testFaucetWallet, _ = wallet.WalletFromBase64("AdUl1LWR0NtSPlR6NktiYVptv2sKOwAZ8djfTt9u1Mk=")
}

func createTestFaucetTx(c *Chain, recipient string, amount int64) block.Transaction {
	seqCache[c]++
	tx, err := c.CreateFaucetTx(recipient, amount, testFaucetWallet, seqCache[c], 1_000_000_000*block.ElectronsPerVCN, nil)
	if err != nil {
		panic(err)
	}
	return tx
}

func createSignedTx(w *wallet.Wallet, recipient string, amount int64, sequence uint64) block.Transaction {
	tx := block.Transaction{
		Sender:    w.Address(),
		Recipient: recipient,
		Amount:    amount,
		Sequence:  sequence,
		Fee:       10, // Default fee for tests
		PublicKey: w.PublicKey,
	}
	tx.ComputeID()
	tx.Sign(w.PrivateKey)
	return tx
}

/*
TestNewChain verifies that a new blockchain is initialized correctly with
the given difficulty, an empty pending pool, and exactly one valid Genesis block.
*/
func TestNewChain(t *testing.T) {
	difficulty := 2
	myChain := NewChain(difficulty, 5, 8, 1, 10, 10)

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
	myChain := NewChain(2, 5, 8, 1, 10, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	// Add money to Alice via FAUCET to test valid transfers later
	fTx := createTestFaucetTx(myChain, addrAlice, 100)
	myChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	// 1. Valid transaction
	tx1 := createSignedTx(wAlice, "Bob", 50, 1)
	b, _ := myChain.MineBlock(context.Background(), []block.Transaction{tx1}, "Miner")
	if len(b.Transactions) != 2 {
		t.Errorf("Expected valid transaction to be mined")
	}
	if b.Transactions[0].Amount != block.MiningReward + tx1.Fee {
		t.Errorf("Expected coinbase amount %d, got %d", block.MiningReward + tx1.Fee, b.Transactions[0].Amount)
	}
	// 2. Reject COINBASE sender
	tx2 := createSignedTx(wAlice, "Alice", 300, 2)
	tx2.Sender = block.SystemAddressCoinbase // tamper to test rejection
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
TestMineBlock verifies that supplied transactions are correctly
mined into a new block, and the block is linked properly with incremented height.
*/
func TestMineBlock(t *testing.T) {
	myChain := NewChain(2, 5, 8, 1, 10, 10)

	/* Setup a wallet and create a faucet transaction to simulate mining a block with user transactions */
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()
	fTx := createTestFaucetTx(myChain, addrAlice, 100)

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
TestChain_JSONSerialization verifies that an entire Blockchain (Chain struct),
including its blocks, headers, transactions, and pending pool, can be safely
converted to JSON and restored without losing structural integrity.
*/
func TestChain_JSONSerialization(t *testing.T) {
	originalChain := NewChain(3, 5, 8, 1, 10, 10)
	wAlice := wallet.NewWallet()
	addrAlice := wAlice.Address()

	fTx := createTestFaucetTx(originalChain, addrAlice, 100)
	originalChain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

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
TestNewChain_DifficultyClamping ensures that the initial difficulty
is properly clamped between MinDifficulty and MaxDifficulty.
*/
func TestNewChain_DifficultyClamping(t *testing.T) {
	// Test difficulty < minDifficulty
	c1 := NewChain(0, 10, 60, 2, 5, 10)
	if c1.Difficulty != 2 {
		t.Errorf("Expected difficulty to be clamped up to MinDifficulty (2), got %d", c1.Difficulty)
	}

	// Test difficulty > maxDifficulty
	c2 := NewChain(10, 10, 60, 2, 5, 10)
	if c2.Difficulty != 5 {
		t.Errorf("Expected difficulty to be clamped down to MaxDifficulty (5), got %d", c2.Difficulty)
	}
}
