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
		record := fmt.Sprintf("%d:%s|%d:%s|%d|%d:%x|%d:%x",
			len(tx.Sender), tx.Sender,
			len(tx.Recipient), tx.Recipient,
			tx.Amount,
			len(tx.PublicKey), tx.PublicKey,
			len(tx.Signature), tx.Signature)
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
