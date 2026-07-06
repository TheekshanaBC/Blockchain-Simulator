package block

import "time"

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
	return &Block{
		Height:       0,
		Timestamp:    time.Now().Unix(),
		Transactions: []string{"Genesis Block"},
		PrevHash:     GenesisPrevHash,
		Nonce:        0,
		Hash:         "",
	}
}
