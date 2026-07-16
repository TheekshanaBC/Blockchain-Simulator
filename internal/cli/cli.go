package cli

import (
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/wallet"
	"bufio"
	"fmt"
	"os"
	"strings"
)

type cliContext struct {
	chain            *chain.Chain
	activeWallet     *wallet.Wallet
	activeWalletName string
	walletFile       string
}

type command struct {
	name    string
	usage   string
	help    string
	handler func(ctx *cliContext, args []string)
}

var commands []command

func init() {
	commands = []command{
		{"createwallet", "createwallet <name>", "Create a new wallet and save it to disk", handleCreateWallet},
		{"loadwallet", "loadwallet <name>", "Load an existing wallet from disk", handleLoadWallet},
		{"wallets", "wallets", "List all saved wallets", handleWallets},
		{"mywallet", "mywallet", "View your current wallet address and balance", handleMyWallet},
		{"faucet", "faucet <amount>", "Request free funds from the Faucet", handleFaucet},
		{"addtx", "addtx <to> <amount>", "Send funds to an address (uses current wallet)", handleAddTx},
		{"mine", "mine", "Mine pending transactions into a new block", handleMine},
		{"pool", "pool", "View all pending transactions", handlePool},
		{"balances", "balances", "View all account balances", handleBalances},
		{"validate", "validate", "Validate the integrity of the blockchain", handleValidate},
		{"print", "print", "Visualize the blockchain structure", handlePrint},
		{"help", "help", "Display available commands", handleHelp},
		{"clear", "clear", "Clear the terminal screen", handleClear},
		{"exit", "exit", "Exit the Blockchain CLI", handleExit},
	}
}


func StartCLI(c *chain.Chain) {
	scanner := bufio.NewScanner(os.Stdin)

	ctx := &cliContext{
		chain:            c,
		activeWallet:     nil,
		activeWalletName: "",
		walletFile:       "data/wallet.json",
	}

	// Ensure data directory exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}

	fmt.Println(ColorYellow + "No active wallet loaded. Use 'loadwallet <name>' or 'createwallet <name>'." + Reset)

	fmt.Println(ColorBlue + "=========================================" + Reset)
	fmt.Println(FormatBold + "         BlockChain Simulator CLI        " + Reset)
	fmt.Println(ColorBlue + "=========================================" + Reset)
	printHelp()

	cmdMap := make(map[string]command)
	var cmdNames []string
	for _, cmd := range commands {
		cmdMap[cmd.name] = cmd
		cmdNames = append(cmdNames, cmd.name)
	}

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

		commandName := args[0]
		cmd, exists := cmdMap[commandName]

		if exists {
			cmd.handler(ctx, args)
			if commandName == "exit" {
				return
			}
		} else {
			fmt.Println(ColorRed + "Unknown Command\n" + Reset + "Available Commands: " + strings.Join(cmdNames, ", "))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}
}
