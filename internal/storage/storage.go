package storage

import (
	"blockchain-simulator/internal/chain"
	"encoding/json"
	"os"
)

func SaveChain(c *chain.Chain, filename string) error {
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644) // 0644 read,write permission for owner, read-only for others.
}

func LoadChain(filename string) (*chain.Chain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var c chain.Chain

	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
