package block

import (
	"blockchain-simulator/internal/wallet"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"
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

	tx2 := &Transaction{Sender: SystemAddressFaucet}
	if !tx2.Verify() {
		t.Error("Verify failed for FAUCET system address")
	}
}

/*
TestVerify_WrongLengthSignature verifies that the verification process
correctly identifies and rejects signatures that do not exactly match
the expected 64-byte length.
*/
func TestVerify_WrongLengthSignature(t *testing.T) {
	w := wallet.NewWallet()
	pubKeyBytes := w.PublicKeyBytes

	tx := &Transaction{
		Sender:    wallet.AddressFromPublicKey(pubKeyBytes),
		PublicKey: pubKeyBytes,
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
TestVerify_BytesToPublicKeyGarbage checks the robustness of the public key
parsing logic by supplying malformed bytes. It ensures that such inputs
cause the parsing to fail and the verification to return false.
*/
func TestVerify_BytesToPublicKeyGarbage(t *testing.T) {
	tx := &Transaction{
		Sender:    "some_address",
		PublicKey: []byte("garbage_bytes_that_are_not_a_valid_public_key"),
		Signature: make([]byte, 64),
	}
	
	// Should fail because BytesToPublicKey will return nil
	if tx.Verify() {
		t.Error("Verify should fail for garbage public key")
	}
}

/*
TestVerify_SpoofingCheck asserts that a transaction cannot be verified if
the claimed sender address does not mathematically match the address
derived from the provided public key, preventing sender spoofing.
*/
func TestVerify_SpoofingCheck(t *testing.T) {
	w := wallet.NewWallet()
	pubKeyBytes := w.PublicKeyBytes

	tx := &Transaction{
		Sender:    "fake_sender_address", // Spoofed claimed sender
		PublicKey: pubKeyBytes,
		Signature: make([]byte, 64),
	}

	if tx.Verify() {
		t.Error("Verify should fail because claimed sender doesn't match the public key")
	}
}

/*
TestVerify_LowSMalleability tests the mitigation against ECDSA signature
malleability. It manually constructs a signature with a High-S value
and asserts that the verification logic strictly enforces the Low-S rule
and rejects it.
*/
func TestVerify_LowSMalleability(t *testing.T) {
	w := wallet.NewWallet()
	pubKeyBytes := w.PublicKeyBytes
	privKey := w.PrivateKey
	
	tx := &Transaction{
		Sender:    wallet.AddressFromPublicKey(pubKeyBytes),
		PublicKey: pubKeyBytes,
		Amount:    100,
	}
	
	hash := tx.Hash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		t.Fatal(err)
	}

	N := elliptic.P256().Params().N
	halfN := new(big.Int).Div(N, big.NewInt(2))

	// Ensure we have a High S value
	if s.Cmp(halfN) <= 0 {
		s.Sub(N, s) // Flip to high S
	}

	rBytes := r.Bytes()
	sBytes := s.Bytes()
	
	signature := make([]byte, 64)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)
	
	tx.Signature = signature

	if tx.Verify() {
		t.Error("Verify should fail for malleable high-S signature")
	}
}
