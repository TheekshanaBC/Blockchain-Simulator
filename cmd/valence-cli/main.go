package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"valence/internal/block"
	"valence/internal/cliclient"
	"valence/internal/wallet"
)

// ANSI Color Codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
)

func pause(reader *bufio.Reader) {
	fmt.Print(Cyan + "\nPress Enter to continue..." + Reset)
	reader.ReadString('\n')
}

func printError(err error) {
	errStr := err.Error()
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connectex") {
		fmt.Printf(Red+Bold+"❌ Error: Cannot connect to node. Please ensure the node is running!%s\n", Reset)
	} else if strings.Contains(errStr, "wallet") && strings.Contains(errStr, "not found") {
		fmt.Printf(Red+Bold+"❌ Error: Wallet not found. Please create or load a valid wallet first.%s\n", Reset)
	} else if strings.Contains(errStr, "not configured as a faucet") {
		fmt.Printf(Red+Bold+"❌ Error: This node is not configured as a faucet.\n💡 Tip: Use option 'N' to switch to a Faucet-enabled node (e.g. Cloud Node).%s\n", Reset)
	} else {
		fmt.Printf(Red+Bold+"❌ Error: %v%s\n", err, Reset)
	}
}

func printJSON(data interface{}, err error) {
	if err != nil {
		printError(err)
		return
	}
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(Green + string(b) + Reset)
}

func main() {
	nodeURL := flag.String("node", "http://localhost:8080", "Node URL")
	keystorePath := flag.String("keystore", "./data/wallets/keys.json", "Path to keystore file")
	walletName := flag.String("wallet", "primary", "Wallet name/index to use")
	
	flag.Usage = func() {
		fmt.Println("Usage: valence-cli [global flags] <command> [args]")
		fmt.Println("\nGlobal Flags:")
		flag.PrintDefaults()
		fmt.Println("\nCommands:")
		fmt.Println("  createwallet                   Create a new wallet in the keystore")
		fmt.Println("  getbalance [address]           Get balance of an address (defaults to active wallet)")
		fmt.Println("  sendtoaddress <address> <amt>  Send VCN to an address (amount is in VCN)")
		fmt.Println("  getmempoolinfo                 Get current mempool contents")
		fmt.Println("  faucet <amt>                   Request faucet funds (amount is in VCN)")
		fmt.Println("  getnetworkinfo                 Get node status (blockchain info)")
		fmt.Println("  getpeerinfo                    Get connected peers")
		fmt.Println("  addnode <address>              Manually connect to a peer")
		fmt.Println("  generate                       Manually mine a block")
	}

	// Rearrange os.Args so that flags can appear after subcommands
	var newArgs []string
	var positional []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			newArgs = append(newArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				newArgs = append(newArgs, os.Args[i+1])
				i++
			}
		} else {
			positional = append(positional, arg)
		}
	}
	os.Args = append([]string{os.Args[0]}, append(newArgs, positional...)...)

	flag.Parse()

	if flag.NArg() < 1 {
		runInteractiveMode(*nodeURL, *keystorePath, *walletName)
		return
	}

	command := flag.Arg(0)
	args := flag.Args()[1:] // everything after the command
	runCommand(command, args, *nodeURL, *keystorePath, *walletName)
}

func runInteractiveMode(nodeURL, keystorePath, walletName string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		asciiLogo := "\n" +
			"  _    __      __\n" +
			" | |  / /___ _/ /__  ____  ________\n" +
			" | | / / __ `/ / _ \\/ __ \\/ ___/ _ \\\n" +
			" | |/ / /_/ / /  __/ / / / /__/  __/\n" +
			" |___/\\__,_/_/\\___/_/ /_/\\___/\\___/\n" +
			"       Blockchain Interactive CLI\n"
		
		fmt.Println(Purple + Bold + asciiLogo + Reset)
		fmt.Println(Purple + Bold + "==========================================" + Reset)
		fmt.Printf(Yellow+"Current Node  : %s%s\n", nodeURL, Reset)
		fmt.Printf(Yellow+"Active Wallet : %s%s\n", walletName, Reset)
		
		fmt.Println(Cyan + "\nAvailable commands:" + Reset)
		fmt.Println(Yellow + "\n \U0001f3e6 Wallet & Transactions:" + Reset)
		fmt.Println("  1. Create New Wallet")
		fmt.Println("  2. Set Active Wallet")
		fmt.Println("  3. Get Balance")
		fmt.Println("  4. Send VLC")
		fmt.Println("  5. Request Faucet")
		fmt.Println(Yellow + "\n \u26CF\uFE0F  Blockchain & Mining:" + Reset)
		fmt.Println("  6. View Mempool")
		fmt.Println("  7. Mine Block")
		fmt.Println("  8. Get Network Info")
		fmt.Println(Yellow + "\n \U0001f310 Node & P2P Network:" + Reset)
		fmt.Println("  9. Get Peers")
		fmt.Println("  10. Connect to Peer")
		fmt.Println("  N. Change Target Node URL")
		fmt.Println("\n  0. Exit")
		fmt.Print(Bold + "\nSelect an option: " + Reset)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch strings.ToUpper(input) {
		case "1":
			fmt.Print("Enter name for new wallet: ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)
			if name == "" {
				fmt.Println(Red + "Wallet name cannot be empty." + Reset)
				pause(reader)
				continue
			}
			err := cliclient.HandleCreateWallet(keystorePath, name)
			if err != nil {
				printError(err)
			} else {
				walletName = name
				fmt.Printf(Green+"Wallet created successfully. Active wallet set to '%s'%s\n", name, Reset)
			}
			pause(reader)
		case "2":
			wallets, err := wallet.GetAllWallets(keystorePath)
			if err != nil {
				fmt.Println(Red + "Could not read keystore: " + err.Error() + Reset)
				pause(reader)
				continue
			}
			if len(wallets) == 0 {
				fmt.Println(Yellow + "No wallets found. Please create one first." + Reset)
				pause(reader)
				continue
			}

			var names []string
			for name := range wallets {
				names = append(names, name)
			}
			sort.Strings(names)

			fmt.Println(Cyan + "\nAvailable Wallets:" + Reset)
			for i, name := range names {
				fmt.Printf(" %d. %s\n", i+1, name)
			}

			fmt.Print(Bold + "\nSelect wallet by number: " + Reset)
			selection, _ := reader.ReadString('\n')
			selection = strings.TrimSpace(selection)

			idx, err := strconv.Atoi(selection)
			if err != nil || idx < 1 || idx > len(names) {
				fmt.Println(Red + "Invalid selection." + Reset)
				pause(reader)
				continue
			}

			walletName = names[idx-1]
			fmt.Printf(Green+"Active wallet set to '%s'%s\n", walletName, Reset)
			pause(reader)
		case "3":
			fmt.Print("Enter address (leave blank for active wallet): ")
			addr, _ := reader.ReadString('\n')
			addr = strings.TrimSpace(addr)
			var args []string
			if addr != "" {
				args = append(args, addr)
			}
			runCommand("getbalance", args, nodeURL, keystorePath, walletName)
			pause(reader)
		case "4":
			fmt.Print("Enter recipient address: ")
			addr, _ := reader.ReadString('\n')
			addr = strings.TrimSpace(addr)
			fmt.Print("Enter amount (VLC): ")
			amt, _ := reader.ReadString('\n')
			amt = strings.TrimSpace(amt)
			if addr == "" || amt == "" {
				fmt.Println(Red + "Address and amount are required." + Reset)
				pause(reader)
				continue
			}
			runCommand("sendtoaddress", []string{addr, amt}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "5":
			fmt.Print("Enter amount (VLC): ")
			amt, _ := reader.ReadString('\n')
			amt = strings.TrimSpace(amt)
			if amt == "" {
				fmt.Println(Red + "Amount is required." + Reset)
				pause(reader)
				continue
			}
			runCommand("faucet", []string{amt}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "6":
			runCommand("getmempoolinfo", []string{}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "7":
			runCommand("generate", []string{}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "8":
			runCommand("getnetworkinfo", []string{}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "9":
			runCommand("getpeerinfo", []string{}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "10":
			fmt.Print("Enter peer address (e.g. localhost:8081): ")
			addr, _ := reader.ReadString('\n')
			addr = strings.TrimSpace(addr)
			if addr == "" {
				fmt.Println(Red + "Peer address is required." + Reset)
				pause(reader)
				continue
			}
			runCommand("addnode", []string{addr}, nodeURL, keystorePath, walletName)
			pause(reader)
		case "N":
			fmt.Println(Cyan + "\nAvailable Nodes:" + Reset)
			fmt.Println(" 1. Local Node 1 (http://localhost:8080)")
			fmt.Println(" 2. Local Node 2 (http://localhost:8081)")
			fmt.Println(" 3. Local Node 3 (http://localhost:8082)")
			fmt.Println(" 4. Cloud Node   (https://blockchain-simulator-production.up.railway.app)")
			fmt.Println(" 5. Enter custom URL manually")

			fmt.Print(Bold + "\nSelect node by number: " + Reset)
			nodeSel, _ := reader.ReadString('\n')
			nodeSel = strings.TrimSpace(nodeSel)

			var newNode string
			switch nodeSel {
			case "1":
				newNode = "http://localhost:8080"
			case "2":
				newNode = "http://localhost:8081"
			case "3":
				newNode = "http://localhost:8082"
			case "4":
				newNode = "https://blockchain-simulator-production.up.railway.app"
			case "5":
				fmt.Print("Enter custom Node URL (e.g. http://192.168.1.5:8080): ")
				custom, _ := reader.ReadString('\n')
				custom = strings.TrimSpace(custom)
				if custom != "" {
					if !strings.HasPrefix(custom, "http://") && !strings.HasPrefix(custom, "https://") {
						custom = "http://" + custom
					}
					newNode = custom
				}
			default:
				fmt.Println(Red + "Invalid selection." + Reset)
			}

			if newNode != "" {
				nodeURL = newNode
				fmt.Printf(Green+"Node URL updated to %s%s\n", nodeURL, Reset)
			}
			pause(reader)
		case "0", "EXIT", "QUIT":
			fmt.Println(Green + "Goodbye!" + Reset)
			return
		default:
			fmt.Println(Red + "Invalid option. Please try again." + Reset)
			pause(reader)
		}
	}
}

func runCommand(command string, args []string, nodeURL, keystorePath, walletName string) {
	switch command {
	case "createwallet":
		err := cliclient.HandleCreateWallet(keystorePath, walletName)
		if err != nil {
			printError(err)
			return
		}
		// If used directly via CLI, we just print success
		fmt.Println(Green + "Wallet operation successful." + Reset)
	case "getbalance":
		addr := walletName 
		if len(args) > 0 {
			addr = args[0]
		} else {
			w, err := wallet.LoadFromKeystore(keystorePath, walletName)
			if err != nil {
				printError(fmt.Errorf("Wallet '%s' not found. Please provide an address or use createwallet.", walletName))
				return
			}
			addr = w.Address()
		}
		res, err := cliclient.HandleGet(nodeURL, "/balances/"+addr)
		printJSON(res, err)

	case "sendtoaddress":
		if len(args) < 2 {
			fmt.Println(Red + "Usage: valence-cli sendtoaddress <address> <amount>" + Reset)
			return
		}
		toAddr := args[0]
		amountStr := args[1]
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			printError(fmt.Errorf("Invalid amount: %v", err))
			return
		}
		amountInt := int64(math.Round(amountFloat * float64(block.ElectronsPerVCN)))
		res, err := cliclient.HandleSubmitTx(nodeURL, keystorePath, walletName, toAddr, amountInt)
		printJSON(res, err)

	case "getmempoolinfo":
		res, err := cliclient.HandleGet(nodeURL, "/mempool")
		printJSON(res, err)

	case "faucet":
		if len(args) < 1 {
			fmt.Println(Red + "Usage: valence-cli faucet <amount>" + Reset)
			return
		}
		amountStr := args[0]
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			printError(fmt.Errorf("Invalid amount: %v", err))
			return
		}
		amountInt := int64(math.Round(amountFloat * float64(block.ElectronsPerVCN)))
		res, err := cliclient.HandleFaucet(nodeURL, keystorePath, walletName, amountInt)
		printJSON(res, err)

	case "getnetworkinfo", "getblockchaininfo":
		res, err := cliclient.HandleGet(nodeURL, "/status")
		printJSON(res, err)

	case "getpeerinfo":
		res, err := cliclient.HandleGet(nodeURL, "/peers")
		printJSON(res, err)

	case "addnode":
		if len(args) < 1 {
			fmt.Println(Red + "Usage: valence-cli addnode <address>" + Reset)
			return
		}
		res, err := cliclient.HandlePeersAdd(nodeURL, args[0])
		printJSON(res, err)

	case "generate":
		res, err := cliclient.HandlePost(nodeURL, "/mine")
		printJSON(res, err)

	default:
		fmt.Printf(Red+"Unknown command: %s%s\n", command, Reset)
	}
}
