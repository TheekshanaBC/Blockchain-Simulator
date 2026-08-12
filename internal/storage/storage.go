package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

	tempFile := fmt.Sprintf("%s.tmp.%d", filename, time.Now().UnixNano())
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

func CleanupTempFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if matched, _ := filepath.Match("*.tmp.*", entry.Name()); matched {
				os.Remove(filepath.Join(dir, entry.Name()))
			}
			// Also clean up keystore temp files (which end with .tmp)
			if matched, _ := filepath.Match("*.tmp", entry.Name()); matched {
				os.Remove(filepath.Join(dir, entry.Name()))
			}
		}
	}
	return nil
}
