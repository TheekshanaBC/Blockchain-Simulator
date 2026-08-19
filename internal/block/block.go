package block

import (
	"fmt"
)

type Transaction struct {
	ID        string `json:"id"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Fee       int64  `json:"fee"`
	Sequence  uint64 `json:"sequence"`
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
	Timestamp int64  `json:"timestamp"`
}

type BlockHeader struct {
	PrevHash   string `json:"prev_hash"`
	MerkleRoot string `json:"merkle_root"`
	Timestamp  int64  `json:"timestamp"`
	Difficulty int    `json:"difficulty"`
	Nonce      uint32 `json:"nonce"`
}

type Block struct {
	Header       BlockHeader   `json:"header"`
	Height       int           `json:"height"`
	Transactions []Transaction `json:"transactions"`
	Hash         string        `json:"hash"`
}

const MiningReward int64 = 50_000_000_000
const GenesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"

const (
	SystemAddressCoinbase = "VALENCE_COINBASE"
)

// ElectronsPerVCN is the number of electrons in 1 VCN (1 billion)
const ElectronsPerVCN int64 = 1_000_000_000

// BaseFee is the minimum fee for a transaction
const BaseFee int64 = 100

// FeeMultiplier is the additional fee per transaction in the mempool
const FeeMultiplier int64 = 10

func IsSystemAddress(addr string) bool {
	return addr == SystemAddressCoinbase
}

// create and return the first block of the blockchain
func NewGenesisBlock() *Block {
	block := &Block{
		Header: BlockHeader{
			PrevHash:   GenesisPrevHash,
			Timestamp:  1700000000000000000,
			Difficulty: 0,
			Nonce:      0,
		},
		Height: 0,
		// Note: The genesis block rewards the "Genesis" address, which has no corresponding
		// private key. This permanently locks the initial 50 VCN.
		// It also pre-allocates 1,000,000,000 VCN to the predefined Development Faucet wallet address
		// (derived from the faucet private key).
		Transactions: []Transaction{
			{Sender: SystemAddressCoinbase, Recipient: "Genesis", Amount: 50 * ElectronsPerVCN, Signature: []byte("0")},
			{Sender: SystemAddressCoinbase, Recipient: "b2be6b76fa3f8e9d88de9128285f73b1deb13e8e1bd44df24e5423fce0171607", Amount: 1_000_000_000 * ElectronsPerVCN, Signature: []byte("0")},
		},
	}
	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	block.Hash = block.CalculateHash()
	return block
}

func calculateHashForNonce(b *Block, nonce uint32) string {
	record := fmt.Sprintf("%d|%d:%s|%d:%s|%d|%d|%d", b.Height, len(b.Header.PrevHash), b.Header.PrevHash, len(b.Header.MerkleRoot), b.Header.MerkleRoot, b.Header.Timestamp, b.Header.Difficulty, nonce)
	return doubleSHA256(record)
}

// calculate hash for a block
func (b *Block) CalculateHash() string {
	return calculateHashForNonce(b, b.Header.Nonce)
}

func FormatVCN(electrons int64) string {
	return fmt.Sprintf("%.9f VCN", float64(electrons)/float64(ElectronsPerVCN))
}
