package wallet

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"valence/internal/crypto"
)

type Wallet struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// generate brand new random private key and derive matching public key. return both wrapped in a Wallet
func NewWallet() *Wallet {
	pub, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	return &Wallet{
		PrivateKey: priv,
		PublicKey:  pub,
	}
}

func (w *Wallet) Address() string {
	return crypto.AddressFromPublicKey(w.PublicKey)
}

// WalletFromBase64 reconstructs a wallet from a base64 encoded 32-byte seed
func WalletFromBase64(b64 string) (*Wallet, error) {
	seed, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid seed length: expected %d, got %d", ed25519.SeedSize, len(seed))
	}
	
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)
	
	return &Wallet{
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}, nil
}
