package chain

import (
	"valence/internal/block"
	"strings"
	"testing"
)

/*
TestCreateFaucetTx_EmptyRecipient ensures that requesting funds with an
empty or whitespace-only recipient address returns an error.
*/
func TestCreateFaucetTx_EmptyRecipient(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)

	_, err := c.CreateFaucetTx("", 100)
	if err == nil || !strings.Contains(err.Error(), "recipient address cannot be empty") {
		t.Errorf("Expected empty recipient error, got: %v", err)
	}

	_, err = c.CreateFaucetTx("   ", 100)
	if err == nil || !strings.Contains(err.Error(), "recipient address cannot be empty") {
		t.Errorf("Expected empty recipient error for whitespace, got: %v", err)
	}
}

/*
TestCreateFaucetTx_NonPositiveAmount ensures that requesting zero or negative
funds from the faucet returns an error.
*/
func TestCreateFaucetTx_NonPositiveAmount(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)

	_, err := c.CreateFaucetTx("recipient", 0)
	if err == nil || !strings.Contains(err.Error(), "strictly positive") {
		t.Errorf("Expected non-positive amount error for 0, got: %v", err)
	}

	_, err = c.CreateFaucetTx("recipient", -50)
	if err == nil || !strings.Contains(err.Error(), "strictly positive") {
		t.Errorf("Expected non-positive amount error for -50, got: %v", err)
	}
}

/*
TestCreateFaucetTx_SingleRequestOverLimit ensures that a single request
exceeding MaxFaucetRequest is rejected.
*/
func TestCreateFaucetTx_SingleRequestOverLimit(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)

	_, err := c.CreateFaucetTx("recipient", MaxFaucetRequest+1)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum allowed limit per request") {
		t.Errorf("Expected over limit error, got: %v", err)
	}
}

/*
TestCreateFaucetTx_LifetimeLimitExceeded ensures that multiple requests
accumulating to more than MaxLifetimeFaucetPerAddress are rejected, accounting
for both mined blocks and the pending pool.
*/
func TestCreateFaucetTx_LifetimeLimitExceeded(t *testing.T) {
	c := NewChain(0, 10, 60, 1, 5)
	recipient := "greedy_user"

	// 1. Give some funds and mine them (MaxLifetimeFaucetPerAddress is 5000, MaxFaucetRequest is 1000)
	
	for i := 0; i < 5; i++ {
		fTx, err := c.CreateFaucetTx(recipient, MaxFaucetRequest)
		if err != nil {
			t.Fatalf("Failed to create faucet tx: %v", err)
		}
		c.MineBlock([]block.Transaction{fTx}, "Miner")
	}

	/* 2. After reaching the lifetime limit of 5000 VCN, any subsequent request must be rejected by the system */
	_, err := c.CreateFaucetTx(recipient, 1)
	if err == nil || !strings.Contains(err.Error(), "lifetime faucet limit exceeded") {
		t.Errorf("Expected lifetime limit exceeded error, got: %v", err)
	}
}


