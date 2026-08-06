package wallet

import (
	"crypto/ed25519"
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
