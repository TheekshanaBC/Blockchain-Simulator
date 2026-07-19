package chain

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/wallet"
	"strings"
	"testing"
)

/*
TestValidate_GenesisMerkleRootMismatch ensures that if the Genesis block's
Merkle Root is tampered with, validation fails appropriately at the root check.
*/
func TestValidate_GenesisMerkleRootMismatch(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)
	c.blocks[0].Header.MerkleRoot = "tampered_root"
	
	res := c.Validate()
	if res.IsValid || !strings.Contains(res.Reason, "Genesis Merkle Root mismatch") {
		t.Errorf("Expected Genesis Merkle Root mismatch, got: %s", res.Reason)
	}
}


/*
TestValidate_CoinbaseAmountWrong ensures that a block is rejected if its
COINBASE transaction has an incorrect mining reward amount.
*/
func TestValidate_CoinbaseAmountWrong(t *testing.T) {
	c := NewChain(0, 10, 60, 0, 5)
	c.RequestFaucetFunds("recipient", 100)
	c.MinePendingTransactions() // Block 1
	
	// Modify the COINBASE tx amount in Block 1
	c.blocks[1].Transactions[0].Amount = 999 
	c.blocks[1].Header.MerkleRoot = block.CalculateMerkleRoot(c.blocks[1].Transactions)
	c.blocks[1].Hash = c.blocks[1].CalculateHash()
	
	res := c.Validate()
	if res.IsValid || !strings.Contains(res.Reason, "Invalid COINBASE reward") {
		t.Errorf("Expected Invalid COINBASE reward, got: %s", res.Reason)
	}
}

/*
TestValidate_SecondCoinbaseMidBlock ensures that a block is rejected if it
contains a COINBASE transaction anywhere other than the very first transaction.
*/
func TestValidate_SecondCoinbaseMidBlock(t *testing.T) {
	c := NewChain(0, 10, 60, 0, 5)
	c.RequestFaucetFunds("recipient", 100)
	c.MinePendingTransactions()
	
	secondCoinbase := block.Transaction{
		Sender:    block.SystemAddressCoinbase,
		Recipient: "Miner2",
		Amount:    block.MiningReward,
		Signature: []byte("0"),
	}
	c.blocks[1].Transactions = append(c.blocks[1].Transactions, secondCoinbase)
	c.blocks[1].Header.MerkleRoot = block.CalculateMerkleRoot(c.blocks[1].Transactions)
	c.blocks[1].Hash = c.blocks[1].CalculateHash()

	res := c.Validate()
	if res.IsValid || !strings.Contains(res.Reason, "COINBASE transaction can only be the first") {
		t.Errorf("Expected COINBASE transaction position error, got: %s", res.Reason)
	}
}

/*
TestValidate_EmptyBlockZeroTransactions ensures that a block with exactly
zero transactions (missing even the COINBASE) is immediately rejected.
*/
func TestValidate_EmptyBlockZeroTransactions(t *testing.T) {
	c := NewChain(0, 10, 60, 0, 5)
	c.RequestFaucetFunds("recipient", 100)
	c.MinePendingTransactions()
	
	c.blocks[1].Transactions = []block.Transaction{}
	c.blocks[1].Header.MerkleRoot = block.CalculateMerkleRoot(c.blocks[1].Transactions)
	c.blocks[1].Hash = c.blocks[1].CalculateHash()

	res := c.Validate()
	if res.IsValid || !strings.Contains(res.Reason, "Block must contain at least one transaction") {
		t.Errorf("Expected empty block error, got: %s", res.Reason)
	}
}

/*
TestValidate_NegativeBalanceFromReplay ensures that if a series of transactions
results in a negative balance during ledger replay, the chain is invalidated.
*/
func TestValidate_NegativeBalanceFromReplay(t *testing.T) {
	c := NewChain(0, 10, 60, 0, 5)
	
	w := wallet.NewWallet()
	addr := wallet.AddressFromPublicKey(w.PublicKeyBytes)
	
	c.RequestFaucetFunds(addr, 100)
	c.MinePendingTransactions() // addr gets 100
	
	// Create a tx spending 150
	tx := block.Transaction{
		Sender:    addr,
		Recipient: "userB",
		Amount:    150,
		Sequence:  1,
		PublicKey: w.PublicKeyBytes,
	}
	tx.Sign(w.PrivateKey)
	
	// Add a valid transaction first to mine a block
	tx2 := block.Transaction{
		Sender:    addr,
		Recipient: "userC",
		Amount:    10,
		Sequence:  1,
		PublicKey: w.PublicKeyBytes,
	}
	tx2.Sign(w.PrivateKey)
	c.AddTransaction(tx2)
	c.MinePendingTransactions()

	// Tamper with the mined block to force an overspend (negative balance)
	c.blocks[2].Transactions[1] = tx // replace tx2 with the overspending tx
	c.blocks[2].Header.MerkleRoot = block.CalculateMerkleRoot(c.blocks[2].Transactions)
	c.blocks[2].Hash = c.blocks[2].CalculateHash()
	
	res := c.Validate()
	if res.IsValid || !strings.Contains(res.Reason, "negative balance for") {
		t.Errorf("Expected negative balance error, got: %s", res.Reason)
	}
}
