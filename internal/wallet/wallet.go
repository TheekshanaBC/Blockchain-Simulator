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

// AddressFromPublicKey generates a simple address string from public key bytes
func AddressFromPublicKey(pubKeyBytes []byte) string {
	hash := sha256.Sum256(pubKeyBytes)
	return hex.EncodeToString(hash[:])
}


