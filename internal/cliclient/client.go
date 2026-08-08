package cliclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"valence/internal/block"
	"valence/internal/wallet"
)

func printJSONResponse(resp *http.Response) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		prettyJSON, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Println(string(prettyJSON))
	} else {
		var parsedArray []interface{}
		if err := json.Unmarshal(body, &parsedArray); err == nil {
			prettyJSON, _ := json.MarshalIndent(parsedArray, "", "  ")
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Println(string(body))
		}
	}
}

func HandleGet(nodeURL, endpoint string) {
	resp, err := http.Get(nodeURL + endpoint)
	if err != nil {
		fmt.Printf("Error connecting to node: %v\n", err)
		os.Exit(1)
	}
	printJSONResponse(resp)
}

func HandlePost(nodeURL, endpoint string) {
	resp, err := http.Post(nodeURL+endpoint, "application/json", nil)
	if err != nil {
		fmt.Printf("Error connecting to node: %v\n", err)
		os.Exit(1)
	}
	printJSONResponse(resp)
}

func HandleFaucet(nodeURL, keystoreFile, walletName string, amount int64) {
	w, err := wallet.LoadFromKeystore(keystoreFile, walletName)
	if err != nil {
		fmt.Printf("Wallet '%s' not found. Use 'valence-cli wallet create' to create one.\n", walletName)
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"address": w.Address(),
		"amount":  amount,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Failed to marshal request: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.Post(nodeURL+"/faucet", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error connecting to node: %v\n", err)
		os.Exit(1)
	}
	printJSONResponse(resp)
}

func HandleSubmitTx(nodeURL, keystoreFile, walletName, toAddr string, amount int64) {
	dir := filepath.Dir(keystoreFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Printf("Failed to create keystore directory: %v\n", err)
		os.Exit(1)
	}

	w, err := wallet.LoadFromKeystore(keystoreFile, walletName)
	if err != nil {
		fmt.Printf("Failed to load wallet '%s': %v\n", walletName, err)
		fmt.Println("To create a wallet, you can run the faucet command or generate one programmatically.")
		os.Exit(1)
	}

	// Fetch next sequence from node
	seqResp, err := http.Get(nodeURL + "/sequence/" + w.Address())
	if err != nil {
		fmt.Printf("Error fetching sequence: %v\n", err)
		os.Exit(1)
	}
	defer seqResp.Body.Close()

	var seqData struct {
		NextSequence uint64 `json:"next_sequence"`
	}
	if err := json.NewDecoder(seqResp.Body).Decode(&seqData); err != nil {
		fmt.Printf("Error parsing sequence response: %v\n", err)
		os.Exit(1)
	}

	tx := block.Transaction{
		Sender:    w.Address(),
		Recipient: toAddr,
		Amount:    amount,
		Sequence:  seqData.NextSequence,
		Timestamp: time.Now().UnixNano(),
	}
	tx.ComputeID()
	tx.Sign(w.PrivateKey)

	body, err := json.Marshal(tx)
	if err != nil {
		fmt.Printf("Failed to marshal transaction: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.Post(nodeURL+"/tx/submit", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error connecting to node: %v\n", err)
		os.Exit(1)
	}
	printJSONResponse(resp)
}
