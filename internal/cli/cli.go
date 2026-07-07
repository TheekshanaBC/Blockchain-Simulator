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

func StartCLI(c *chain.Chain) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("==== BlockChain Simulator CLI ====")
	fmt.Println("Commands: addtx <from> <to> <amount> | mine | balances | validate | print | exit")

	for {
		fmt.Print("\n>")
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

}
