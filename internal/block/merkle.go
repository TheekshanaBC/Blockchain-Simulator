package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func CalculateMerkleRoot(txs []Transaction) string {
	if len(txs) == 0 {
		return ""
	}

	var hashes []string

	for _, tx := range txs {
		record := fmt.Sprintf("%s|%s|%d|%d", tx.Sender, tx.Recipient, tx.Amount, tx.ExtraNonce)
		hashes = append(hashes, doubleSHA256(record))
	}

	for len(hashes) > 1 {
		// if number of transactions is odd duplicate last hash
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var newLevel []string
		for i := 0; i < len(hashes); i += 2 {
			combined := hashes[i] + hashes[i+1]
			newLevel = append(newLevel, doubleSHA256(combined))
		}

		hashes = newLevel
	}

	return hashes[0] // return the root hash value

}

func doubleSHA256(data string) string {
	h1 := sha256.New()
	h1.Write([]byte(data))
	hash1 := h1.Sum(nil)

	h2 := sha256.New()
	h2.Write(hash1)
	hash2 := h2.Sum(nil)

	return hex.EncodeToString(hash2)
}
