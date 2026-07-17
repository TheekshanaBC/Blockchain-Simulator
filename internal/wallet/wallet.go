package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
)

type Wallet struct {
	PrivateKey     *ecdsa.PrivateKey
	PublicKeyBytes []byte
}

// generate brand new random private key and derive matching public key. return both wrapped in a Wallet
func NewWallet() *Wallet {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pubKeyBytes := elliptic.Marshal(privateKey.PublicKey.Curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)

	return &Wallet{
		PrivateKey:     privateKey,
		PublicKeyBytes: pubKeyBytes,
	}
}

// takes raw public key bytes and turn them back into usable ecdsa.Public key object
func BytesToPublicKey(pubKeyBytes []byte) *ecdsa.PublicKey {
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, pubKeyBytes)
	if x == nil {
		log.Println("Error parsing public key: invalid bytes")
		return nil
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
}

// generates a simple address string from public key bytes
func AddressFromPublicKey(pubKeyBytes []byte) string {
	hash := sha256.Sum256(pubKeyBytes)
	return hex.EncodeToString(hash[:])
}
