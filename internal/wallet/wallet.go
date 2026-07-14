package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

type walletData struct {
	PrivateKey []byte `json:"private_key"`
}

type Keystore map[string]walletData

func loadRawKeystore(filename string) (Keystore, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// if it's the old format (just one wallet without names), it will fail to unmarshal into Keystore
	// so we can just return empty and let it be overwritten or ignored.
	var keystore Keystore
	err = json.Unmarshal(file, &keystore)
	if err != nil {
		return nil, err
	}
	return keystore, nil
}

// SaveToKeystore saves a specific wallet to the keystore file under a given name
func SaveToKeystore(filename string, name string, w *Wallet) error {
	keystore, err := loadRawKeystore(filename)
	if err != nil {
		keystore = make(Keystore)
	}

	privKeyBytes, err := x509.MarshalECPrivateKey(w.PrivateKey)
	if err != nil {
		return err
	}
	keystore[name] = walletData{PrivateKey: privKeyBytes}

	file, err := json.MarshalIndent(keystore, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, file, 0644)
}

// LoadFromKeystore loads a specific wallet from the keystore
func LoadFromKeystore(filename string, name string) (*Wallet, error) {
	keystore, err := loadRawKeystore(filename)
	if err != nil {
		return nil, err
	}

	data, exists := keystore[name]
	if !exists {
		return nil, fmt.Errorf("wallet '%s' not found", name)
	}

	privKey, err := x509.ParseECPrivateKey(data.PrivateKey)
	if err != nil {
		return nil, err
	}
	pubKeyBytes := elliptic.Marshal(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)
	return &Wallet{
		PrivateKey:     privKey,
		PublicKeyBytes: pubKeyBytes,
	}, nil
}

// GetAllWallets returns all wallets in the keystore
func GetAllWallets(filename string) (map[string]*Wallet, error) {
	keystore, err := loadRawKeystore(filename)
	if err != nil {
		return nil, err
	}

	wallets := make(map[string]*Wallet)
	for name, data := range keystore {
		privKey, err := x509.ParseECPrivateKey(data.PrivateKey)
		if err == nil {
			pubKeyBytes := elliptic.Marshal(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)
			wallets[name] = &Wallet{
				PrivateKey:     privKey,
				PublicKeyBytes: pubKeyBytes,
			}
		}
	}
	return wallets, nil
}
