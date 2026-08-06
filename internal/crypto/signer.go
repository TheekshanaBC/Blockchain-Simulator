package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateKeyPair creates a new Ed25519 key pair.
// Returns (publicKey, privateKey). Public key is 32 bytes. Private key is 64 bytes.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// Sign signs a message with an Ed25519 private key.
// Returns a 64-byte signature. Ed25519 signatures are deterministic.
func Sign(privateKey ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(privateKey, message)
}

// Verify checks an Ed25519 signature against a public key and message.
func Verify(publicKey ed25519.PublicKey, message []byte, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}

// AddressFromPublicKey derives a wallet address from an Ed25519 public key.
// Address = hex(SHA256(publicKeyBytes)).
func AddressFromPublicKey(publicKey ed25519.PublicKey) string {
	hash := sha256.Sum256(publicKey)
	return hex.EncodeToString(hash[:])
}
