package node

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegration_TwoNodes(t *testing.T) {
	// Setup Node 1
	cfg1 := Config{Port: 0, DataDir: t.TempDir(), MaxTxPerBlock: 10, FaucetKey: "AdUl1LWR0NtSPlR6NktiYVptv2sKOwAZ8djfTt9u1Mk="}
	node1, err := NewNode(cfg1)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	mux1 := http.NewServeMux()
	node1.setupAPI(mux1)
	server1 := httptest.NewServer(mux1)
	defer server1.Close()

	// Setup Node 2
	cfg2 := Config{Port: 0, DataDir: t.TempDir(), MaxTxPerBlock: 10}
	node2, err := NewNode(cfg2)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}
	mux2 := http.NewServeMux()
	node2.setupAPI(mux2)
	server2 := httptest.NewServer(mux2)
	defer server2.Close()

	// Connect Node 2 to Node 1
	node2.PeerManager.AddPeer(server1.URL)
	node1.PeerManager.AddPeer(server2.URL)

	// Use /faucet endpoint on Node 1
	payload := map[string]interface{}{
		"address": "IntegrationUser",
		"amount":  100,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", server1.URL+"/faucet", strings.NewReader(string(body)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Failed to request faucet from Node 1: %v, status: %v", err, resp.StatusCode)
	}

	// We expect the faucet transaction to be mined and eventually gossiped
	fTxID := ""
	var respData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respData)
	if id, ok := respData["tx_id"].(string); ok {
		fTxID = id
	}

	// Wait for gossip/sync
	assertEventually(t, func() bool {
		// Node 2 should have the Faucet Tx in its mempool (since it was gossiped)
		if node2.Mempool.Has(fTxID) {
			return true
		}
		
		// Also trigger sync in case it's in blocks (if it was mined)
		node2.runSync()
		return false
	}, 3*time.Second, "Node 2 did not receive the transaction from Node 1")
}

func assertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(msg)
}
