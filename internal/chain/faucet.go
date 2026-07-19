package chain

import (
	"blockchain-simulator/internal/block"
	"fmt"
	"strings"
)

const MaxFaucetRequest int64 = 1000
const MaxLifetimeFaucetPerAddress int64 = 5000

// RequestFaucetFunds creates and adds a system-approved FAUCET transaction to the pending pool.
// This bypasses the AddTransaction sender check but enforces its own limits.
func (c *Chain) RequestFaucetFunds(recipient string, amount int64) error {
	if strings.TrimSpace(recipient) == "" {
		return fmt.Errorf("recipient address cannot be empty")
	}
	if amount <= 0 {
		return fmt.Errorf("faucet amount must be strictly positive")
	}
	if amount > MaxFaucetRequest {
		return fmt.Errorf("faucet request exceeds maximum allowed limit per request (%d)", MaxFaucetRequest)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate total amount already given to this address by the faucet
	var totalReceived int64 = 0
	for _, b := range c.blocks {
		for _, tx := range b.Transactions {
			if tx.Sender == block.SystemAddressFaucet && tx.Recipient == recipient {
				totalReceived += tx.Amount
			}
		}
	}
	for _, tx := range c.pendingPool {
		if tx.Sender == block.SystemAddressFaucet && tx.Recipient == recipient {
			totalReceived += tx.Amount
		}
	}

	if totalReceived+amount > MaxLifetimeFaucetPerAddress {
		return fmt.Errorf("lifetime faucet limit exceeded for address (max: %d, already received: %d)", MaxLifetimeFaucetPerAddress, totalReceived)
	}

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: recipient,
		Amount:    amount,
		// FAUCET transactions don't need a signature or sequence
	}

	c.pendingPool = append(c.pendingPool, tx)
	return nil
}
