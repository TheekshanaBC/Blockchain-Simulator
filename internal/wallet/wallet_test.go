package wallet

import (
	"os"
	"testing"
)

/*
TestNewWallet verifies that a newly created wallet contains
a valid PrivateKey and a populated PublicKeyBytes slice.
*/
func TestNewWallet(t *testing.T) {
	w := NewWallet()
	if w.PrivateKey == nil {
		t.Errorf("Expected PrivateKey to be generated, got nil")
	}
	if len(w.PublicKeyBytes) == 0 {
		t.Errorf("Expected PublicKeyBytes to be populated, got empty bytes")
	}
}

/*
TestAddressFromPublicKey checks if the derived wallet address
is a valid 64-character SHA-256 hex string.
*/
func TestAddressFromPublicKey(t *testing.T) {
	w := NewWallet()
	address := AddressFromPublicKey(w.PublicKeyBytes)

	if len(address) != 64 {
		t.Errorf("Expected address length of 64 characters (SHA-256 hex), got %d", len(address))
	}
}

/*
TestBytesToPublicKey ensures that we can parse the public key bytes
back into a valid ecdsa.PublicKey object and that the coordinates match.
*/
func TestBytesToPublicKey(t *testing.T) {
	w := NewWallet()
	pubKey := BytesToPublicKey(w.PublicKeyBytes)

	if pubKey == nil {
		t.Errorf("Expected PublicKey to be parsed, got nil")
	}

	// Check if the parsed key matches the original
	if w.PrivateKey.PublicKey.X.Cmp(pubKey.X) != 0 || w.PrivateKey.PublicKey.Y.Cmp(pubKey.Y) != 0 {
		t.Errorf("Parsed PublicKey coordinates do not match original PrivateKey's public part")
	}
}

/*
TestKeystoreOperations tests the complete keystore lifecycle:
saving wallets, loading existing wallets, handling missing wallets,
and retrieving all wallets from a keystore file.
*/
func TestKeystoreOperations(t *testing.T) {
	tempFile := "test_keystore.json"
	defer os.Remove(tempFile) // Cleanup after test

	// Create wallets
	w1 := NewWallet()
	w2 := NewWallet()

	// 1. Test SaveToKeystore
	err := SaveToKeystore(tempFile, "Alice", w1)
	if err != nil {
		t.Fatalf("Failed to save Alice's wallet: %v", err)
	}

	err = SaveToKeystore(tempFile, "Bob", w2)
	if err != nil {
		t.Fatalf("Failed to save Bob's wallet: %v", err)
	}

	// 2. Test LoadFromKeystore
	loadedW1, err := LoadFromKeystore(tempFile, "Alice")
	if err != nil {
		t.Fatalf("Failed to load Alice's wallet: %v", err)
	}

	// Compare private keys
	if loadedW1.PrivateKey.D.Cmp(w1.PrivateKey.D) != 0 {
		t.Errorf("Loaded wallet's private key does not match original")
	}

	// 3. Test loading a non-existent wallet
	_, err = LoadFromKeystore(tempFile, "Charlie")
	if err == nil {
		t.Errorf("Expected an error when loading a non-existent wallet, got nil")
	}

	// 4. Test GetAllWallets
	allWallets, err := GetAllWallets(tempFile)
	if err != nil {
		t.Fatalf("Failed to get all wallets: %v", err)
	}

	if len(allWallets) != 2 {
		t.Errorf("Expected 2 wallets in keystore, got %d", len(allWallets))
	}

	if _, ok := allWallets["Alice"]; !ok {
		t.Errorf("Expected to find Alice's wallet in GetAllWallets")
	}

	if _, ok := allWallets["Bob"]; !ok {
		t.Errorf("Expected to find Bob's wallet in GetAllWallets")
	}
}

/*
TestBytesToPublicKey_InvalidLength ensures that passing wrong-length byte slices
returns nil and does not panic.
*/
func TestBytesToPublicKey_InvalidLength(t *testing.T) {
	// Too short
	shortBytes := []byte{4, 1, 2, 3}
	if BytesToPublicKey(shortBytes) != nil {
		t.Errorf("Expected nil for short public key bytes")
	}

	// Too long
	w := NewWallet()
	longBytes := append(w.PublicKeyBytes, []byte{1, 2, 3}...)
	if BytesToPublicKey(longBytes) != nil {
		t.Errorf("Expected nil for long public key bytes")
	}
}

/*
TestBytesToPublicKey_NotOnCurve ensures that passing a mathematically invalid
point (not on the P-256 curve) returns nil.
*/
func TestBytesToPublicKey_NotOnCurve(t *testing.T) {
	w := NewWallet()
	invalidBytes := make([]byte, len(w.PublicKeyBytes))
	copy(invalidBytes, w.PublicKeyBytes)
	
	// Corrupt the X coordinate to make it almost certainly not on the curve
	invalidBytes[5] ^= 0xFF
	invalidBytes[10] ^= 0xFF

	if BytesToPublicKey(invalidBytes) != nil {
		t.Errorf("Expected nil for a public key point not on the curve")
	}
}
