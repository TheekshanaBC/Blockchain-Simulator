package chain

import (
	"strings"
	"testing"
)

/*
TestRequestFaucetFunds_EmptyRecipient ensures that requesting funds with an
empty or whitespace-only recipient address returns an error.
*/
func TestRequestFaucetFunds_EmptyRecipient(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)
	
	err := c.RequestFaucetFunds("", 100)
	if err == nil || !strings.Contains(err.Error(), "recipient address cannot be empty") {
		t.Errorf("Expected empty recipient error, got: %v", err)
	}

	err = c.RequestFaucetFunds("   ", 100)
	if err == nil || !strings.Contains(err.Error(), "recipient address cannot be empty") {
		t.Errorf("Expected empty recipient error for whitespace, got: %v", err)
	}
}

/*
TestRequestFaucetFunds_NonPositiveAmount ensures that requesting zero or negative
funds from the faucet returns an error.
*/
func TestRequestFaucetFunds_NonPositiveAmount(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)
	
	err := c.RequestFaucetFunds("recipient", 0)
	if err == nil || !strings.Contains(err.Error(), "strictly positive") {
		t.Errorf("Expected non-positive amount error for 0, got: %v", err)
	}

	err = c.RequestFaucetFunds("recipient", -50)
	if err == nil || !strings.Contains(err.Error(), "strictly positive") {
		t.Errorf("Expected non-positive amount error for -50, got: %v", err)
	}
}

/*
TestRequestFaucetFunds_SingleRequestOverLimit ensures that a single request
exceeding MaxFaucetRequest is rejected.
*/
func TestRequestFaucetFunds_SingleRequestOverLimit(t *testing.T) {
	c := NewChain(1, 10, 60, 1, 5)
	
	err := c.RequestFaucetFunds("recipient", MaxFaucetRequest+1)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum allowed limit per request") {
		t.Errorf("Expected over limit error, got: %v", err)
	}
}

/*
TestRequestFaucetFunds_LifetimeLimitExceeded ensures that multiple requests
accumulating to more than MaxLifetimeFaucetPerAddress are rejected, accounting
for both mined blocks and the pending pool.
*/
func TestRequestFaucetFunds_LifetimeLimitExceeded(t *testing.T) {
	c := NewChain(0, 10, 60, 1, 5)
	recipient := "greedy_user"
	
	// MaxLifetimeFaucetPerAddress is 5000, MaxFaucetRequest is 1000

	// 1. Give some funds and mine them
	c.RequestFaucetFunds(recipient, 1000)
	c.RequestFaucetFunds(recipient, 1000)
	c.MinePendingTransactions() // 2000 total received in blocks

	// 2. Add some to pending pool
	c.RequestFaucetFunds(recipient, 1000)
	c.RequestFaucetFunds(recipient, 1000) // 4000 total received across blocks + pending

	// 3. Request exactly the remaining limit (1000) - should pass
	err := c.RequestFaucetFunds(recipient, 1000)
	if err != nil {
		t.Errorf("Expected successful request up to limit, got: %v", err)
	}

	// 4. Request more, should fail due to lifetime limit
	err = c.RequestFaucetFunds(recipient, 1)
	if err == nil || !strings.Contains(err.Error(), "lifetime faucet limit exceeded") {
		t.Errorf("Expected lifetime limit exceeded error, got: %v", err)
	}
}
