package main

import (
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/cli"
	"blockchain-simulator/internal/storage"
	"fmt"
)

const dbFile = "data/chain.json"

func main() {
	var myChain *chain.Chain
	difficulty := 5

	loadedChain, err := storage.LoadChain(dbFile)
	if err != nil {
		fmt.Println("No existing chain found. Creating a new one...")
		myChain = chain.NewChain(difficulty)
	} else {
		fmt.Println("Loaded existing blockchain from disk.")
		myChain = loadedChain
	}

	cli.StartCLI(myChain)

	fmt.Println("Savin chain to disk...")
	err = storage.SaveChain(myChain, dbFile)

	if err != nil {
		fmt.Println("Error saving chain: ", err)
	} else {
		fmt.Println("Chain saved successfully!")
	}

}
