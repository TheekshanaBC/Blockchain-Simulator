package chain

import (
	"fmt"
	"strings"
	"valence/internal/block"
)

const MaxFaucetRequest int64 = 1000 * block.ElectronsPerVCN
const MaxLifetimeFaucetPerAddress int64 = 5000 * block.ElectronsPerVCN

// CreateFaucetTx creates a system-approved FAUCET transaction.
// This bypasses the sender signature check but enforces its own limits against the blockchain.
func (c *Chain) CreateFaucetTx(recipient string, amount int64, pendingPool []block.Transaction) (block.Transaction, error) {
	if strings.TrimSpace(recipient) == "" {
		return block.Transaction{}, fmt.Errorf("recipient address cannot be empty")
	}
	if recipient == block.SystemAddressCoinbase {
		return block.Transaction{}, fmt.Errorf("cannot request faucet funds for COINBASE address")
	}
	if amount <= 0 {
		return block.Transaction{}, fmt.Errorf("faucet amount must be strictly positive")
	}
	if amount > MaxFaucetRequest {
		return block.Transaction{}, fmt.Errorf("faucet request exceeds maximum allowed limit per request (%d)", MaxFaucetRequest)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalReceived int64 = 0
	for _, b := range c.blocks {
		for _, tx := range b.Transactions {
			if tx.Sender == block.SystemAddressFaucet && tx.Recipient == recipient {
				totalReceived += tx.Amount
			}
		}
	}
	for _, tx := range pendingPool {
		if tx.Sender == block.SystemAddressFaucet && tx.Recipient == recipient {
			totalReceived += tx.Amount
		}
	}

	if totalReceived+amount > MaxLifetimeFaucetPerAddress {
		return block.Transaction{}, fmt.Errorf("lifetime faucet limit exceeded for address (max: %d, already received: %d)", MaxLifetimeFaucetPerAddress, totalReceived)
	}

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: recipient,
		Amount:    amount,
	}
	tx.ComputeID()

	return tx, nil
}
