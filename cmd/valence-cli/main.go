package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
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
	nodeURL := flag.String("node", "http://localhost:3001", "Node URL")
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

	// Fix L4: Rearrange os.Args so that flags can appear after subcommands
	var newArgs []string
	var positional []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			newArgs = append(newArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				newArgs = append(newArgs, os.Args[i+1])
				i++
			}
		} else {
			positional = append(positional, arg)
		}
	}
	os.Args = append([]string{os.Args[0]}, append(newArgs, positional...)...)

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	// Since we are extracting arguments manually, we should use flag.Args() for subcommands
	args := flag.Args()[1:] // everything after the command


	switch command {
	case "createwallet":
		err := cliclient.HandleCreateWallet(*keystorePath, *walletName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "getbalance":
		addr := *walletName 
		if len(args) > 0 {
			addr = args[0]
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
		if len(args) < 2 {
			fmt.Println("Usage: valence-cli sendtoaddress <address> <amount>")
			os.Exit(1)
		}
		toAddr := args[0]
		amountStr := args[1]
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid amount: %v\n", err)
			os.Exit(1)
		}
		amountInt := int64(math.Round(amountFloat * float64(block.ElectronsPerVCN))) // convert VCN to Electrons
		res, err := cliclient.HandleSubmitTx(*nodeURL, *keystorePath, *walletName, toAddr, amountInt)
		printJSON(res, err)

	case "getmempoolinfo":
		res, err := cliclient.HandleGet(*nodeURL, "/mempool")
		printJSON(res, err)

	case "faucet":
		if len(args) < 1 {
			fmt.Println("Usage: valence-cli faucet <amount>")
			os.Exit(1)
		}
		amountStr := args[0]
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid amount: %v\n", err)
			os.Exit(1)
		}
		amountInt := int64(math.Round(amountFloat * float64(block.ElectronsPerVCN))) // convert VCN to Electrons
		res, err := cliclient.HandleFaucet(*nodeURL, *keystorePath, *walletName, amountInt)
		printJSON(res, err)

	case "getnetworkinfo", "getblockchaininfo":
		res, err := cliclient.HandleGet(*nodeURL, "/status")
		printJSON(res, err)

	case "getpeerinfo":
		res, err := cliclient.HandleGet(*nodeURL, "/peers")
		printJSON(res, err)

	case "addnode":
		if len(args) < 1 {
			fmt.Println("Usage: valence-cli addnode <address>")
			os.Exit(1)
		}
		res, err := cliclient.HandlePeersAdd(*nodeURL, args[0])
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
