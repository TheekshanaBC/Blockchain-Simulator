package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"valence/internal/block"
)

/*
TestAPIGossipTx_FaucetLimitProtection ensures that even if a malicious node 
bypasses the /faucet API and directly gossips a forged FAUCET transaction 
to the /tx/gossip endpoint, the core Ledger validation catches it and 
blocks it if the requested amount exceeds the limits (1000 VCN).
*/
func TestAPIGossipTx_FaucetLimitProtection(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	tx := block.Transaction{
		Sender:    block.SystemAddressFaucet,
		Recipient: n.Wallet.Address(),
		Amount:    2000 * block.ElectronsPerVCN, // Try to forge 2000 VCN (Limit is 1000)
	}
	tx.ComputeID()

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/gossip", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Since 2000 > 1000, ledger validation should catch it and return 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for forged large faucet tx: got %v want %v", status, http.StatusBadRequest)
	}
	t.Logf("Success! Forged transaction blocked by Ledger. Status: %v, Body: %v", rr.Code, rr.Body.String())
}
