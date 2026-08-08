package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"valence/internal/chain"
)

func SaveChain(c *chain.Chain, filename string) error {
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0750); err != nil {
		return err
	}

	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil { // 0600 read,write permission for owner.
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
	if c.MaxTxPerBlock <= 0 {
		c.MaxTxPerBlock = 10
	}

	return &c, nil
}
