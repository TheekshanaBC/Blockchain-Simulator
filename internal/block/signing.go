package block

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"valence/internal/crypto"
)

// performs a double SHA-256 hash on raw bytes.
func DoubleHashBytes(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:]
}

// returns double sha256 hash of the transaction data(without signature)
func (tx *Transaction) Hash() []byte {
	record := fmt.Sprintf("%d:%s|%d:%s|%d|%d|%d", len(tx.Sender), tx.Sender, len(tx.Recipient), tx.Recipient, tx.Amount, tx.Sequence, tx.Timestamp)
	return DoubleHashBytes([]byte(record))
}

func (tx *Transaction) ComputeID() {
	tx.ID = hex.EncodeToString(tx.Hash())
}

// signs the transaction hash using the private key
func (tx *Transaction) Sign(privKey ed25519.PrivateKey) error {
	hash := tx.Hash()
	tx.Signature = crypto.Sign(privKey, hash)
	tx.PublicKey = privKey[32:]
	return nil
}

// checks if the transaction signature is valid
func (tx *Transaction) Verify() bool {
	// System-generated transactions don't need signatures
	if IsSystemAddress(tx.Sender) {
		return true
	}

	if len(tx.PublicKey) != 32 || len(tx.Signature) != 64 {
		return false
	}

	// 1. Check if the public key actually belongs to the sender!
	senderAddress := crypto.AddressFromPublicKey(tx.PublicKey)
	if senderAddress != tx.Sender {
		return false
	}

	hash := tx.Hash()
	return crypto.Verify(tx.PublicKey, hash, tx.Signature)
}
