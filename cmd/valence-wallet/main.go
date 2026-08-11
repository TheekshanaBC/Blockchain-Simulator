package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"valence/internal/block"
	"valence/internal/cliclient"
)

func printUsage() {
	fmt.Println(cliclient.ColorBlue + cliclient.FormatItalic + "\nAvailable Commands" + cliclient.Reset)
	fmt.Println("  " + cliclient.ColorYellow + "create-wallet" + cliclient.Reset + "                 Create a new wallet in the keystore")
	fmt.Println("  " + cliclient.ColorYellow + "balances" + cliclient.Reset + "                      Show all balances")
	fmt.Println("  " + cliclient.ColorYellow + "mempool" + cliclient.Reset + "                       Show pending transactions")
	fmt.Println("  " + cliclient.ColorYellow + "submit-tx <to> <amount>" + cliclient.Reset + "       Sign and submit a transaction")
	fmt.Println("  " + cliclient.ColorYellow + "faucet <amount>" + cliclient.Reset + "               Request test funds (in VCN)")
	fmt.Println("  " + cliclient.ColorYellow + "exit" + cliclient.Reset + "                          Exit the interactive console")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
}

func main() {
	nodeFlag := flag.String("node", "localhost:3000", "Node HTTP API address")
	walletFlag := flag.String("wallet", "default", "Wallet name to use for signing")
	keystoreFlag := flag.String("keystore", "./data/keystore.json", "Path to keystore file")
	flag.Parse()

	nodeURL := "http://" + *nodeFlag
	if strings.HasPrefix(*nodeFlag, "http") {
		nodeURL = *nodeFlag
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println(cliclient.ColorBlue + "=========================================" + cliclient.Reset)
		fmt.Println(cliclient.FormatBold + "         Valence Wallet Console          " + cliclient.Reset)
		fmt.Println(cliclient.ColorBlue + "=========================================" + cliclient.Reset)
		printUsage()
		runInteractive(nodeURL, *keystoreFlag, *walletFlag)
		return
	}

	runCommand(args, nodeURL, *keystoreFlag, *walletFlag)
}

func runInteractive(nodeURL, keystoreFlag, walletFlag string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n" + cliclient.ColorBlue + "wallet> " + cliclient.Reset)
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		args := strings.Fields(line)
		if args[0] == "exit" || args[0] == "quit" {
			break
		}
		runCommand(args, nodeURL, keystoreFlag, walletFlag)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func runCommand(args []string, nodeURL, keystoreFlag, walletFlag string) {
	command := args[0]
	switch command {
	case "create-wallet":
		err := cliclient.HandleCreateWallet(keystoreFlag, walletFlag)
		if err != nil {
			cliclient.PrintError(err)
		}
	case "balances":
		data, err := cliclient.HandleGet(nodeURL, "/balances")
		if err != nil {
			cliclient.PrintError(err)
		} else {
			cliclient.PrintBalances(data, keystoreFlag)
		}
	case "mempool":
		data, err := cliclient.HandleGet(nodeURL, "/mempool")
		if err != nil {
			cliclient.PrintError(err)
		} else {
			cliclient.PrintMempool(data, keystoreFlag)
		}
	case "faucet":
		if len(args) < 2 {
			fmt.Println("Usage: faucet <amount>")
			return
		}
		amount, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			cliclient.PrintError(fmt.Errorf("invalid amount: %v", err))
			return
		}
		data, err := cliclient.HandleFaucet(nodeURL, keystoreFlag, walletFlag, amount)
		if err != nil {
			cliclient.PrintError(err)
		} else {
			cliclient.PrintGenericJSON(data)
		}
	case "submit-tx":
		if len(args) < 3 {
			fmt.Println("Usage: submit-tx <to_address> <amount_in_vcn>")
			return
		}
		toAddr := args[1]
		amount, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			cliclient.PrintError(fmt.Errorf("invalid amount: %v", err))
			return
		}
		data, err := cliclient.HandleSubmitTx(nodeURL, keystoreFlag, walletFlag, toAddr, amount*block.ElectronsPerVCN)
		if err != nil {
			cliclient.PrintError(err)
		} else {
			cliclient.PrintGenericJSON(data)
		}
	case "help":
		printUsage()
	case "clear":
		fmt.Print("\033[H\033[2J")
	default:
		fmt.Printf("%sUnknown wallet command: %s%s\n", cliclient.ColorRed, command, cliclient.Reset)
	}
}
