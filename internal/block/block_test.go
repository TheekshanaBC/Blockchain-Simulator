package block

import (
	"encoding/json"
	"strings"
	"testing"
)

/*
TestNewGenesisBlock verifies that the very first block of the blockchain
(the Genesis Block) is created with the correct initial properties,
specifically height 0 and the correct predefined previous hash.
It also checks the Genesis transaction, Merkle Root, and Hash.
*/
func TestNewGenesisBlock(t *testing.T) {
	block := NewGenesisBlock()

	if block.Height != 0 {
		t.Errorf("Expected Height 0, got %d", block.Height)
	}

	if block.Header.PrevHash != GenesisPrevHash {
		t.Errorf("Expected PrevHash %s, got %s", GenesisPrevHash, block.Header.PrevHash)
	}

	if len(block.Transactions) != 1 {
		t.Fatalf("Expected exactly 1 transaction in Genesis block, got %d", len(block.Transactions))
	}

	if block.Transactions[0].Sender != "COINBASE" {
		t.Errorf("Expected sender COINBASE, got %s", block.Transactions[0].Sender)
	}

	expectedMerkleRoot := CalculateMerkleRoot(block.Transactions)
	if block.Header.MerkleRoot != expectedMerkleRoot {
		t.Errorf("Expected MerkleRoot %s, got %s", expectedMerkleRoot, block.Header.MerkleRoot)
	}

	expectedHash := block.CalculateHash()
	if block.Hash != expectedHash {
		t.Errorf("Expected Hash %s, got %s", expectedHash, block.Hash)
	}
}

/*
TestCalculateHash ensures that a block's hash changes if its transactions
(and consequently its Merkle Root) change. It also verifies that hashing
the exact same set of transactions twice yields the same hash.
*/
func TestCalculateHash(t *testing.T) {
	block := &Block{
		Header: BlockHeader{
			Timestamp: 1720211552,
			PrevHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Nonce:     12345,
		},
		Height:       1,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}},
	}

	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	hash1 := block.CalculateHash()

	block.Transactions = []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 100}}
	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	hash2 := block.CalculateHash()

	block.Transactions = []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}}
	block.Header.MerkleRoot = CalculateMerkleRoot(block.Transactions)
	hash3 := block.CalculateHash()

	if hash1 == hash2 {
		t.Errorf("Hashes should be different for different Merkle Roots")
	}

	if hash1 != hash3 {
		t.Errorf("Hashes should be same for same Merkle Roots")
	}
}

/*
TestMine tests the Proof-of-Work mining mechanism. It checks that the
resulting block hash starts with the required number of leading zeros
(based on the difficulty level) and that recalculating the hash with the
found nonce produces the same result.
*/
func TestMine(t *testing.T) {
	block := &Block{
		Header: BlockHeader{
			PrevHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Timestamp: 1627890123,
			Nonce:     0,
		},
		Height:       1,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 100}},
	}

	difficulty := 4
	block.Mine(difficulty)

	// check the hash starts with {difficulty} zeros after mining
	target := strings.Repeat("0", difficulty) // target = 0000
	if !strings.HasPrefix(block.Hash, target) {
		t.Errorf("Mined hash does not meet difficulty target. Hash: %s", block.Hash)
	}

	// now nonce is set to a value where blocks hash is starting with {difficulty} zeros.
	// recalculate the hash with the found Nonce amd check is it same with the Mined value
	expectedHash := block.CalculateHash()
	if block.Hash != expectedHash {
		t.Errorf("Hash does not match recalculated hash with found nonce")
	}

}

/*
TestMine_ZeroDifficulty verifies that if difficulty is 0, the block is mined
immediately without needing a specific prefix.
*/
func TestMine_ZeroDifficulty(t *testing.T) {
	block := &Block{
		Header: BlockHeader{PrevHash: "test"},
		Height: 1,
	}
	block.Mine(0)

	expectedHash := block.CalculateHash()
	if block.Hash != expectedHash {
		t.Errorf("Hash does not match recalculated hash with found nonce. Expected %s, got %s", expectedHash, block.Hash)
	}

	if len(block.Transactions) != 1 {
		t.Fatalf("Expected COINBASE transaction to be added")
	}
}

/*
TestMine_ExistingCoinbase ensures that if a block already has a COINBASE
transaction as its first transaction, another one is not prepended during mining.
*/
func TestMine_ExistingCoinbase(t *testing.T) {
	block := &Block{
		Header: BlockHeader{PrevHash: "test"},
		Height: 1,
		Transactions: []Transaction{
			{Sender: "COINBASE", Recipient: "Miner", Amount: 50, Signature: []byte("0")},
			{Sender: "Alice", Recipient: "Bob", Amount: 10},
		},
	}

	initialTxCount := len(block.Transactions)
	block.Mine(1)

	if len(block.Transactions) != initialTxCount {
		t.Errorf("Expected transaction count to remain %d, got %d", initialTxCount, len(block.Transactions))
	}
	if block.Transactions[0].Sender != "COINBASE" {
		t.Errorf("Expected first transaction to be COINBASE")
	}
}

/*
TestMine_NoTransactions verifies that mining an empty block correctly
adds the COINBASE transaction as the miner's reward.
*/
func TestMine_NoTransactions(t *testing.T) {
	block := &Block{
		Header:       BlockHeader{PrevHash: "test"},
		Height:       1,
		Transactions: []Transaction{},
	}

	block.Mine(1)

	if len(block.Transactions) != 1 {
		t.Fatalf("Expected exactly 1 transaction (COINBASE), got %d", len(block.Transactions))
	}
	if block.Transactions[0].Sender != "COINBASE" {
		t.Errorf("Expected COINBASE transaction, got %s", block.Transactions[0].Sender)
	}
}

/*
TestMine_NonceOverflow tests the scenario where the Nonce reaches its maximum
uint32 value. It ensures that the Nonce wraps around to 0, and the ExtraNonce
in the coinbase transaction is incremented to provide new entropy.
*/
func TestMine_NonceOverflow(t *testing.T) {
	block := &Block{
		Header: BlockHeader{
			PrevHash: "test_overflow",
			Nonce:    4294967295, // Set to max uint32
		},
		Height:       1,
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 10}},
	}

	// difficulty 4 ensures it's highly improbable to find a hash on the very first try (1 in 65536)
	block.Mine(4)

	// Since we started at MaxUint32, ExtraNonce must have incremented (in Signature).
	if string(block.Transactions[0].Signature) == "0" {
		t.Errorf("Expected ExtraNonce (in Signature) to be incremented after Nonce overflow")
	}
}

/*
TestBlock_JSONSerialization verifies that a Block and its transactions can be
correctly marshaled to a JSON string and unmarshaled back into a struct without
any data loss or corruption.
*/
func TestBlock_JSONSerialization(t *testing.T) {
	originalBlock := &Block{
		Header: BlockHeader{
			PrevHash:   "prev_hash_123",
			MerkleRoot: "merkle_root_456",
			Timestamp:  1627890123,
			Difficulty: 4,
			Nonce:      42,
		},
		Height: 1,
		Transactions: []Transaction{
			{Sender: "Alice", Recipient: "Bob", Amount: 100, Signature: []byte("1")},
		},
		Hash: "block_hash_789",
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(originalBlock)
	if err != nil {
		t.Fatalf("Failed to marshal block to JSON: %v", err)
	}

	// Deserialize back to a Block struct
	var decodedBlock Block
	err = json.Unmarshal(jsonData, &decodedBlock)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to block: %v", err)
	}

	// Verify fields
	if decodedBlock.Hash != originalBlock.Hash {
		t.Errorf("Expected Hash %s, got %s", originalBlock.Hash, decodedBlock.Hash)
	}
	if decodedBlock.Height != originalBlock.Height {
		t.Errorf("Expected Height %d, got %d", originalBlock.Height, decodedBlock.Height)
	}
	if decodedBlock.Header.Nonce != originalBlock.Header.Nonce {
		t.Errorf("Expected Nonce %d, got %d", originalBlock.Header.Nonce, decodedBlock.Header.Nonce)
	}
	if len(decodedBlock.Transactions) != len(originalBlock.Transactions) {
		t.Fatalf("Expected %d transactions, got %d", len(originalBlock.Transactions), len(decodedBlock.Transactions))
	}
	if decodedBlock.Transactions[0].Amount != originalBlock.Transactions[0].Amount {
		t.Errorf("Expected Transaction Amount %d, got %d", originalBlock.Transactions[0].Amount, decodedBlock.Transactions[0].Amount)
	}
}
