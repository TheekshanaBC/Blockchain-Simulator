package main

import (
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/cli"
)

func main() {
	difficulty := 3

	myChain := chain.NewChain(difficulty)
	cli.StartCLI(myChain)
}
