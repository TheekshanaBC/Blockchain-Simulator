package main

import (
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/cli"
	"blockchain-simulator/internal/storage"
	"flag"
	"fmt"
	"os"
)

const dbFile = "data/chain.json"

func main() {
	var difficulty int
	var retargetWindow int
	var targetBlockTime int64
	var minDiff int
	var maxDiff int
	flag.IntVar(&difficulty, "diff", 4, "Mining difficulty (number of leading zeros)")
	flag.IntVar(&retargetWindow, "retarget-window", 2, "Number of blocks between difficulty retargets")
	flag.Int64Var(&targetBlockTime, "target-block-time", 8, "Target time per block in seconds")
	flag.IntVar(&minDiff, "min-diff", 3, "Minimum difficulty")
	flag.IntVar(&maxDiff, "max-diff", 8, "Maximum difficulty")
	flag.Parse()

	var myChain *chain.Chain

	loadedChain, err := storage.LoadChain(dbFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No existing chain found. Creating a new one...")
			myChain = chain.NewChain(difficulty, retargetWindow, targetBlockTime, minDiff, maxDiff)
		} else {
			fmt.Printf("Error loading existing chain from %s: %v\n", dbFile, err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Loaded existing blockchain from disk.")
		fmt.Println("Validating...")
		valRes := loadedChain.Validate()
		if !valRes.IsValid {
			fmt.Printf("Loaded chain is invalid! Failed at block %d: %s\n", valRes.FailedAtHeight, valRes.Reason)
			os.Exit(1)
		}
		fmt.Println("Chain validated successfully.")
		myChain = loadedChain
	}

	cli.StartCLI(myChain)

	fmt.Println("Saving chain to disk...")
	err = storage.SaveChain(myChain, dbFile)

	if err != nil {
		fmt.Println("Error saving chain: ", err)
	} else {
		fmt.Println("Chain saved successfully!")
	}

}
