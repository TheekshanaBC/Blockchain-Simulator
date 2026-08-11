package cliclient

import (
	"encoding/json"
	"fmt"
	"valence/internal/wallet"
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

func getAddressLabel(addr string, wallets map[string]*wallet.Wallet) string {
	if addr == "FAUCET" || addr == "VALENCE_COINBASE" || addr == "Genesis" || addr == "Miner" {
		return addr
	}

	label := addr
	if len(addr) > 8 {
		label = addr[:8] + "..."
	}

	for name, w := range wallets {
		if w.Address() == addr {
			return fmt.Sprintf("%s (%s)", name, label)
		}
	}
	return fmt.Sprintf("Unknown (%s)", label)
}

func PrintError(err error) {
	fmt.Printf("%sError: %v%s\n", ColorRed, err, Reset)
}

func PrintGenericJSON(data interface{}) {
	prettyJSON, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(prettyJSON))
}

func PrintBalances(data interface{}, keystoreFile string) {
	balances, ok := data.(map[string]interface{})
	if !ok {
		PrintGenericJSON(data)
		return
	}

	wallets, _ := wallet.GetAllWallets(keystoreFile)
	
	fmt.Println(ColorCyan + "--- Account Balances ---" + Reset)
	for acc, balInterface := range balances {
		bal, _ := balInterface.(float64)
		label := getAddressLabel(acc, wallets)
		fmt.Printf("%s : %.2f VCN\n", label, bal/1e9)
	}
}

func PrintMempool(data interface{}, keystoreFile string) {
	txs, ok := data.([]interface{})
	if !ok {
		PrintGenericJSON(data)
		return
	}

	if len(txs) == 0 {
		fmt.Println(ColorYellow + "No pending transactions in mempool." + Reset)
		return
	}

	wallets, _ := wallet.GetAllWallets(keystoreFile)
	fmt.Println(ColorCyan + "--- Pending Transactions ---" + Reset)

	for i, txInterface := range txs {
		txMap, ok := txInterface.(map[string]interface{})
		if !ok {
			continue
		}
		sender, _ := txMap["sender"].(string)
		recipient, _ := txMap["recipient"].(string)
		amount, _ := txMap["amount"].(float64)

		senderLabel := getAddressLabel(sender, wallets)
		recipientLabel := getAddressLabel(recipient, wallets)
		fmt.Printf("%s%d.%s %s --> %s : %.2f VCN\n", ColorYellow, i+1, Reset, senderLabel, recipientLabel, amount/1e9)
	}
}
