package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"valence/internal/block"
	"valence/internal/cliclient"
	"valence/internal/wallet"
)

func printJSON(data interface{}, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}

func main() {
	// Global flags
	nodeURL := flag.String("node", "http://localhost:8080", "Node URL")
	keystorePath := flag.String("keystore", "./data/wallets/keys.json", "Path to keystore file")
	walletName := flag.String("wallet", "primary", "Wallet name/index to use")
	
	flag.Usage = func() {
		fmt.Println("Usage: valence-cli [global flags] <command> [args]")
		fmt.Println("\nGlobal Flags:")
		flag.PrintDefaults()
		fmt.Println("\nCommands:")
		fmt.Println("  createwallet                   Create a new wallet in the keystore")
		fmt.Println("  getbalance [address]           Get balance of an address (defaults to active wallet)")
		fmt.Println("  sendtoaddress <address> <amt>  Send VCN to an address (amount is in VCN)")
		fmt.Println("  getmempoolinfo                 Get current mempool contents")
		fmt.Println("  faucet <amt>                   Request faucet funds (amount is in VCN)")
		fmt.Println("  getnetworkinfo                 Get node status (blockchain info)")
		fmt.Println("  getpeerinfo                    Get connected peers")
		fmt.Println("  addnode <address>              Manually connect to a peer")
		fmt.Println("  generate                       Manually mine a block")
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	switch command {
	case "createwallet":
		err := cliclient.HandleCreateWallet(*keystorePath, *walletName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "getbalance":
		addr := *walletName 
		if flag.NArg() > 1 {
			addr = flag.Arg(1)
		} else {
			// If no address provided, try to load from keystore
			w, err := wallet.LoadFromKeystore(*keystorePath, *walletName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading wallet: %v. Please provide an address or use createwallet.\n", err)
				os.Exit(1)
			}
			addr = w.Address()
		}
		res, err := cliclient.HandleGet(*nodeURL, "/balances/"+addr)
		printJSON(res, err)

	case "sendtoaddress":
		if flag.NArg() < 3 {
			fmt.Println("Usage: valence-cli sendtoaddress <address> <amount>")
			os.Exit(1)
		}
		toAddr := flag.Arg(1)
		amountStr := flag.Arg(2)
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid amount: %v\n", err)
			os.Exit(1)
		}
		amountInt := int64(amountFloat * float64(block.ElectronsPerVCN)) // convert VCN to Electrons
		res, err := cliclient.HandleSubmitTx(*nodeURL, *keystorePath, *walletName, toAddr, amountInt)
		printJSON(res, err)

	case "getmempoolinfo":
		res, err := cliclient.HandleGet(*nodeURL, "/mempool")
		printJSON(res, err)

	case "faucet":
		if flag.NArg() < 2 {
			fmt.Println("Usage: valence-cli faucet <amount>")
			os.Exit(1)
		}
		amountStr := flag.Arg(1)
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid amount: %v\n", err)
			os.Exit(1)
		}
		amountInt := int64(amountFloat) // Do NOT multiply by ElectronsPerVCN, API expects raw VCN
		res, err := cliclient.HandleFaucet(*nodeURL, *keystorePath, *walletName, amountInt)
		printJSON(res, err)

	case "getnetworkinfo", "getblockchaininfo":
		res, err := cliclient.HandleGet(*nodeURL, "/status")
		printJSON(res, err)

	case "getpeerinfo":
		res, err := cliclient.HandleGet(*nodeURL, "/peers")
		printJSON(res, err)

	case "addnode":
		if flag.NArg() < 2 {
			fmt.Println("Usage: valence-cli addnode <address>")
			os.Exit(1)
		}
		res, err := cliclient.HandlePeersAdd(*nodeURL, flag.Arg(1))
		printJSON(res, err)

	case "generate":
		res, err := cliclient.HandlePost(*nodeURL, "/mine")
		printJSON(res, err)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}
