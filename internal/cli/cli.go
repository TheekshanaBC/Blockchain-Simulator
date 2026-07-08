package cli

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/ledger"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ANSI Color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// Text Formatting
const (
	FormatBold      = "\033[1m"
	FormatDim       = "\033[2m"
	FormatItalic    = "\033[3m"
	FormatUnderline = "\033[4m"
)

func printHelp() {
	fmt.Println(ColorBlue + FormatItalic + "\nAvailable Commands" + ColorReset)
	fmt.Println(ColorYellow + "  addtx " + FormatDim + "<from> <to> <amount>" + ColorReset + " - Add a new transaction to the pending pool")
	fmt.Println(ColorYellow + "  mine" + ColorReset + "                       - Mine pending transactions into a new block")
	fmt.Println(ColorYellow + "  pool" + ColorReset + "                       - View all pending transactions")
	fmt.Println(ColorYellow + "  balances" + ColorReset + "                   - View account balances")
	fmt.Println(ColorYellow + "  validate" + ColorReset + "                   - Validate the integrity of the blockchain")
	fmt.Println(ColorYellow + "  print" + ColorReset + "                      - Visualize the blockchain structure")
	fmt.Println(ColorYellow + "  help" + ColorReset + "                       - Display available commands")
	fmt.Println(ColorYellow + "  exit" + ColorReset + "                       - Exit the Blockchain CLI")
}

func StartCLI(c *chain.Chain) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(ColorBlue + "=========================================" + ColorReset)
	fmt.Println(FormatBold + "         BlockChain Simulator CLI        " + ColorReset)
	fmt.Println(ColorBlue + "=========================================" + ColorReset)
	printHelp()

	for {
		fmt.Print("\n" + ColorBlue + "Blockchain> " + ColorReset)
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		args := strings.Fields(input)

		if len(args) == 0 {
			continue
		}

		command := args[0]

		switch command {
		case "addtx":
			if len(args) != 4 {
				fmt.Println("Usage: addtx <from> <to> <amount>")
				continue
			}
			amount, err := strconv.ParseFloat(args[3], 64)
			if err != nil {
				fmt.Println("Error: Amount must be a number")
				continue
			}

			tx := block.Transaction{Sender: args[1], Recipient: args[2], Amount: amount}

			err = c.AddTransaction(tx)
			if err != nil {
				fmt.Println("Error: Failed to add Transaction:", err)
			} else {
				fmt.Println("Transaction added to the pending pool!")
			}

		case "mine":
			fmt.Println("Mining new block...")
			err := c.MinePendingTransactions()
			if err != nil {
				fmt.Println("Mine error!", err)
			} else {
				fmt.Println("Block mined successfully!")
			}

		case "balances":
			balances := ledger.CalculateBalances(c.Blocks)
			fmt.Println("--- Account Balances ---")
			for acc, bal := range balances {
				fmt.Printf("%s : %.2f\n", acc, bal)
			}
		case "validate":
			result := c.Validate()
			if result.IsValid {
				fmt.Println("Chain is valid!")
			} else {
				fmt.Printf("Chain is INVALID!. Failed at Block %d: %s\n", result.FailedAtHeight, result.Reason)
			}

		case "print":
			chainJSON, _ := json.MarshalIndent(c.Blocks, "", "  ")
			fmt.Println(string(chainJSON))

		case "exit":
			fmt.Println("Existing...")
			return

		default:
			fmt.Println("Unknown Command. Available: addtx, mine, balances, validate, print, exit")
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}
}
