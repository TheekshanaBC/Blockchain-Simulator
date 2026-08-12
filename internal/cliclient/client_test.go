package cliclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"valence/internal/wallet"
)

func TestHandleGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test" {
			t.Errorf("Expected path /test, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	resp, err := HandleGet(server.URL, "/test")
	if err != nil {
		t.Fatalf("HandleGet failed: %v", err)
	}

	respMap, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", resp)
	}
	if respMap["status"] != "ok" {
		t.Errorf("Expected status ok, got %v", respMap["status"])
	}
}

func TestHandlePost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test-post" {
			t.Errorf("Expected path /test-post, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	resp, err := HandlePost(server.URL, "/test-post")
	if err != nil {
		t.Fatalf("HandlePost failed: %v", err)
	}

	respMap, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", resp)
	}
	if respMap["message"] != "success" {
		t.Errorf("Expected message success, got %v", respMap["message"])
	}
}

func TestHandleFaucet(t *testing.T) {
	tempDir := t.TempDir()
	keystoreFile := filepath.Join(tempDir, "keystore.json")
	
	// Create wallet
	err := HandleCreateWallet(keystoreFile, "test_wallet")
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/faucet" {
			t.Errorf("Expected path /faucet, got %s", r.URL.Path)
		}
		
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tx_id": "test_tx"}`))
	}))
	defer server.Close()

	resp, err := HandleFaucet(server.URL, keystoreFile, "test_wallet", 100)
	if err != nil {
		t.Fatalf("HandleFaucet failed: %v", err)
	}
	respMap := resp.(map[string]interface{})
	if respMap["tx_id"] != "test_tx" {
		t.Errorf("Expected test_tx, got %v", respMap["tx_id"])
	}
}

func TestHandleSubmitTx(t *testing.T) {
	tempDir := t.TempDir()
	keystoreFile := filepath.Join(tempDir, "keystore.json")
	
	err := HandleCreateWallet(keystoreFile, "test_wallet")
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	w, _ := wallet.LoadFromKeystore(keystoreFile, "test_wallet")

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sequence/"+w.Address() {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"next_sequence": 1}`))
			return
		}
		
		if r.URL.Path == "/tx/submit" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusAccepted)
			rw.Write([]byte(`{"message": "accepted"}`))
			return
		}
		t.Errorf("Unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	resp, err := HandleSubmitTx(server.URL, keystoreFile, "test_wallet", "recipient_addr", 50)
	if err != nil {
		t.Fatalf("HandleSubmitTx failed: %v", err)
	}
	respMap := resp.(map[string]interface{})
	if respMap["message"] != "accepted" {
		t.Errorf("Expected message accepted, got %v", respMap["message"])
	}
}
