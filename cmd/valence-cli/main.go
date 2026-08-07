package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"valence/internal/block"
	"valence/internal/wallet"
)

func printJSONResponse(resp *http.Response) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		prettyJSON, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Println(string(prettyJSON))
	} else {
		// Might be an array
		var parsedArray []interface{}
		if err := json.Unmarshal(body, &parsedArray); err == nil {
			prettyJSON, _ := json.MarshalIndent(parsedArray, "", "  ")
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Println(string(body))
		}
	}
}

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
		resp, err := http.Get(nodeURL + "/status")
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

	case "balances":
		resp, err := http.Get(nodeURL + "/balances")
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

	case "mempool":
		resp, err := http.Get(nodeURL + "/mempool")
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

	case "peers":
		resp, err := http.Get(nodeURL + "/peers")
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

	case "mine":
		resp, err := http.Post(nodeURL+"/mine", "application/json", nil)
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

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

		ks := wallet.NewKeystore(*keystoreFlag)
		w, err := ks.LoadWallet(*walletFlag)
		if err != nil {
			fmt.Printf("Failed to load wallet '%s': %v\n", *walletFlag, err)
			fmt.Println("Creating a new wallet for faucet...")
			w = wallet.NewWallet()
			err = ks.SaveWallet(*walletFlag, w)
			if err != nil {
				fmt.Printf("Failed to save new wallet: %v\n", err)
				os.Exit(1)
			}
		}

		payload := map[string]interface{}{
			"address": w.Address(),
			"amount":  amount,
		}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(nodeURL+"/faucet", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

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

		// Ensure directory exists for keystore
		dir := filepath.Dir(*keystoreFlag)
		os.MkdirAll(dir, 0750)

		ks := wallet.NewKeystore(*keystoreFlag)
		w, err := ks.LoadWallet(*walletFlag)
		if err != nil {
			fmt.Printf("Failed to load wallet '%s': %v\n", *walletFlag, err)
			fmt.Println("To create a wallet, you can run the faucet command or generate one programmatically.")
			os.Exit(1)
		}

		tx := block.Transaction{
			Sender:    w.Address(),
			Recipient: toAddr,
			Amount:    amount,
			Sequence:  uint64(time.Now().UnixNano()), // Hack for sprint 2 (should fetch real sequence from node)
			Timestamp: time.Now().UnixNano(),
		}
		tx.ComputeID()
		tx.Sign(w.PrivateKey)

		body, _ := json.Marshal(tx)
		resp, err := http.Post(nodeURL+"/tx/submit", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("Error connecting to node: %v\n", err)
			os.Exit(1)
		}
		printJSONResponse(resp)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
