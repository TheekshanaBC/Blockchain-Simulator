package storage

import (
	"blockchain-simulator/internal/chain"
	"encoding/json"
	"os"
	"path/filepath"
)

func SaveChain(c *chain.Chain, filename string) error {
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil { // 0644 read,write permission for owner, read-only for others.
		return err
	}

	return os.Rename(tempFile, filename)
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

	if c.RetargetWindow < 2 {
		c.RetargetWindow = 2
	}
	if c.MinDifficulty > c.MaxDifficulty {
		c.MinDifficulty, c.MaxDifficulty = c.MaxDifficulty, c.MinDifficulty
	}

	return &c, nil
}
