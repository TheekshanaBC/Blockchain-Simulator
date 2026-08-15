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
TestAPIGossipTx_FaucetLimitProtection ensures that if an unsigned or forged transaction
claiming to be from the Faucet wallet is gossiped to the /tx/gossip endpoint,
the core ledger validation rejects it with HTTP 400 Bad Request.
*/
func TestAPIGossipTx_FaucetLimitProtection(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	tx := block.Transaction{
		Sender:    n.FaucetWallet.Address(),
		Recipient: n.Wallet.Address(),
		Amount:    2000 * block.ElectronsPerVCN, // Try to forge 2000 VCN (Limit is 1000)
	}
	tx.ComputeID()

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/gossip", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Since faucet transactions are allowed through the gossip boundary, it gets blocked by the Ledger validation with 400
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for forged large faucet tx: got %v want %v", status, http.StatusBadRequest)
	}
	t.Logf("Success! Forged transaction blocked by Ledger. Status: %v, Body: %v", rr.Code, rr.Body.String())
}
