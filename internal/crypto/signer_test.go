package crypto

import (
	"testing"
)

/*
TestGenerateKeyPair verifies that the crypto package can successfully
generate a new Ed25519 key pair, and that the public and private keys
have the correct expected lengths (32 and 64 bytes respectively).
*/
func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	if len(pub) != 32 {
		t.Errorf("Expected public key length 32, got %d", len(pub))
	}
	if len(priv) != 64 {
		t.Errorf("Expected private key length 64, got %d", len(priv))
	}
}

/*
TestSignAndVerify verifies the standard happy-path for digital signatures.
It generates a key pair, signs a message, checks that the signature length
is 64 bytes, and ensures that the signature is successfully verified
using the corresponding public key.
*/
func TestSignAndVerify(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	msg := []byte("hello valence")
	sig := Sign(priv, msg)
	
	if len(sig) != 64 {
		t.Errorf("Expected signature length 64, got %d", len(sig))
	}
	
	valid := Verify(pub, msg, sig)
	if !valid {
		t.Error("Expected signature to be valid")
	}
}

/*
TestVerifyTamperedMessage ensures that any alteration to the original
message after it has been signed will cause the signature verification
to fail. This guarantees the integrity of the data.
*/
func TestVerifyTamperedMessage(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	msg := []byte("hello valence")
	sig := Sign(priv, msg)
	
	tamperedMsg := []byte("hello valence!")
	valid := Verify(pub, tamperedMsg, sig)
	if valid {
		t.Error("Expected tampered message signature to be invalid")
	}
}

/*
TestVerifyTamperedSignature ensures that any modification to the signature
itself will cause the verification to fail, preventing attackers from
forging or altering signatures.
*/
func TestVerifyTamperedSignature(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	msg := []byte("hello valence")
	sig := Sign(priv, msg)
	
	sig[0] ^= 0xFF // Tamper with signature
	valid := Verify(pub, msg, sig)
	if valid {
		t.Error("Expected tampered signature to be invalid")
	}
}

/*
TestVerifyWrongKey ensures that a signature cannot be verified using a
different public key than the one corresponding to the private key used
for signing. This prevents identity spoofing.
*/
func TestVerifyWrongKey(t *testing.T) {
	_, privA, _ := GenerateKeyPair()
	pubB, _, _ := GenerateKeyPair()
	
	msg := []byte("hello valence")
	sig := Sign(privA, msg)
	
	valid := Verify(pubB, msg, sig)
	if valid {
		t.Error("Expected signature verified with wrong public key to be invalid")
	}
}

/*
TestAddressFromPublicKey verifies that wallet address generation is
deterministic. Hashing the same public key multiple times must always
yield the exact same 64-character hexadecimal address string.
*/
func TestAddressFromPublicKey(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	addr1 := AddressFromPublicKey(pub)
	addr2 := AddressFromPublicKey(pub)
	
	if addr1 != addr2 {
		t.Error("Expected identical addresses for the same public key")
	}
	if len(addr1) != 64 { // hex string of sha256
		t.Errorf("Expected address length 64, got %d", len(addr1))
	}
}

/*
TestAddressDifferentKeys verifies that generating addresses for two
different public keys results in two completely distinct addresses,
ensuring collision resistance and uniqueness.
*/
func TestAddressDifferentKeys(t *testing.T) {
	pubA, _, _ := GenerateKeyPair()
	pubB, _, _ := GenerateKeyPair()
	
	addrA := AddressFromPublicKey(pubA)
	addrB := AddressFromPublicKey(pubB)
	
	if addrA == addrB {
		t.Error("Expected different addresses for different public keys")
	}
}
