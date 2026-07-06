package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Block struct {
	Height       int
	Timestamp    int64
	Transactions []string
	PrevHash     string
	Nonce        int
	Hash         string
}

const GenesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"

func NewGenesisBlock() *Block {
	block := &Block{
		Height:       0,
		Timestamp:    time.Now().Unix(),
		Transactions: []string{"Genesis Block"},
		PrevHash:     GenesisPrevHash,
		Nonce:        0,
	}
	block.Hash = block.CalculateHash()
	return block
}

func (b *Block) CalculateHash() string {
	txData := strings.Join(b.Transactions, "") // In real project Merkle tree data structure can be used
	record := fmt.Sprintf("%d%d%s%s%d", b.Height, b.Timestamp, txData, b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	hashedBytes := h.Sum(nil)
	return hex.EncodeToString(hashedBytes)
}

func (b *Block) Mine(difficulty int) {
	target := strings.Repeat("0", difficulty)
	for {
		b.Hash = b.CalculateHash()

		if strings.HasPrefix(b.Hash, target) {
			break
		} else {
			b.Nonce++
		}
	}
}
