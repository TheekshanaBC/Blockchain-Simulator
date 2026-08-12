package block

import (
	"context"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// proof of work algorithm
func (b *Block) Mine(ctx context.Context, difficulty int) {
	if difficulty < 0 {
		difficulty = 0
	}
	b.Header.Difficulty = difficulty
	b.Header.MerkleRoot = CalculateMerkleRoot(b.Transactions)
	target := strings.Repeat("0", difficulty)

	// add coinbase transaction for reward miner
	if len(b.Transactions) == 0 || b.Transactions[0].Sender != SystemAddressCoinbase {
		coinbaseTx := Transaction{Sender: SystemAddressCoinbase, Recipient: "Miner", Amount: MiningReward, Signature: []byte("0"), Timestamp: time.Now().UnixNano()}
		coinbaseTx.ComputeID()
		b.Transactions = append([]Transaction{coinbaseTx}, b.Transactions...)
	}

	extraNonce := 0

	for {
		b.Header.MerkleRoot = CalculateMerkleRoot(b.Transactions) // recalculate the merkle root for updated extra nonce
		numWorkers := runtime.NumCPU()
		
		// Create an inner context to cancel siblings if one finds the block
		innerCtx, innerCancel := context.WithCancel(ctx)

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

				if math.MaxUint32-startNonce < uint32(workerID) {
					return
				}

				for nonce := startNonce + uint32(workerID); ; nonce += uint32(numWorkers) {
					select {
					case <-innerCtx.Done():
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
						case <-innerCtx.Done():
						}
						return
					}

					if nonce > math.MaxUint32-uint32(numWorkers) {
						return
					}
				}
			}(i)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		select {
		case result, ok := <-resultChan:
			if ok {
				b.Header.Nonce = result.nonce
				b.Hash = result.hash
				innerCancel()
				return
			}
		case <-ctx.Done():
			b.Hash = ""
			innerCancel()
			wg.Wait()
			return
		}

		innerCancel()
		extraNonce++
		b.Transactions[0].Signature = strconv.AppendInt(nil, int64(extraNonce), 10)
		b.Header.Nonce = 0
	}
}
