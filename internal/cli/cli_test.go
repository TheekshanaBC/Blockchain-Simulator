package cli

import (
	"blockchain-simulator/internal/chain"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCLICharacterization(t *testing.T) {
	// Create a dummy chain
	c := chain.NewChain(2, 10, 60, 1, 5)

	// Mock stdin and stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()
	os.Stdin = r
	os.Stdout = stdoutW

	// Define the scripted input
	inputCommands := []string{
		"help",
		"createwallet test1",
		"mywallet",
		"wallets",
		"faucet 100",
		"pool",
		"mine",
		"balances",
		"unknown_command",
		"exit",
	}

	go func() {
		for _, cmd := range inputCommands {
			w.Write([]byte(cmd + "\n"))
		}
		w.Close()
	}()

	// Run CLI (it should return when stdin is closed)
	StartCLI(c)

	stdoutW.Close()

	var buf bytes.Buffer
	io.Copy(&buf, stdoutR)

	output := buf.String()

	// Perform basic characterization checks
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("expected output to contain 'Available Commands'")
	}
	if !strings.Contains(output, "Wallet 'test1' created and saved successfully!") {
		t.Errorf("expected output to contain wallet creation success message")
	}
	if !strings.Contains(output, "Unknown Command") {
		t.Errorf("expected output to handle unknown commands")
	}
}
