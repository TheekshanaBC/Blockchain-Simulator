package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"valence/internal/block"
	"valence/internal/cliclient"
)

func main() {
	nodeFlag := flag.String("node", "localhost:3000", "Node HTTP API address")
	walletFlag := flag.String("wallet", "default", "Wallet name to use for signing")
	keystoreFlag := flag.String("keystore", "./data/keystore.json", "Path to keystore file")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: valence-wallet [flags] <command> [args]")
		fmt.Println("\nCommands:")
		fmt.Println("  create-wallet                 Create a new wallet in the keystore")
		fmt.Println("  balances                      Show all balances")
		fmt.Println("  mempool                       Show pending transactions")
		fmt.Println("  submit-tx <to> <amount>       Sign and submit a transaction")
		fmt.Println("  faucet <amount>               Request test funds (in VCN)")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	command := args[0]
	nodeURL := "http://" + *nodeFlag
	if strings.HasPrefix(*nodeFlag, "http") {
		nodeURL = *nodeFlag
	}

	switch command {
	case "create-wallet":
		cliclient.HandleCreateWallet(*keystoreFlag, *walletFlag)
	case "balances":
		cliclient.HandleGet(nodeURL, "/balances")
	case "mempool":
		cliclient.HandleGet(nodeURL, "/mempool")
	case "faucet":
		if len(args) < 2 {
			fmt.Println("Usage: valence-wallet faucet <amount>")
			os.Exit(1)
		}
		amount, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Printf("Invalid amount: %v\n", err)
			os.Exit(1)
		}
		cliclient.HandleFaucet(nodeURL, *keystoreFlag, *walletFlag, amount)
	case "submit-tx":
		if len(args) < 3 {
			fmt.Println("Usage: valence-wallet submit-tx <to_address> <amount_in_vcn>")
			os.Exit(1)
		}
		toAddr := args[1]
		amount, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			fmt.Printf("Invalid amount: %v\n", err)
			os.Exit(1)
		}
		cliclient.HandleSubmitTx(nodeURL, *keystoreFlag, *walletFlag, toAddr, amount*block.ElectronsPerVCN)
	default:
		fmt.Printf("Unknown wallet command: %s\n", command)
		os.Exit(1)
	}
}
