package cli

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/ledger"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Reset           = "\033[0m"
	ColorRed        = "\033[31m"
	ColorGreen      = "\033[32m"
	ColorYellow     = "\033[33m"
	ColorBlue       = "\033[34m"
	ColorCyan       = "\033[36m"
	FormatBold      = "\033[1m"
	FormatDim       = "\033[2m"
	FormatItalic    = "\033[3m"
	FormatUnderline = "\033[4m"
)

func printHelp() {
	fmt.Println(ColorBlue + FormatItalic + "\nAvailable Commands" + Reset)
	fmt.Println(ColorYellow + "  addtx " + FormatDim + "<from> <to> <amount>" + Reset + " - Add a new transaction to the pending pool")
	fmt.Println(ColorYellow + "  mine" + Reset + "                       - Mine pending transactions into a new block")
	fmt.Println(ColorYellow + "  pool" + Reset + "                       - View all pending transactions")
	fmt.Println(ColorYellow + "  balances" + Reset + "                   - View account balances")
	fmt.Println(ColorYellow + "  validate" + Reset + "                   - Validate the integrity of the blockchain")
	fmt.Println(ColorYellow + "  print" + Reset + "                      - Visualize the blockchain structure")
	fmt.Println(ColorYellow + "  help" + Reset + "                       - Display available commands")
	fmt.Println(ColorYellow + "  clear" + Reset + "                      - Clear the terminal screen")
	fmt.Println(ColorYellow + "  exit" + Reset + "                       - Exit the Blockchain CLI")
}

func StartCLI(c *chain.Chain) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(ColorBlue + "=========================================" + Reset)
	fmt.Println(FormatBold + "         BlockChain Simulator CLI        " + Reset)
	fmt.Println(ColorBlue + "=========================================" + Reset)
	printHelp()

	for {
		fmt.Print("\n" + ColorBlue + "Blockchain> " + Reset)
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
				fmt.Println(ColorRed + "Error: " + Reset + "Invalid arguments!" + Reset + "\nTry again using correct format:" + ColorGreen + FormatDim + " addtx <from> <to> <amount>" + Reset)
				continue
			}
			amount, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				fmt.Println(ColorRed + "Error: " + Reset + "Amount must be a number" + Reset)
				continue
			}

			tx := block.Transaction{Sender: args[1], Recipient: args[2], Amount: amount}

			err = c.AddTransaction(tx)
			if err != nil {
				fmt.Println(ColorRed+"Error: "+Reset+"Failed to add Transaction:", err)
			} else {
				fmt.Println(ColorGreen + "Transaction added to the pending pool!" + Reset)
			}

		case "mine":
			fmt.Println(ColorYellow + FormatDim + "Mining new block..." + Reset)
			startTime := time.Now()
			err := c.MinePendingTransactions()
			miningTime := time.Since(startTime)
			if err != nil {
				fmt.Println(ColorRed+"Error: "+Reset+"Failed to mine block:", err)
			} else {
				fmt.Printf(ColorGreen+"Block mined successfully! Time: %s\n"+Reset, miningTime.Round(time.Millisecond))
			}

		case "pool":
			if len(c.PendingPool) == 0 {
				fmt.Println(ColorYellow + "No pending transactions!" + Reset)
			} else {
				fmt.Println(ColorCyan + "--- Pending Transactions ---" + Reset)
				for i, tx := range c.PendingPool {
					fmt.Printf("%s%d.%s %s --> %s : %d\n", ColorYellow, i+1, Reset, tx.Sender, tx.Recipient, tx.Amount)
				}
			}

		case "balances":
			balances := ledger.CalculateBalances(c.Blocks)
			fmt.Println(ColorCyan + "--- Account Balances ---" + Reset)
			for acc, bal := range balances {
				fmt.Printf("%s : %d\n", acc, bal)
			}

		case "validate":
			result := c.Validate()
			if result.IsValid {
				fmt.Println(ColorGreen + "Chain is valid!" + Reset)
			} else {
				fmt.Printf(ColorRed+"Error: "+Reset+"Chain is INVALID. Failed at Block %d: %s\n", result.FailedAtHeight, result.Reason)
			}

		case "print":
			printBlockchain(c)

		case "help":
			printHelp()

		case "clear":
			fmt.Print("\033[H\033[2J")

		case "exit":
			fmt.Println(ColorYellow + "Exiting..." + Reset)
			return

		default:
			fmt.Println(ColorRed + "Unknown Command\n" + Reset + "Available Commands: addtx, mine, pool, balances, validate, print, help, clear, exit")
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}
}

func printLine(text string, color string, innerW int) {
	if len(text) > innerW {
		text = text[:innerW-3] + "..."
	}

	padding := innerW - len(text)
	fmt.Printf("%s| %s%s%s%s %s|\n", ColorBlue, color, text, Reset, strings.Repeat(" ", padding), ColorBlue)
}

func printBlockchain(c *chain.Chain) {
	fmt.Println(ColorCyan + "--- Blockchain Visualizer ---" + Reset)
	boxWidth := 85
	innerW := boxWidth - 4

	for i, b := range c.Blocks {
		fmt.Println(ColorBlue + "+" + strings.Repeat("-", boxWidth-2) + "+" + Reset)
		printLine(fmt.Sprintf("Block %d", b.Height), ColorCyan, innerW)
		printLine(fmt.Sprintf("Hash: %s", b.Hash), ColorYellow, innerW)
		printLine(fmt.Sprintf("Prev: %s", b.Header.PrevHash), Reset, innerW)
		printLine(fmt.Sprintf("Merkle Root: %s", b.Header.MerkleRoot), Reset, innerW)
		printLine(fmt.Sprintf("Nonce: %d", b.Header.Nonce), Reset, innerW)
		printLine(fmt.Sprintf("Tx Count: %d", len(b.Transactions)), ColorGreen, innerW)
		if len(b.Transactions) > 0 {
			printLine("Transactions:", ColorCyan, innerW)
			for j, tx := range b.Transactions {
				txStr := fmt.Sprintf("  %d. %s -> %s : %d", j+1, tx.Sender, tx.Recipient, tx.Amount)
				printLine(txStr, Reset, innerW)
			}
		}
		fmt.Println(ColorBlue + "+" + strings.Repeat("-", boxWidth-2) + "+" + Reset)
		if i < len(c.Blocks)-1 {
			spaces := strings.Repeat(" ", boxWidth/2)
			fmt.Printf("%s%s|\n", ColorBlue, spaces)
			fmt.Printf("%s%sv%s\n", ColorBlue, spaces, Reset)
		}
		fmt.Println()
	}
}
