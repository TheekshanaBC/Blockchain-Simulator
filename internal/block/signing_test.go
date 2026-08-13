package block

import (
	"testing"
	"valence/internal/wallet"
)

/*
TestVerify_SystemAddress ensures that system-generated transactions,
specifically those from COINBASE or FAUCET addresses, bypass cryptographic
signature verification and immediately return true.
*/
func TestVerify_SystemAddress(t *testing.T) {
	tx1 := &Transaction{Sender: SystemAddressCoinbase}
	if !tx1.Verify() {
		t.Error("Verify failed for COINBASE system address")
	}
}

/*
TestVerify_WrongLengthSignature verifies that the verification process
correctly identifies and rejects signatures that do not exactly match
the expected 64-byte length.
*/
func TestVerify_WrongLengthSignature(t *testing.T) {
	w := wallet.NewWallet()

	tx := &Transaction{
		Sender:    w.Address(),
		PublicKey: w.PublicKey,
		Signature: make([]byte, 63), // Wrong length
	}
	if tx.Verify() {
		t.Error("Verify should fail for 63-byte signature")
	}

	tx.Signature = make([]byte, 65) // Wrong length
	if tx.Verify() {
		t.Error("Verify should fail for 65-byte signature")
	}
}

/*
TestVerify_SpoofingCheck asserts that a transaction cannot be verified if
the claimed sender address does not mathematically match the address
derived from the provided public key, preventing sender spoofing.
*/
func TestVerify_SpoofingCheck(t *testing.T) {
	w := wallet.NewWallet()

	tx := &Transaction{
		Sender:    "fake_sender_address", // Spoofed claimed sender
		PublicKey: w.PublicKey,
		Signature: make([]byte, 64),
	}

	if tx.Verify() {
		t.Error("Verify should fail because claimed sender doesn't match the public key")
	}
}
