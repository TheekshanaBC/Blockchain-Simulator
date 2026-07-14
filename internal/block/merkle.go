package block

import (
	"encoding/hex"
	"fmt"
)

func CalculateMerkleRoot(txs []Transaction) string {
	if len(txs) == 0 {
		return ""
	}

	var hashes []string

	for _, tx := range txs {
		record := fmt.Sprintf("%s|%s|%d|%x|%x", tx.Sender, tx.Recipient, tx.Amount, tx.PublicKey, tx.Signature)
		hashes = append(hashes, doubleSHA256(record))
	}

	for len(hashes) > 1 {
		var newLevel []string
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				combined := hashes[i] + hashes[i+1]
				newLevel = append(newLevel, doubleSHA256(combined))
			} else {
				// odd node: promote unchanged to next level
				newLevel = append(newLevel, hashes[i])
			}
		}

		hashes = newLevel
	}

	return hashes[0] // return the root hash value

}

func doubleSHA256(data string) string {
	hashBytes := DoubleHashBytes([]byte(data))
	return hex.EncodeToString(hashBytes)
}
