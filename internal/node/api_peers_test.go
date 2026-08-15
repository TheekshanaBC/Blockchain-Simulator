package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
TestAPIPeersAnnounce verifies the /peers/announce endpoint.
It ensures that when a new peer announces itself to the node,
the node correctly updates its internal PeerManager with the new addresses.
*/
func TestAPIPeersAnnounce(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	payload := map[string]interface{}{
		"address": "node2:3002",
		"peers":   []string{"node3:3003"},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/peers/announce", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	peers := n.PeerManager.GetPeers()
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %v", len(peers))
	}
}

func TestAPIPeers(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Add peer manually
	n.PeerManager.AddPeer("127.0.0.1:4000")

	req, _ := http.NewRequest("GET", "/peers", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var peers []string
	json.Unmarshal(rr.Body.Bytes(), &peers)

	if len(peers) != 1 || peers[0] != "http://127.0.0.1:4000" {
		t.Errorf("Expected [http://127.0.0.1:4000], got %v", peers)
	}
}
