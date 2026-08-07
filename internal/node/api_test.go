package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"valence/internal/block"
)

func setupTestNode(t *testing.T) *Node {
	cfg := Config{
		Port:            3000,
		DataDir:         t.TempDir(),
		Difficulty:      1,
		RetargetWindow:  4,
		TargetBlockTime: 10,
	}
	return NewNode(cfg)
}

func TestAPIStatus(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	req, _ := http.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response["height"].(float64) != 0 {
		t.Errorf("Expected height 0, got %v", response["height"])
	}
}

func TestAPISubmitTx_InvalidSignature(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Create a transaction but don't sign it properly
	tx := block.Transaction{
		Sender:    "Alice",
		Recipient: "Bob",
		Amount:    100,
	}
	tx.ComputeID()
	// No signature

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/submit", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestAPISubmitTx_ValidSignature(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	tx := block.Transaction{
		Sender:    n.Wallet.Address(),
		Recipient: "Bob",
		Amount:    100,
	}
	tx.ComputeID()
	tx.Sign(n.Wallet.PrivateKey)

	body, _ := json.Marshal(tx)
	req, _ := http.NewRequest("POST", "/tx/submit", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v (body: %s)", status, http.StatusAccepted, rr.Body.String())
	}

	if !n.Mempool.Has(tx.ID) {
		t.Error("Transaction was not added to mempool")
	}
}
