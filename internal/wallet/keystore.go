package wallet

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
)

type walletData struct {
	Address    string `json:"address,omitempty"`
	PublicKey  []byte `json:"public_key,omitempty"`
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

	if _, exists := keystore[name]; exists {
		return fmt.Errorf("wallet '%s' already exists", name)
	}

	keystore[name] = walletData{
		Address:    w.Address(),
		PublicKey:  w.PublicKey,
		PrivateKey: w.PrivateKey.Seed(),
	}

	file, err := json.MarshalIndent(keystore, "", "  ")
	if err != nil {
		return err
	}
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, file, 0600); err != nil {
		return err
	}
	return os.Rename(tempFile, filename)
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

	if len(data.PrivateKey) != ed25519.SeedSize {
		return nil, fmt.Errorf("wallet '%s' has invalid seed length: expected %d, got %d", name, ed25519.SeedSize, len(data.PrivateKey))
	}

	privKey := ed25519.NewKeyFromSeed(data.PrivateKey)
	pubKey := privKey.Public().(ed25519.PublicKey)

	return &Wallet{
		PrivateKey: privKey,
		PublicKey:  pubKey,
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
		if len(data.PrivateKey) != ed25519.SeedSize {
			// Skip corrupt wallets so the node can still start with the healthy ones
			continue
		}
		privKey := ed25519.NewKeyFromSeed(data.PrivateKey)
		pubKey := privKey.Public().(ed25519.PublicKey)
		wallets[name] = &Wallet{
			PrivateKey: privKey,
			PublicKey:  pubKey,
		}
	}
	return wallets, nil
}
