package block

import (
	"blockchain-simulator/internal/wallet"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
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
		Transactions: []Transaction{{Sender: "COINBASE", Recipient: "Genesis", Amount: 0, Signature: []byte("0")}},
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

	// add coinbase transaction for reward miner
	if len(b.Transactions) == 0 || b.Transactions[0].Sender != "COINBASE" {
		coinbaseTx := Transaction{Sender: "COINBASE", Recipient: "Miner", Amount: 50, Signature: []byte("0")}
		b.Transactions = append([]Transaction{coinbaseTx}, b.Transactions...)
	}

	extraNonce := 0

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
		extraNonce++
		b.Transactions[0].Signature = []byte(fmt.Sprintf("%d", extraNonce))
	}
}

// DoubleHashBytes performs a double SHA-256 hash on raw bytes.
func DoubleHashBytes(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:]
}

// returns double sha256 hash of the transaction data(without signature)
func (tx *Transaction) Hash() []byte {
	record := fmt.Sprintf("%s|%s|%d|%d", tx.Sender, tx.Recipient, tx.Amount, tx.Sequence)
	return DoubleHashBytes([]byte(record))
}

// signs the transaction hash using the private key
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.Hash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return err
	}

	// Enforce Low-S for malleability protection
	N := elliptic.P256().Params().N
	halfN := new(big.Int).Div(N, big.NewInt(2))
	if s.Cmp(halfN) > 0 {
		s.Sub(N, s) // s = N - s
	}

	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Ensure R and S are exactly 32 bytes to form a fixed 64-byte signature
	signature := make([]byte, 64)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	tx.Signature = signature
	return nil
}

// Verify checks if the transaction signature is valid
func (tx *Transaction) Verify() bool {
	// System-generated transactions don't need signatures
	if tx.Sender == "COINBASE" || tx.Sender == "FAUCET" {
		return true
	}

	pubKey := wallet.BytesToPublicKey(tx.PublicKey)
	if pubKey == nil || len(tx.Signature) != 64 {
		return false
	}

	// 1. Check if the public key actually belongs to the sender!
	senderAddress := wallet.AddressFromPublicKey(tx.PublicKey)
	if senderAddress != tx.Sender {
		return false
	}

	r := big.Int{}
	s := big.Int{}
	r.SetBytes(tx.Signature[:32])
	s.SetBytes(tx.Signature[32:])

	// 2. Enforce Low-S (Reject malleable signatures)
	N := elliptic.P256().Params().N
	halfN := new(big.Int).Div(N, big.NewInt(2))
	if s.Cmp(halfN) > 0 {
		return false // Signature must be in lower half of the curve
	}

	hash := tx.Hash()
	return ecdsa.Verify(pubKey, hash, &r, &s)
}
