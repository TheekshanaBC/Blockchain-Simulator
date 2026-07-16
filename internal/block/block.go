package block

import (
	"fmt"
)

type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Sequence  uint64 `json:"sequence"`
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
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

const MiningReward int64 = 50
const GenesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"

const (
	SystemAddressCoinbase = "COINBASE"
	SystemAddressFaucet   = "FAUCET"
)

func IsSystemAddress(addr string) bool {
	return addr == SystemAddressCoinbase || addr == SystemAddressFaucet
}

// create and return the first block of the blockchain
func NewGenesisBlock() *Block {
	block := &Block{
		Header: BlockHeader{
			PrevHash:   GenesisPrevHash,
			Timestamp:  1700000000,
			Difficulty: 0,
			Nonce:      0,
		},
		Height:       0,
		Transactions: []Transaction{{Sender: SystemAddressCoinbase, Recipient: "Genesis", Amount: 0, Signature: []byte("0")}},
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


