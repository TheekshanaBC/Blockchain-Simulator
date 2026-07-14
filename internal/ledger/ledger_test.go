package ledger

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/wallet"
	"testing"
)

/*
TestValidateTransaction tests the basic rules of transaction validation.
It checks three main scenarios: preventing overspending (sending more than the balance),
preventing zero-amount transactions, and successfully validating a good transaction.
*/
func TestValidateTransaction(t *testing.T) {
	wAlice := wallet.NewWallet()
	addrAlice := wallet.AddressFromPublicKey(wAlice.PublicKeyBytes)

	balances := map[string]int64{
		addrAlice: 100,
		"Bob":     50,
	}

	sequences := map[string]uint64{
		addrAlice: 0,
	}

	createTx := func(recipient string, amount int64) block.Transaction {
		tx := block.Transaction{
			Sender:    addrAlice,
			Recipient: recipient,
			Amount:    amount,
			Sequence:  1,
			PublicKey: wAlice.PublicKeyBytes,
		}
		tx.Sign(wAlice.PrivateKey)
		return tx
	}

	// Overspending
	overTx := createTx("Bob", 150)
	err := ValidateTransaction(overTx, balances, sequences)
	if err == nil {
		t.Errorf("Expected an error for overspending, but got nil")
	}

	// Zero amount transaction
	zeroTx := createTx("Bob", 0)
	err = ValidateTransaction(zeroTx, balances, sequences)
	if err == nil {
		t.Errorf("Expected an error for zero amount transaction, but got nil")
	}

	// Good Transaction
	goodTx := createTx("Bob", 20)
	err = ValidateTransaction(goodTx, balances, sequences)
	if err != nil {
		t.Errorf("Did not expect an error for valid transaction, but got: %v", err)
	}

	// Invalid Sequence
	badSeqTx := createTx("Bob", 20)
	badSeqTx.Sequence = 5 // Expected 1
	badSeqTx.Sign(wAlice.PrivateKey)
	err = ValidateTransaction(badSeqTx, balances, sequences)
	if err == nil {
		t.Errorf("Expected an error for invalid sequence, but got nil")
	}

	// Send to COINBASE
	coinbaseTx := createTx("COINBASE", 10)
	err = ValidateTransaction(coinbaseTx, balances, sequences)
	if err == nil {
		t.Errorf("Expected an error for sending to COINBASE, but got nil")
	}

	// Negative amount transaction
	negTx := createTx("Bob", -10)
	err = ValidateTransaction(negTx, balances, sequences)
	if err == nil {
		t.Errorf("Expected an error for negative amount, but got nil")
	}

	// System Senders (FAUCET/COINBASE) bypassing limits
	sysTx := block.Transaction{Sender: "FAUCET", Recipient: addrAlice, Amount: 10000}
	err = ValidateTransaction(sysTx, balances, sequences)
	if err != nil {
		t.Errorf("Expected FAUCET to bypass balance checks, but got error: %v", err)
	}
}

/*
TestCalculateBalances verifies that iterating through the blockchain correctly
tallies the balances for all users based on their incoming and outgoing transactions.
It also ensures that FAUCET and COINBASE special senders do not get negative balances.
*/
func TestCalculateBalances(t *testing.T) {
	chain := []*block.Block{
		{
			Transactions: []block.Transaction{
				{Sender: "FAUCET", Recipient: "Alice", Amount: 100},
				{Sender: "COINBASE", Recipient: "Miner", Amount: 50},
			},
		},
		{
			Transactions: []block.Transaction{
				{Sender: "Alice", Recipient: "Bob", Amount: 30},
				{Sender: "Bob", Recipient: "Charlie", Amount: 10},
				{Sender: "Dave", Recipient: "Eve", Amount: 0}, // Zero amount, should be skipped
			},
		},
	}

	balances := CalculateBalances(chain)

	if balances["Alice"] != 70 {
		t.Errorf("Expected Alice balance 70, got %d", balances["Alice"])
	}
	if balances["Bob"] != 20 {
		t.Errorf("Expected Bob balance 20, got %d", balances["Bob"])
	}
	if balances["Charlie"] != 10 {
		t.Errorf("Expected Charlie balance 10, got %d", balances["Charlie"])
	}
	if balances["Miner"] != 50 {
		t.Errorf("Expected Miner balance 50, got %d", balances["Miner"])
	}

	// FAUCET and COINBASE should not have balances tracked negatively
	if val, exists := balances["FAUCET"]; exists && val < 0 {
		t.Errorf("FAUCET should not have a negative balance")
	}
	if val, exists := balances["COINBASE"]; exists && val < 0 {
		t.Errorf("COINBASE should not have a negative balance")
	}
}

/*
TestCalculateAvailableBalances verifies that pending transactions are correctly
subtracted from a user's balance to prevent double-spending before the transactions
are mined into a block.
*/
func TestCalculateAvailableBalances(t *testing.T) {
	chain := []*block.Block{
		{
			Transactions: []block.Transaction{
				{Sender: "FAUCET", Recipient: "Alice", Amount: 100},
			},
		},
	}

	pendingPool := []block.Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 40},
		{Sender: "FAUCET", Recipient: "Bob", Amount: 1000}, // FAUCET shouldn't be deducted
	}

	available := CalculateAvailableBalances(chain, pendingPool)

	if available["Alice"] != 60 {
		t.Errorf("Expected Alice available balance to be 60, got %d", available["Alice"])
	}
	// Bob hasn't received it yet because it's pending, but CalculateAvailableBalances
	// only deducts outbounds, it doesn't add pending inbounds. Let's check Bob.
	if available["Bob"] != 0 {
		t.Errorf("Expected Bob available balance to be 0 (pending inbounds not counted), got %d", available["Bob"])
	}
}
