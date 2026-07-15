package block

import (
	"blockchain-simulator/internal/wallet"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
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

// proof of work algorithm
func (b *Block) Mine(difficulty int) {
	b.Header.Difficulty = difficulty
	b.Header.MerkleRoot = CalculateMerkleRoot(b.Transactions)
	target := strings.Repeat("0", difficulty)

	// add coinbase transaction for reward miner
	if len(b.Transactions) == 0 || b.Transactions[0].Sender != SystemAddressCoinbase {
		coinbaseTx := Transaction{Sender: SystemAddressCoinbase, Recipient: "Miner", Amount: MiningReward, Signature: []byte("0")}
		b.Transactions = append([]Transaction{coinbaseTx}, b.Transactions...)
	}

	extraNonce := 0

	for {
		b.Header.MerkleRoot = CalculateMerkleRoot(b.Transactions) // recalculate the merkle root for updated extra nonce
		numWorkers := runtime.NumCPU()
		ctx, cancel := context.WithCancel(context.Background())

		resultChan := make(chan struct {
			nonce uint32
			hash  string
		})

		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for nonce := uint32(workerID); ; nonce += uint32(numWorkers) {
					select {
					case <-ctx.Done():
						return
					default:
					}

					hash := calculateHashForNonce(b, nonce)
					if strings.HasPrefix(hash, target) {
						select {
						case resultChan <- struct {
							nonce uint32
							hash  string
						}{nonce, hash}:
						case <-ctx.Done():
						}
						return
					}

					if nonce > 4294967295-uint32(numWorkers) {
						return
					}
				}
			}(i)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		if res, ok := <-resultChan; ok {
			b.Header.Nonce = res.nonce
			b.Hash = res.hash
			cancel()
			return
		}

		cancel()
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
	record := fmt.Sprintf("%d:%s|%d:%s|%d|%d", len(tx.Sender), tx.Sender, len(tx.Recipient), tx.Recipient, tx.Amount, tx.Sequence)
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
	if IsSystemAddress(tx.Sender) {
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
