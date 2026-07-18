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

/*
TestLoadChain_MissingFile ensures that LoadChain returns an appropriate error
when attempting to load from a non-existent file path.
*/
func TestLoadChain_MissingFile(t *testing.T) {
	_, err := LoadChain("non_existent_file_path_12345.json")
	if err == nil {
		t.Error("Expected error when loading a missing file, got nil")
	}
}

/*
TestLoadChain_CorruptJSON ensures that LoadChain returns a parsing error
when the file exists but contains invalid JSON data.
*/
func TestLoadChain_CorruptJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "corrupt.json")
	os.WriteFile(filename, []byte("{corrupt_json: true, [}"), 0644)

	_, err = LoadChain(filename)
	if err == nil {
		t.Error("Expected JSON unmarshal error for corrupt file, got nil")
	}
}

/*
TestLoadChain_RetargetWindowCorrection ensures that loading an older chain
file with an invalid RetargetWindow (< 2) auto-corrects it to 2.
*/
func TestLoadChain_RetargetWindowCorrection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "retarget.json")
	// Save a chain with RetargetWindow = 1
	validJSON := `{"retarget_window": 1, "min_difficulty": 1, "max_difficulty": 5}`
	os.WriteFile(filename, []byte(validJSON), 0644)

	loadedChain, err := LoadChain(filename)
	if err != nil {
		t.Fatalf("LoadChain failed: %v", err)
	}

	if loadedChain.RetargetWindow != 2 {
		t.Errorf("Expected RetargetWindow to be corrected to 2, got %d", loadedChain.RetargetWindow)
	}
}

/*
TestLoadChain_DifficultySwap ensures that loading a chain with inverted
difficulty bounds (Min > Max) correctly swaps them.
*/
func TestLoadChain_DifficultySwap(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "difficulty.json")
	// Save a chain with min > max
	validJSON := `{"retarget_window": 10, "min_difficulty": 10, "max_difficulty": 2}`
	os.WriteFile(filename, []byte(validJSON), 0644)

	loadedChain, err := LoadChain(filename)
	if err != nil {
		t.Fatalf("LoadChain failed: %v", err)
	}

	if loadedChain.MinDifficulty != 2 || loadedChain.MaxDifficulty != 10 {
		t.Errorf("Expected difficulties to swap (min: 2, max: 10), got (min: %d, max: %d)", loadedChain.MinDifficulty, loadedChain.MaxDifficulty)
	}
}
