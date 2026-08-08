package ledger

import (
	"errors"
	"fmt"
	"strings"
	"valence/internal/block"
)

// MaxTransactionAmount is 1,000,000 VCN
const MaxTransactionAmount = 1_000_000 * block.ElectronsPerVCN

const MaxFaucetRequest int64 = 1000 * block.ElectronsPerVCN
const MaxLifetimeFaucetPerAddress int64 = 5000 * block.ElectronsPerVCN

func CalculateBalances(chain []*block.Block) map[string]int64 {
	balances := make(map[string]int64)

	for _, b := range chain {
		for _, tx := range b.Transactions {
			if tx.Amount == 0 {
				continue
			}

			if !block.IsSystemAddress(tx.Sender) {
				balances[tx.Sender] -= tx.Amount
			}
			balances[tx.Recipient] += tx.Amount
		}
	}
	return balances
}

// CalculateAvailableBalances returns the balance available to spend (ledger minus pending outbounds)
func CalculateAvailableBalances(chain []*block.Block, pendingPool []block.Transaction) map[string]int64 {
	balances := CalculateBalances(chain)

	// deduct the pending pool transactions to prevent double spending
	for _, tx := range pendingPool {
		if !block.IsSystemAddress(tx.Sender) {
			balances[tx.Sender] -= tx.Amount
		}
	}
	return balances
}

// CalculateSequences returns the highest sequence number used by each address in the blockchain
func CalculateSequences(chain []*block.Block) map[string]uint64 {
	sequences := make(map[string]uint64)
	for _, b := range chain {
		for _, tx := range b.Transactions {
			if !block.IsSystemAddress(tx.Sender) {
				if tx.Sequence > sequences[tx.Sender] {
					sequences[tx.Sender] = tx.Sequence
				}
			}
		}
	}
	return sequences
}

// CalculatePendingSequences returns the highest sequence number considering both blockchain and pending pool
func CalculatePendingSequences(chain []*block.Block, pendingPool []block.Transaction) map[string]uint64 {
	sequences := CalculateSequences(chain)
	for _, tx := range pendingPool {
		if !block.IsSystemAddress(tx.Sender) {
			if tx.Sequence > sequences[tx.Sender] {
				sequences[tx.Sender] = tx.Sequence
			}
		}
	}
	return sequences
}

func ValidateTransaction(tx block.Transaction, balances map[string]int64, sequences map[string]uint64, faucetReceived map[string]int64) error {
	if tx.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if tx.Amount > MaxTransactionAmount {
		return errors.New("amount exceeds maximum allowed transaction size")
	}
	if strings.TrimSpace(tx.Recipient) == "" {
		return errors.New("recipient address cannot be empty")
	}

	if tx.Recipient == block.SystemAddressCoinbase {
		return errors.New("cannot send funds to coinbase address")
	}

	if tx.Sender == block.SystemAddressFaucet {
		if tx.Amount > MaxFaucetRequest {
			return fmt.Errorf("faucet request exceeds maximum allowed limit per request (%d)", MaxFaucetRequest)
		}
		if faucetReceived[tx.Recipient]+tx.Amount > MaxLifetimeFaucetPerAddress {
			return fmt.Errorf("lifetime faucet limit exceeded for address (Max: %d)", MaxLifetimeFaucetPerAddress)
		}
	} else if !block.IsSystemAddress(tx.Sender) {
		if !tx.Verify() {
			return errors.New("invalid transaction signature")
		}

		// Sequence Validation (Replay Protection)
		expectedSeq := sequences[tx.Sender] + 1
		if tx.Sequence != expectedSeq {
			return fmt.Errorf("invalid sequence number: expected %d, got %d", expectedSeq, tx.Sequence)
		}

		if balances[tx.Sender] < tx.Amount {
			return fmt.Errorf("insufficent funds: need %d, but have %d", tx.Amount, balances[tx.Sender])
		}
	}
	return nil
}

// ValidateTransactions validates a batch of transactions sequentially against the current chain and pending pool state.
func ValidateTransactions(txs []block.Transaction, chain []*block.Block, pendingPool []block.Transaction) error {
	balances := CalculateAvailableBalances(chain, pendingPool)
	sequences := CalculatePendingSequences(chain, pendingPool)
	faucetReceived := make(map[string]int64)

	// Pre-populate faucetReceived from chain and pending pool
	for _, b := range chain {
		for _, tx := range b.Transactions {
			if tx.Sender == block.SystemAddressFaucet {
				faucetReceived[tx.Recipient] += tx.Amount
			}
		}
	}
	for _, tx := range pendingPool {
		if tx.Sender == block.SystemAddressFaucet {
			faucetReceived[tx.Recipient] += tx.Amount
		}
	}

	for _, tx := range txs {
		if err := ValidateTransaction(tx, balances, sequences, faucetReceived); err != nil {
			return err
		}
		// Update state for subsequent transactions in this batch
		if !block.IsSystemAddress(tx.Sender) {
			balances[tx.Sender] -= tx.Amount
			sequences[tx.Sender] = tx.Sequence
		}
		balances[tx.Recipient] += tx.Amount
		if tx.Sender == block.SystemAddressFaucet {
			faucetReceived[tx.Recipient] += tx.Amount
		}
	}
	return nil
}

// FilterValidTransactions returns only the transactions from the pending pool that are valid against the current chain state.
func FilterValidTransactions(pendingPool []block.Transaction, chain []*block.Block) []block.Transaction {
	balances := CalculateAvailableBalances(chain, []block.Transaction{})
	sequences := CalculatePendingSequences(chain, []block.Transaction{})
	faucetReceived := make(map[string]int64)

	for _, b := range chain {
		for _, tx := range b.Transactions {
			if tx.Sender == block.SystemAddressFaucet {
				faucetReceived[tx.Recipient] += tx.Amount
			}
		}
	}

	var validPool []block.Transaction
	for _, tx := range pendingPool {
		if tx.Sender == block.SystemAddressFaucet {
			if tx.Recipient == block.SystemAddressCoinbase {
				continue
			}
			if err := ValidateTransaction(tx, balances, sequences, faucetReceived); err == nil {
				faucetReceived[tx.Recipient] += tx.Amount
				validPool = append(validPool, tx)
			}
		} else if !block.IsSystemAddress(tx.Sender) {
			if err := ValidateTransaction(tx, balances, sequences, faucetReceived); err == nil {
				balances[tx.Sender] -= tx.Amount
				balances[tx.Recipient] += tx.Amount
				sequences[tx.Sender] = tx.Sequence
				validPool = append(validPool, tx)
			}
		}
	}
	return validPool
}
