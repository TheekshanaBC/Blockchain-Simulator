package node

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"valence/internal/block"
)

func setupTestNode(t *testing.T) *Node {
	cfg := Config{
		Port:            8080,
		DataDir:         t.TempDir(),
		Difficulty:      1,
		RetargetWindow:  4,
		TargetBlockTime: 10,
		MinDifficulty:   1,
		MaxDifficulty:   6,
		MaxTxPerBlock:   10,
		FaucetKey:       "AdUl1LWR0NtSPlR6NktiYVptv2sKOwAZ8djfTt9u1Mk=",
	}
	n, err := NewNode(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	return n
}

var seqCache = make(map[*Node]uint64)

func createTestFaucetTx(n *Node, recipient string, amount int64) block.Transaction {
	seqCache[n]++
	tx, err := n.Chain.CreateFaucetTx(recipient, amount, n.FaucetWallet, seqCache[n], 1_000_000_000*block.ElectronsPerVCN, nil)
	if err != nil {
		panic(err)
	}
	return tx
}

/*
TestAPIStatus verifies the /status endpoint.
It ensures that a newly started node returns HTTP 200 OK
and reports an initial blockchain height of 0.
*/
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

func TestAPIBalances(t *testing.T) {
	n := setupTestNode(t)
	mux := http.NewServeMux()
	n.setupAPI(mux)

	// Faucet gives 100 to Alice
	fTx := createTestFaucetTx(n, "Alice", 100)
	n.Chain.MineBlock(context.Background(), []block.Transaction{fTx}, "Miner")

	req, _ := http.NewRequest("GET", "/balances/Alice", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var balanceResp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &balanceResp)

	balanceFloat, ok := balanceResp["balance"].(float64)
	if !ok {
		t.Fatalf("Expected float64 balance")
	}
	if int64(balanceFloat) != 100 {
		t.Errorf("Expected balance 100, got %v", balanceFloat)
	}
}
