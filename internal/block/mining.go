package block

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

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
		startNonce := b.Header.Nonce

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				if uint32(workerID) > 4294967295-startNonce {
					return
				}

				for nonce := startNonce + uint32(workerID); ; nonce += uint32(numWorkers) {
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
		b.Transactions[0].Signature = fmt.Appendf(nil, "%d", extraNonce)
		b.Header.Nonce = 0
	}
}
