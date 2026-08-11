package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"valence/internal/cliclient"
)

func printUsage() {
	fmt.Println(cliclient.ColorBlue + cliclient.FormatItalic + "\nAvailable Commands" + cliclient.Reset)
	fmt.Println("  " + cliclient.ColorYellow + "status" + cliclient.Reset + "                        Show node status")
	fmt.Println("  " + cliclient.ColorYellow + "peers" + cliclient.Reset + "                         Show connected peers")
	fmt.Println("  " + cliclient.ColorYellow + "peers add <peer_address>" + cliclient.Reset + "      Add a new peer")
	fmt.Println("  " + cliclient.ColorYellow + "mine" + cliclient.Reset + "                          Trigger mining on the node")
	fmt.Println("  " + cliclient.ColorYellow + "exit" + cliclient.Reset + "                          Exit the interactive console")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
}

func main() {
	nodeFlag := flag.String("node", "localhost:3000", "Node HTTP API address")
	flag.Parse()

	nodeURL := "http://" + *nodeFlag
	if strings.HasPrefix(*nodeFlag, "http") {
		nodeURL = *nodeFlag
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println(cliclient.ColorBlue + "=========================================" + cliclient.Reset)
		fmt.Println(cliclient.FormatBold + "         Node Admin Console              " + cliclient.Reset)
		fmt.Println(cliclient.ColorBlue + "=========================================" + cliclient.Reset)
		printUsage()
		runInteractive(nodeURL)
		return
	}

	runCommand(args, nodeURL)
}

func runInteractive(nodeURL string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n" + cliclient.ColorCyan + "admin> " + cliclient.Reset)
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
		runCommand(args, nodeURL)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func runCommand(args []string, nodeURL string) {
	command := args[0]
	switch command {
	case "status":
		data, err := cliclient.HandleGet(nodeURL, "/status")
		if err != nil {
			cliclient.PrintError(err)
		} else {
			cliclient.PrintGenericJSON(data)
		}
	case "peers":
		if len(args) > 1 && args[1] == "add" {
			if len(args) < 3 {
				fmt.Println("Usage: peers add <peer_address>")
				return
			}
			data, err := cliclient.HandlePeersAdd(nodeURL, args[2])
			if err != nil {
				cliclient.PrintError(err)
			} else {
				cliclient.PrintGenericJSON(data)
			}
		} else {
			data, err := cliclient.HandleGet(nodeURL, "/peers")
			if err != nil {
				cliclient.PrintError(err)
			} else {
				cliclient.PrintGenericJSON(data)
			}
		}
	case "mine":
		data, err := cliclient.HandlePost(nodeURL, "/mine")
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
		fmt.Printf("%sUnknown admin command: %s%s\n", cliclient.ColorRed, command, cliclient.Reset)
	}
}
