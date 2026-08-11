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

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func parseJSONResponse(resp *http.Response) (interface{}, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		return parsed, nil
	}

	var parsedArray []interface{}
	if err := json.Unmarshal(body, &parsedArray); err == nil {
		return parsedArray, nil
	}

	return string(body), nil
}

func HandleGet(nodeURL, endpoint string) (interface{}, error) {
	resp, err := httpClient.Get(nodeURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("error connecting to node: %v", err)
	}
	return parseJSONResponse(resp)
}

func HandlePost(nodeURL, endpoint string) (interface{}, error) {
	resp, err := httpClient.Post(nodeURL+endpoint, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("error connecting to node: %v", err)
	}
	return parseJSONResponse(resp)
}

func HandleFaucet(nodeURL, keystoreFile, walletName string, amount int64) (interface{}, error) {
	w, err := wallet.LoadFromKeystore(keystoreFile, walletName)
	if err != nil {
		return nil, fmt.Errorf("wallet '%s' not found. Use 'create-wallet' to create one", walletName)
	}

	payload := map[string]interface{}{
		"address": w.Address(),
		"amount":  amount,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	resp, err := httpClient.Post(nodeURL+"/faucet", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error connecting to node: %v", err)
	}
	return parseJSONResponse(resp)
}

func HandleSubmitTx(nodeURL, keystoreFile, walletName, toAddr string, amount int64) (interface{}, error) {
	dir := filepath.Dir(keystoreFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %v", err)
	}

	w, err := wallet.LoadFromKeystore(keystoreFile, walletName)
	if err != nil {
		return nil, fmt.Errorf("failed to load wallet '%s': %v. Create one first", walletName, err)
	}

	// Fetch next sequence from node
	seqResp, err := httpClient.Get(nodeURL + "/sequence/" + w.Address())
	if err != nil {
		return nil, fmt.Errorf("error fetching sequence: %v", err)
	}
	defer seqResp.Body.Close()

	var seqData struct {
		NextSequence uint64 `json:"next_sequence"`
	}
	if err := json.NewDecoder(seqResp.Body).Decode(&seqData); err != nil {
		return nil, fmt.Errorf("error parsing sequence response: %v", err)
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
		return nil, fmt.Errorf("failed to marshal transaction: %v", err)
	}
	resp, err := httpClient.Post(nodeURL+"/tx/submit", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error connecting to node: %v", err)
	}
	return parseJSONResponse(resp)
}

func HandlePeersAdd(nodeURL, peerAddr string) (interface{}, error) {
	payload := map[string]interface{}{
		"address": peerAddr,
		"peers":   []string{},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	resp, err := httpClient.Post(nodeURL+"/peers/announce", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error connecting to node: %v", err)
	}
	return parseJSONResponse(resp)
}

func HandleCreateWallet(keystoreFile, walletName string) error {
	dir := filepath.Dir(keystoreFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create keystore directory: %v", err)
	}
	w := wallet.NewWallet()
	err := wallet.SaveToKeystore(keystoreFile, walletName, w)
	if err != nil {
		return fmt.Errorf("failed to save wallet: %v", err)
	}
	fmt.Printf("Wallet '%s' created successfully! Address: %s\n", walletName, w.Address())
	return nil
}
