package block

import (
	"blockchain-simulator/internal/wallet"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

// performs a double SHA-256 hash on raw bytes.
func DoubleHashBytes(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:]
}

// returns double sha256 hash of the transaction data(without signature)
func (tx *Transaction) Hash() []byte {
	record := fmt.Sprintf("%d:%s|%d:%s|%d|%d", len(tx.Sender), tx.Sender, len(tx.Recipient), tx.Recipient, tx.Amount, tx.Sequence)
	return DoubleHashBytes([]byte(record))
}

// signs the transaction hash using the private key
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.Hash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return err
	}

	// Enforce Low-S for malleability protection
	N := elliptic.P256().Params().N
	halfN := new(big.Int).Div(N, big.NewInt(2))
	if s.Cmp(halfN) > 0 {
		s.Sub(N, s) // s = N - s
	}

	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Ensure R and S are exactly 32 bytes to form a fixed 64-byte signature
	signature := make([]byte, 64)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	tx.Signature = signature
	return nil
}

// checks if the transaction signature is valid
func (tx *Transaction) Verify() bool {
	// System-generated transactions don't need signatures
	if IsSystemAddress(tx.Sender) {
		return true
	}

	pubKey := wallet.BytesToPublicKey(tx.PublicKey)
	if pubKey == nil || len(tx.Signature) != 64 {
		return false
	}

	// 1. Check if the public key actually belongs to the sender!
	senderAddress := wallet.AddressFromPublicKey(tx.PublicKey)
	if senderAddress != tx.Sender {
		return false
	}

	r := big.Int{}
	s := big.Int{}
	r.SetBytes(tx.Signature[:32])
	s.SetBytes(tx.Signature[32:])

	// 2. Enforce Low-S (Reject malleable signatures)
	N := elliptic.P256().Params().N
	halfN := new(big.Int).Div(N, big.NewInt(2))
	if s.Cmp(halfN) > 0 {
		return false // Signature must be in lower half of the curve
	}

	hash := tx.Hash()
	return ecdsa.Verify(pubKey, hash, &r, &s)
}
