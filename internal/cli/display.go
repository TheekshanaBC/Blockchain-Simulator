package cli

import (
	"blockchain-simulator/internal/chain"
	"blockchain-simulator/internal/wallet"
	"fmt"
	"strings"
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
	for _, cmd := range commands {
		fmt.Printf("%s  %-20s%s - %s\n", ColorYellow, cmd.usage, Reset, cmd.help)
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
		printLine(fmt.Sprintf("Difficulty: %d", b.Header.Difficulty), Reset, innerW)
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
