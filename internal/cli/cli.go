package cli

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/ledger"
	"blockchain-simulator/internal/wallet"
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
	fmt.Println(ColorYellow + "  createwallet " + FormatDim + "<name>" + Reset + "   - Create a new wallet and save it to disk")
	fmt.Println(ColorYellow + "  loadwallet " + FormatDim + "<name>" + Reset + "     - Load an existing wallet from disk")
	fmt.Println(ColorYellow + "  wallets" + Reset + "               - List all saved wallets")
	fmt.Println(ColorYellow + "  mywallet" + Reset + "              - View your current wallet address and balance")
	fmt.Println(ColorYellow + "  faucet " + FormatDim + "<amount>" + Reset + "       - Request free funds from the Faucet")
	fmt.Println(ColorYellow + "  addtx " + FormatDim + "<to> <amount>" + Reset + "   - Send funds to an address (uses current wallet)")
	fmt.Println(ColorYellow + "  mine" + Reset + "                  - Mine pending transactions into a new block")
	fmt.Println(ColorYellow + "  pool" + Reset + "                  - View all pending transactions")
	fmt.Println(ColorYellow + "  balances" + Reset + "              - View all account balances")
	fmt.Println(ColorYellow + "  validate" + Reset + "              - Validate the integrity of the blockchain")
	fmt.Println(ColorYellow + "  print" + Reset + "                 - Visualize the blockchain structure")
	fmt.Println(ColorYellow + "  help" + Reset + "                  - Display available commands")
	fmt.Println(ColorYellow + "  clear" + Reset + "                 - Clear the terminal screen")
	fmt.Println(ColorYellow + "  exit" + Reset + "                  - Exit the Blockchain CLI")
}

func StartCLI(c *chain.Chain) {
	scanner := bufio.NewScanner(os.Stdin)

	var activeWallet *wallet.Wallet
	activeWalletName := ""
	walletFile := "data/wallet.json"

	// Ensure data directory exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}

	fmt.Println(ColorYellow + "No active wallet loaded. Use 'loadwallet <name>' or 'createwallet <name>'." + Reset)

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

		case "createwallet":
			if len(args) != 2 {
				fmt.Println(ColorRed + "Error: " + Reset + "Please specify a name: " + ColorGreen + FormatDim + "createwallet <name>" + Reset)
				continue
			}
			name := args[1]
			w := wallet.NewWallet()
			err := wallet.SaveToKeystore(walletFile, name, w)
			if err != nil {
				fmt.Println(ColorRed+"Error saving wallet:"+Reset, err)
			} else {
				activeWallet = w
				activeWalletName = name
				address := wallet.AddressFromPublicKey(w.PublicKeyBytes)
				fmt.Printf("%sWallet '%s' created and saved successfully!%s\n", ColorGreen, name, Reset)
				fmt.Println(ColorCyan + "Your Address: " + Reset + address)
			}

		case "loadwallet":
			if len(args) != 2 {
				fmt.Println(ColorRed + "Error: " + Reset + "Please specify a name: " + ColorGreen + FormatDim + "loadwallet <name>" + Reset)
				continue
			}
			name := args[1]
			w, err := wallet.LoadFromKeystore(walletFile, name)
			if err != nil {
				fmt.Println(ColorRed+"Error loading wallet:"+Reset, err)
			} else {
				activeWallet = w
				activeWalletName = name
				address := wallet.AddressFromPublicKey(w.PublicKeyBytes)
				fmt.Printf("%sWallet '%s' loaded successfully!%s\n", ColorGreen, name, Reset)
				fmt.Println(ColorCyan + "Your Address: " + Reset + address)
			}

		case "wallets":
			wallets, err := wallet.GetAllWallets(walletFile)
			if err != nil || len(wallets) == 0 {
				fmt.Println(ColorYellow + "No wallets found." + Reset)
				continue
			}
			fmt.Println(ColorCyan + "--- Saved Wallets ---" + Reset)
			for name, w := range wallets {
				addr := wallet.AddressFromPublicKey(w.PublicKeyBytes)
				fmt.Printf("%s- %s%s : %s\n", ColorYellow, name, Reset, addr)
			}

		case "mywallet":
			if activeWallet == nil {
				fmt.Println(ColorRed + "Error: " + Reset + "No active wallet. Use 'loadwallet <name>' or 'createwallet <name>'.")
				continue
			}
			address := wallet.AddressFromPublicKey(activeWallet.PublicKeyBytes)
			balances := ledger.CalculateBalances(c.Blocks)
			fmt.Printf("%sActive Wallet: %s%s\n", ColorCyan, Reset, activeWalletName)
			fmt.Println(ColorCyan + "Your Address: " + Reset + address)
			fmt.Printf(ColorCyan+"Your Balance: "+Reset+"%d\n", balances[address])

		case "faucet":
			if activeWallet == nil {
				fmt.Println(ColorRed + "Error: " + Reset + "No active wallet. Use 'loadwallet <name>' or 'createwallet <name>'.")
				continue
			}
			if len(args) != 2 {
				fmt.Println(ColorRed + "Error: " + Reset + "Invalid arguments!" + Reset + "\nTry again using correct format:" + ColorGreen + FormatDim + " faucet <amount>" + Reset)
				continue
			}
			amount, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				fmt.Println(ColorRed + "Error: " + Reset + "Amount must be a number" + Reset)
				continue
			}

			address := wallet.AddressFromPublicKey(activeWallet.PublicKeyBytes)
			tx := block.Transaction{
				Sender:    "FAUCET",
				Recipient: address,
				Amount:    amount,
			}

			err = c.AddTransaction(tx)
			if err != nil {
				fmt.Println(ColorRed+"Error: "+Reset+"Failed to get FAUCET funds:\n", err)
			} else {
				fmt.Println(ColorGreen + "Requested funds from FAUCET to pending pool successfully!" + Reset)
			}

		case "addtx":
			if activeWallet == nil {
				fmt.Println(ColorRed + "Error: " + Reset + "No active wallet. Use 'loadwallet <name>' or 'createwallet <name>' first to send funds.")
				continue
			}
			if len(args) != 3 {
				fmt.Println(ColorRed + "Error: " + Reset + "Invalid arguments!" + Reset + "\nTry again using correct format:" + ColorGreen + FormatDim + " addtx <to> <amount>" + Reset)
				continue
			}
			amount, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				fmt.Println(ColorRed + "Error: " + Reset + "Amount must be a number" + Reset)
				continue
			}

			senderAddress := wallet.AddressFromPublicKey(activeWallet.PublicKeyBytes)
			tx := block.Transaction{
				Sender:    senderAddress,
				Recipient: args[1],
				Amount:    amount,
				PublicKey: activeWallet.PublicKeyBytes,
			}

			err = tx.Sign(activeWallet.PrivateKey)
			if err != nil {
				fmt.Println(ColorRed+"Error signing transaction:"+Reset, err)
				continue
			}

			err = c.AddTransaction(tx)
			if err != nil {
				fmt.Println(ColorRed+"Error: "+Reset+"Failed to add Transaction:\n", err)
			} else {
				fmt.Println(ColorGreen + "Transaction signed and added to the pending pool!" + Reset)
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
				wallets, _ := wallet.GetAllWallets(walletFile)
				fmt.Println(ColorCyan + "--- Pending Transactions ---" + Reset)
				for i, tx := range c.PendingPool {
					senderLabel := getAddressLabel(tx.Sender, wallets)
					recipientLabel := getAddressLabel(tx.Recipient, wallets)
					fmt.Printf("%s%d.%s %s --> %s : %d\n", ColorYellow, i+1, Reset, senderLabel, recipientLabel, tx.Amount)
				}
			}

		case "balances":
			balances := ledger.CalculateBalances(c.Blocks)
			wallets, _ := wallet.GetAllWallets(walletFile)
			fmt.Println(ColorCyan + "--- Account Balances ---" + Reset)
			for acc, bal := range balances {
				label := getAddressLabel(acc, wallets)
				fmt.Printf("%s : %d\n", label, bal)
			}

		case "validate":
			result := c.Validate()
			if result.IsValid {
				fmt.Println(ColorGreen + "Chain is valid!" + Reset)
			} else {
				fmt.Printf(ColorRed+"Error: "+Reset+"Chain is INVALID. Failed at Block %d: %s\n", result.FailedAtHeight, result.Reason)
			}

		case "print":
			wallets, _ := wallet.GetAllWallets(walletFile)
			printBlockchain(c, wallets)

		case "help":
			printHelp()

		case "clear":
			fmt.Print("\033[H\033[2J")

		case "exit":
			fmt.Println(ColorYellow + "Exiting..." + Reset)
			return

		default:
			fmt.Println(ColorRed + "Unknown Command\n" + Reset + "Available Commands: createwallet, loadwallet, wallets, mywallet, faucet, addtx, mine, pool, balances, validate, print, help, clear, exit")
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}
}

func getAddressLabel(addr string, wallets map[string]*wallet.Wallet) string {
	if addr == "FAUCET" || addr == "COINBASE" || addr == "Genesis" || addr == "Miner" {
		return addr
	}

	// Default label is just the address truncated
	label := addr
	if len(addr) > 8 {
		label = addr[:8] + "..."
	}

	for name, w := range wallets {
		if wallet.AddressFromPublicKey(w.PublicKeyBytes) == addr {
			return fmt.Sprintf("%s (%s)", name, label)
		}
	}
	return fmt.Sprintf("Unknown (%s)", label)
}

func printLine(text string, color string, innerW int) {
	if len(text) > innerW {
		text = text[:innerW-3] + "..."
	}

	padding := innerW - len(text)
	fmt.Printf("%s| %s%s%s%s %s|\n", ColorBlue, color, text, Reset, strings.Repeat(" ", padding), ColorBlue)
}

func printBlockchain(c *chain.Chain, wallets map[string]*wallet.Wallet) {
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
				senderLabel := getAddressLabel(tx.Sender, wallets)
				recipientLabel := getAddressLabel(tx.Recipient, wallets)
				txStr := fmt.Sprintf("  %d. %s -> %s : %d", j+1, senderLabel, recipientLabel, tx.Amount)
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
