package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"valence/internal/cliclient"
)

func main() {
	nodeFlag := flag.String("node", "localhost:3000", "Node HTTP API address")
	walletFlag := flag.String("wallet", "default", "Wallet name to use for signing")
	keystoreFlag := flag.String("keystore", "./data/keystore.json", "Path to keystore file")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: valence-cli [flags] <command> [args]")
		fmt.Println("\nCommands:")
		fmt.Println("  status                        Show node status")
		fmt.Println("  balances                      Show all balances")
		fmt.Println("  mempool                       Show pending transactions")
		fmt.Println("  peers                         Show connected peers")
		fmt.Println("  submit-tx <to> <amount>       Sign and submit a transaction")
		fmt.Println("  mine                          Trigger mining")
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
	case "status":
		cliclient.HandleGet(nodeURL, "/status")
	case "balances":
		cliclient.HandleGet(nodeURL, "/balances")
	case "mempool":
		cliclient.HandleGet(nodeURL, "/mempool")
	case "peers":
		cliclient.HandleGet(nodeURL, "/peers")
	case "mine":
		cliclient.HandlePost(nodeURL, "/mine")
	case "faucet":
		if len(args) < 2 {
			fmt.Println("Usage: valence-cli faucet <amount>")
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
			fmt.Println("Usage: valence-cli submit-tx <to_address> <amount_in_electrons>")
			os.Exit(1)
		}
		toAddr := args[1]
		amount, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			fmt.Printf("Invalid amount: %v\n", err)
			os.Exit(1)
		}
		cliclient.HandleSubmitTx(nodeURL, *keystoreFlag, *walletFlag, toAddr, amount)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
