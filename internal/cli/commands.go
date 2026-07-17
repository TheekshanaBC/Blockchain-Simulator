package cli

import (
	"blockchain-simulator/internal/block"
	"blockchain-simulator/internal/ledger"
	"blockchain-simulator/internal/wallet"
	"fmt"
	"os"
	"strconv"
	"time"
)

func handleCreateWallet(ctx *cliContext, args []string) {
	if len(args) != 2 {
		fmt.Println(ColorRed + "Error: " + Reset + "Please specify a name: " + ColorGreen + FormatDim + "createwallet <name>" + Reset)
		return
	}
	name := args[1]
	w := wallet.NewWallet()
	err := wallet.SaveToKeystore(ctx.walletFile, name, w)
	if err != nil {
		fmt.Println(ColorRed+"Error saving wallet:"+Reset, err)
	} else {
		ctx.activeWallet = w
		ctx.activeWalletName = name
		address := wallet.AddressFromPublicKey(w.PublicKeyBytes)
		fmt.Printf("%sWallet '%s' created and saved successfully!%s\n", ColorGreen, name, Reset)
		fmt.Println(ColorCyan + "Your Address: " + Reset + address)
	}
}

func handleLoadWallet(ctx *cliContext, args []string) {
	if len(args) != 2 {
		fmt.Println(ColorRed + "Error: " + Reset + "Please specify a name: " + ColorGreen + FormatDim + "loadwallet <name>" + Reset)
		return
	}
	name := args[1]
	w, err := wallet.LoadFromKeystore(ctx.walletFile, name)
	if err != nil {
		fmt.Println(ColorRed+"Error loading wallet:"+Reset, err)
	} else {
		ctx.activeWallet = w
		ctx.activeWalletName = name
		address := wallet.AddressFromPublicKey(w.PublicKeyBytes)
		fmt.Printf("%sWallet '%s' loaded successfully!%s\n", ColorGreen, name, Reset)
		fmt.Println(ColorCyan + "Your Address: " + Reset + address)
	}
}

func handleWallets(ctx *cliContext, args []string) {
	wallets, err := wallet.GetAllWallets(ctx.walletFile)
	if err != nil || len(wallets) == 0 {
		fmt.Println(ColorYellow + "No wallets found." + Reset)
		return
	}
	fmt.Println(ColorCyan + "--- Saved Wallets ---" + Reset)
	for name, w := range wallets {
		addr := wallet.AddressFromPublicKey(w.PublicKeyBytes)
		activeMark := ""
		if name == ctx.activeWalletName {
			activeMark = ColorGreen + " (Active)" + Reset
		}
		fmt.Printf("- %s: %s%s\n", name, addr, activeMark)
	}
}

func handleMyWallet(ctx *cliContext, args []string) {
	if ctx.activeWallet == nil {
		fmt.Println(ColorRed + "Error: " + Reset + "No active wallet. Use 'loadwallet <name>' or 'createwallet <name>'.")
		return
	}
	address := wallet.AddressFromPublicKey(ctx.activeWallet.PublicKeyBytes)
	balances := ledger.CalculateBalances(ctx.chain.Blocks)
	balance := balances[address]
	fmt.Printf("%sActive Wallet:%s %s\n", ColorCyan, Reset, ctx.activeWalletName)
	fmt.Printf("%sYour Address:%s %s\n", ColorCyan, Reset, address)
	fmt.Printf("%sYour Balance:%s %d\n", ColorCyan, Reset, balance)
}

func handleFaucet(ctx *cliContext, args []string) {
	if ctx.activeWallet == nil {
		fmt.Println(ColorRed + "Error: " + Reset + "No active wallet. Use 'loadwallet <name>' or 'createwallet <name>' first to request funds.")
		return
	}
	if len(args) != 2 {
		fmt.Println(ColorRed + "Error: " + Reset + "Invalid arguments!" + Reset + "\nTry again using correct format:" + ColorGreen + FormatDim + " faucet <amount>" + Reset)
		return
	}
	amount, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		fmt.Println(ColorRed + "Error: " + Reset + "Amount must be a number" + Reset)
		return
	}

	address := wallet.AddressFromPublicKey(ctx.activeWallet.PublicKeyBytes)
	err = ctx.chain.RequestFaucetFunds(address, amount)
	if err != nil {
		fmt.Println(ColorRed+"Error: "+Reset+"Failed to get FAUCET funds:\n", err)
	} else {
		fmt.Println(ColorGreen + "Requested funds from FAUCET to pending pool successfully!" + Reset)
	}
}

func handleAddTx(ctx *cliContext, args []string) {
	if ctx.activeWallet == nil {
		fmt.Println(ColorRed + "Error: " + Reset + "No active wallet. Use 'loadwallet <name>' or 'createwallet <name>' first to send funds.")
		return
	}
	if len(args) != 3 {
		fmt.Println(ColorRed + "Error: " + Reset + "Invalid arguments!" + Reset + "\nTry again using correct format:" + ColorGreen + FormatDim + " addtx <to> <amount>" + Reset)
		return
	}
	amount, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		fmt.Println(ColorRed + "Error: " + Reset + "Amount must be a number" + Reset)
		return
	}

	senderAddress := wallet.AddressFromPublicKey(ctx.activeWallet.PublicKeyBytes)

	sequences := ledger.CalculatePendingSequences(ctx.chain.Blocks, ctx.chain.PendingPool)
	nextSeq := sequences[senderAddress] + 1

	tx := block.Transaction{
		Sender:    senderAddress,
		Recipient: args[1],
		Amount:    amount,
		Sequence:  nextSeq,
		PublicKey: ctx.activeWallet.PublicKeyBytes,
	}

	err = tx.Sign(ctx.activeWallet.PrivateKey)
	if err != nil {
		fmt.Println(ColorRed+"Error signing transaction:"+Reset, err)
		return
	}

	err = ctx.chain.AddTransaction(tx)
	if err != nil {
		fmt.Println(ColorRed+"Error: "+Reset+"Failed to add Transaction:\n", err)
	} else {
		fmt.Println(ColorGreen + "Transaction signed and added to the pending pool!" + Reset)
	}
}

func handleMine(ctx *cliContext, args []string) {
	fmt.Println(ColorYellow + FormatDim + "Mining new block..." + Reset)
	startTime := time.Now()
	oldDiff := ctx.chain.Difficulty
	err := ctx.chain.MinePendingTransactions()
	newDiff := ctx.chain.Difficulty
	miningTime := time.Since(startTime)
	if err != nil {
		fmt.Println(ColorRed+"Error: "+Reset+"Failed to mine block:", err)
	} else {
		fmt.Printf(ColorGreen+"Block mined successfully! (Difficulty: %d) Time: %s\n"+Reset, newDiff, miningTime.Round(time.Millisecond))
		if oldDiff != newDiff {
			fmt.Printf(ColorCyan+"Difficulty retargeted from %d to %d\n"+Reset, oldDiff, newDiff)
		}
	}
}

func handlePool(ctx *cliContext, args []string) {
	if len(ctx.chain.PendingPool) == 0 {
		fmt.Println(ColorYellow + "No pending transactions!" + Reset)
	} else {
		wallets, err := wallet.GetAllWallets(ctx.walletFile)
		if err != nil && !os.IsNotExist(err) {
			fmt.Println(ColorRed+"Warning: Failed to load wallets from keystore:"+Reset, err)
		}
		fmt.Println(ColorCyan + "--- Pending Transactions ---" + Reset)
		for i, tx := range ctx.chain.PendingPool {
			senderLabel := getAddressLabel(tx.Sender, wallets)
			recipientLabel := getAddressLabel(tx.Recipient, wallets)
			fmt.Printf("%s%d.%s %s --> %s : %d\n", ColorYellow, i+1, Reset, senderLabel, recipientLabel, tx.Amount)
		}
	}
}

func handleBalances(ctx *cliContext, args []string) {
	balances := ledger.CalculateBalances(ctx.chain.Blocks)
	wallets, err := wallet.GetAllWallets(ctx.walletFile)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println(ColorRed+"Warning: Failed to load wallets from keystore:"+Reset, err)
	}
	fmt.Println(ColorCyan + "--- Account Balances ---" + Reset)
	for acc, bal := range balances {
		label := getAddressLabel(acc, wallets)
		fmt.Printf("%s : %d\n", label, bal)
	}
}

func handleValidate(ctx *cliContext, args []string) {
	result := ctx.chain.Validate()
	if result.IsValid {
		fmt.Println(ColorGreen + "Chain is valid!" + Reset)
	} else {
		fmt.Printf(ColorRed+"Error: "+Reset+"Chain is INVALID. Failed at Block %d: %s\n", result.FailedAtHeight, result.Reason)
	}
}

func handlePrint(ctx *cliContext, args []string) {
	wallets, err := wallet.GetAllWallets(ctx.walletFile)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println(ColorRed+"Warning: Failed to load wallets from keystore:"+Reset, err)
	}
	printBlockchain(ctx.chain, wallets)
}

func handleHelp(ctx *cliContext, args []string) {
	printHelp()
}

func handleClear(ctx *cliContext, args []string) {
	fmt.Print("\033[H\033[2J")
}

func handleExit(ctx *cliContext, args []string) {
	fmt.Println(ColorYellow + "Exiting..." + Reset)
}
