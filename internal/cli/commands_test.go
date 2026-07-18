package cli

import (
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/wallet"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// Helper to capture stdout
func captureOutput(f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	oldStdout := os.Stdout
	defer func() {
		os.Stdout = oldStdout
	}()
	os.Stdout = w

	f()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

/*
TestHandleAddTx_NoWallet verifies that attempting to add a transaction
without an active wallet loaded results in an error message.
*/
func TestHandleAddTx_NoWallet(t *testing.T) {
	ctx := &cliContext{chain: chain.NewChain(2, 10, 60, 1, 5)}
	out := captureOutput(func() {
		handleAddTx(ctx, []string{"addtx", "recipient", "100"})
	})
	if !strings.Contains(out, "No active wallet") {
		t.Errorf("Expected No active wallet error, got: %s", out)
	}
}

/*
TestHandleAddTx_WrongArgs verifies that attempting to add a transaction
with an incorrect number of arguments (e.g., missing amount) results in
an invalid arguments error message.
*/
func TestHandleAddTx_WrongArgs(t *testing.T) {
	w := wallet.NewWallet()
	ctx := &cliContext{
		chain:        chain.NewChain(2, 10, 60, 1, 5),
		activeWallet: w,
	}

	out := captureOutput(func() {
		handleAddTx(ctx, []string{"addtx", "recipient"}) // missing amount
	})
	if !strings.Contains(out, "Invalid arguments!") {
		t.Errorf("Expected Invalid arguments error, got: %s", out)
	}
}

/*
TestHandleAddTx_NonNumericAmount verifies that providing a non-numeric
string as the amount parameter fails gracefully with an appropriate error.
*/
func TestHandleAddTx_NonNumericAmount(t *testing.T) {
	w := wallet.NewWallet()
	ctx := &cliContext{
		chain:        chain.NewChain(2, 10, 60, 1, 5),
		activeWallet: w,
	}

	out := captureOutput(func() {
		handleAddTx(ctx, []string{"addtx", "recipient", "abc"})
	})
	if !strings.Contains(out, "Amount must be a number") {
		t.Errorf("Expected Amount must be a number error, got: %s", out)
	}
}

/*
TestHandleAddTx_HappyPath verifies that a valid addtx command with correct
arguments and sufficient balance correctly signs the transaction and adds
it to the pending pool.
*/
func TestHandleAddTx_HappyPath(t *testing.T) {
	w := wallet.NewWallet()
	c := chain.NewChain(0, 10, 60, 1, 5) // difficulty 0 for instant mine
	ctx := &cliContext{
		chain:        c,
		activeWallet: w,
	}

	// First give the wallet some balance via faucet and mine
	addr := wallet.AddressFromPublicKey(w.PublicKeyBytes)
	c.RequestFaucetFunds(addr, 500)
	c.MinePendingTransactions()

	out := captureOutput(func() {
		handleAddTx(ctx, []string{"addtx", "recipient", "100"})
	})
	if !strings.Contains(out, "Transaction signed and added to the pending pool!") {
		t.Errorf("Expected successful addtx, got: %s", out)
	}
}

/*
TestHandleValidate_Valid verifies that running the validate command on a
correct blockchain outputs a success message indicating the chain is valid.
*/
func TestHandleValidate_Valid(t *testing.T) {
	ctx := &cliContext{chain: chain.NewChain(2, 10, 60, 1, 5)}
	out := captureOutput(func() {
		handleValidate(ctx, []string{"validate"})
	})
	if !strings.Contains(out, "Chain is valid!") {
		t.Errorf("Expected Chain is valid, got: %s", out)
	}
}

/*
TestHandleValidate_Invalid verifies that running the validate command on a
blockchain with a corrupted block correctly identifies the invalid chain.
*/
func TestHandleValidate_Invalid(t *testing.T) {
	c := chain.NewChain(0, 10, 60, 1, 5)
	c.RequestFaucetFunds("someone", 100)
	c.MinePendingTransactions()
	
	// Corrupt a block
	c.Blocks[1].Hash = "corrupted_hash"
	ctx := &cliContext{chain: c}

	out := captureOutput(func() {
		handleValidate(ctx, []string{"validate"})
	})
	if !strings.Contains(out, "Chain is INVALID") {
		t.Errorf("Expected Chain is INVALID error, got: %s", out)
	}
}

/*
TestHandlePrint verifies that the print command successfully visualizes
the blockchain structure and its transactions without panicking. It also
provides coverage for display.go's printBlockchain and printLine.
*/
func TestHandlePrint(t *testing.T) {
	c := chain.NewChain(0, 10, 60, 1, 5)
	c.RequestFaucetFunds("someone", 100)
	c.MinePendingTransactions()
	
	ctx := &cliContext{
		chain:      c,
		walletFile: "non_existent_wallet.json", // to cover load failure warning
	}
	out := captureOutput(func() {
		handlePrint(ctx, []string{"print"})
	})
	if !strings.Contains(out, "Blockchain Visualizer") {
		t.Errorf("Expected output to contain Blockchain Visualizer, got: %s", out)
	}
	if !strings.Contains(out, "Transactions:") {
		t.Errorf("Expected output to contain Transactions:, got: %s", out)
	}
}

/*
TestHandleClear verifies that the clear command outputs the correct terminal
escape sequence to clear the screen.
*/
func TestHandleClear(t *testing.T) {
	ctx := &cliContext{}
	out := captureOutput(func() {
		handleClear(ctx, []string{"clear"})
	})
	// Check for clear screen escape sequence
	if !strings.Contains(out, "\033[H\033[2J") {
		t.Errorf("Expected clear sequence, got: %s", out)
	}
}

/*
TestHandleLoadWallet verifies the behavior of the loadwallet command,
ensuring it handles invalid arguments, nonexistent wallets, and successful
wallet loads correctly.
*/
func TestHandleLoadWallet(t *testing.T) {
	// First create a wallet
	testDir := "test_data"
	os.Mkdir(testDir, 0755)
	defer os.RemoveAll(testDir)
	
	walletFile := testDir + "/wallet.json"
	ctx := &cliContext{walletFile: walletFile}

	// 1. Invalid args
	outArgs := captureOutput(func() {
		handleLoadWallet(ctx, []string{"loadwallet"})
	})
	if !strings.Contains(outArgs, "Please specify a name") {
		t.Errorf("Expected specify a name error, got: %s", outArgs)
	}

	// 2. Load non-existent
	outNoExist := captureOutput(func() {
		handleLoadWallet(ctx, []string{"loadwallet", "nope"})
	})
	if !strings.Contains(outNoExist, "Error loading wallet") {
		t.Errorf("Expected Error loading wallet, got: %s", outNoExist)
	}

	// 3. Valid load
	w := wallet.NewWallet()
	wallet.SaveToKeystore(walletFile, "my_wallet", w)

	outValid := captureOutput(func() {
		handleLoadWallet(ctx, []string{"loadwallet", "my_wallet"})
	})
	if !strings.Contains(outValid, "loaded successfully!") {
		t.Errorf("Expected loaded successfully, got: %s", outValid)
	}
	if ctx.activeWalletName != "my_wallet" {
		t.Errorf("Expected active wallet to be set")
	}
}
