package block

import (
	"fmt"
	"strings"
)

type Transaction struct {
	Sender     string `json:"sender"`
	Recipient  string `json:"recipient"`
	Amount     uint64 `json:"amount"`
	ExtraNonce int    `json:"extra_nonce"`
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

const GenesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"

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
		Transactions: []Transaction{{Sender: "COINBASE", Recipient: "Genesis", Amount: 0, ExtraNonce: 0}},
	}
	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	block.Hash = block.CalculateHash()
	return block
}

// calculate hash for a block
func (b *Block) CalculateHash() string {
	record := fmt.Sprintf("%d|%s|%s|%d|%d|%d", b.Height, b.Header.PrevHash, b.Header.MerkleRoot, b.Header.Timestamp, b.Header.Difficulty, b.Header.Nonce)
	doubleHash := doubleSHA256(record)
	return doubleHash
}

// proof of work algorithm
func (b *Block) Mine(difficulty int) {
	b.Header.Difficulty = difficulty
	b.Header.MerkleRoot = CalculateMerkleRoot(b.Transactions)
	target := strings.Repeat("0", difficulty)

	// add coinbase transaction for reward minor
	if len(b.Transactions) == 0 || b.Transactions[0].Sender != "COINBASE" {
		coinbaseTx := Transaction{Sender: "COINBASE", Recipient: "Miner", Amount: 50, ExtraNonce: 0}
		b.Transactions = append([]Transaction{coinbaseTx}, b.Transactions...)
	}

	for {
		b.Header.MerkleRoot = CalculateMerkleRoot(b.Transactions) // recalculate the merkle root for updated extra nonce
		for {
			b.Hash = b.CalculateHash()

			if strings.HasPrefix(b.Hash, target) {
				return
			}
			if b.Header.Nonce == 4294967295 { // max value for uint32
				b.Header.Nonce = 0
				break // go out of the inner loop if hash is not found for every possible nonce
			}

			b.Header.Nonce++
		}
		b.Transactions[0].ExtraNonce++
	}

}
