package wallet

import (
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
)

type walletData struct {
	PrivateKey []byte `json:"private_key"`
}

type Keystore map[string]walletData

// read and parses the keystore json file
func loadRawKeystore(filename string) (Keystore, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var keystore Keystore
	if err = json.Unmarshal(file, &keystore); err != nil {
		return nil, err
	}
	return keystore, nil
}

// saves a specific wallet to the keystore file under a given name
func SaveToKeystore(filename string, name string, w *Wallet) error {
	keystore, err := loadRawKeystore(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
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

// loads a specific wallet from the keystore
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

// returns all wallets in the keystore
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
