package storage

import (
	"blockchain-simulator/internal/chain"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadChain(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test_chain.json")

	// Create a dummy chain
	c := chain.NewChain(2, 10, 60, 1, 5)
	
	// Save
	if err := SaveChain(c, filename); err != nil {
		t.Fatalf("SaveChain failed: %v", err)
	}

	// Load
	loadedChain, err := LoadChain(filename)
	if err != nil {
		t.Fatalf("LoadChain failed: %v", err)
	}

	// Simple check, reflect.DeepEqual might be tricky with blocks due to time etc, but NewChain uses empty block basically.
	if !reflect.DeepEqual(c, loadedChain) {
		t.Errorf("loaded chain does not match saved chain.\nGot: %+v\nWant: %+v", loadedChain, c)
	}
}
