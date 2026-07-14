package main

import (
	"blockchain-simulator/internal/block"
	"fmt"
	"time"
)

func main() {
	fmt.Printf("%-12s | %-12s | %-12s\n", "Difficulty", "Time Taken", "Hashes Tried")
	fmt.Println("----------------------------------------------")

	for difficulty := 1; difficulty <= 8; difficulty++ {
		// Create a dummy block
		b := &block.Block{
			Header: block.BlockHeader{
				Timestamp: time.Now().Unix(),
				PrevHash:  "0000000000000000000000000000000000000000000000000000000000000000",
				Nonce:     0,
			},
			Height: 1,
			Transactions: []block.Transaction{
				{Sender: "Alice", Recipient: "Bob", Amount: 10},
			},
		}

		start := time.Now()
		b.Mine(difficulty)
		elapsed := time.Since(start)

		totalHashes := int64(b.Header.Nonce) + 1
		fmt.Printf("%-12d | %-12v | %-12d\n", difficulty, elapsed, totalHashes)
	}
}
