package chain

import (
	"fmt"
	"strings"
	"time"
	"valence/internal/block"
	"valence/internal/ledger"
	"valence/internal/wallet"
)

// CreateFaucetTx creates a cryptographically signed faucet transaction.
// It enforces limits against the blockchain and signs with the given wallet.
func (c *Chain) CreateFaucetTx(recipient string, amount int64, fee int64, faucetWallet *wallet.Wallet, sequence uint64, balance int64, pendingPool []block.Transaction) (block.Transaction, error) {
	if strings.TrimSpace(recipient) == "" {
		return block.Transaction{}, fmt.Errorf("recipient address cannot be empty")
	}
	if recipient == block.SystemAddressCoinbase {
		return block.Transaction{}, fmt.Errorf("cannot request faucet funds for COINBASE address")
	}
	if amount <= 0 {
		return block.Transaction{}, fmt.Errorf("faucet amount must be strictly positive")
	}
	if amount > ledger.MaxFaucetRequest {
		return block.Transaction{}, fmt.Errorf("faucet request exceeds maximum allowed limit per request (%d)", ledger.MaxFaucetRequest)
	}
	if balance < amount + fee {
		return block.Transaction{}, fmt.Errorf("faucet is out of funds")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalReceived int64 = 0
	for _, b := range c.blocks {
		for _, tx := range b.Transactions {
			if tx.Sender == faucetWallet.Address() && tx.Recipient == recipient {
				totalReceived += tx.Amount
			}
		}
	}
	for _, tx := range pendingPool {
		if tx.Sender == faucetWallet.Address() && tx.Recipient == recipient {
			totalReceived += tx.Amount
		}
	}

	if totalReceived+amount > ledger.MaxLifetimeFaucetPerAddress {
		return block.Transaction{}, fmt.Errorf("lifetime faucet limit exceeded for address (max: %d, already received: %d)", ledger.MaxLifetimeFaucetPerAddress, totalReceived)
	}

	tx := block.Transaction{
		Sender:    faucetWallet.Address(),
		Recipient: recipient,
		Amount:    amount,
		Fee:       fee,
		Timestamp: time.Now().UnixNano(),
		Sequence:  sequence,
		PublicKey: faucetWallet.PublicKey,
	}
	tx.Sign(faucetWallet.PrivateKey)
	tx.ComputeID()

	return tx, nil
}
