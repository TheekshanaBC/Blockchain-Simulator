package block

import (
	"encoding/hex"
	"fmt"
)

type ProofNode struct {
	Hash string `json:"hash"`
	Left bool   `json:"left"`
}

func CalculateMerkleRoot(txs []Transaction) string {
	if len(txs) == 0 {
		return ""
	}

	var hashes []string

	for _, tx := range txs {
		record := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x",
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
				combined := "\x01" + hashes[i] + hashes[i+1]
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

func BuildMerkleProof(txs []Transaction, txIndex int) []ProofNode {
	if len(txs) == 0 || txIndex < 0 || txIndex >= len(txs) {
		return nil
	}

	var hashes []string
	for _, tx := range txs {
		record := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x",
			len(tx.Sender), tx.Sender,
			len(tx.Recipient), tx.Recipient,
			tx.Amount,
			len(tx.PublicKey), tx.PublicKey,
			len(tx.Signature), tx.Signature)
		hashes = append(hashes, doubleSHA256(record))
	}

	var proof []ProofNode
	currentIndex := txIndex

	for len(hashes) > 1 {
		var newLevel []string
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				combined := "\x01" + hashes[i] + hashes[i+1]
				newLevel = append(newLevel, doubleSHA256(combined))

				if i == currentIndex {
					proof = append(proof, ProofNode{Hash: hashes[i+1], Left: false})
				} else if i+1 == currentIndex {
					proof = append(proof, ProofNode{Hash: hashes[i], Left: true})
				}
			} else {
				newLevel = append(newLevel, hashes[i])
			}
		}
		currentIndex = currentIndex / 2
		hashes = newLevel
	}

	return proof
}

func VerifyMerkleProof(tx Transaction, proof []ProofNode, root string) bool {
	record := fmt.Sprintf("\x00%d:%s|%d:%s|%d|%d:%x|%d:%x",
		len(tx.Sender), tx.Sender,
		len(tx.Recipient), tx.Recipient,
		tx.Amount,
		len(tx.PublicKey), tx.PublicKey,
		len(tx.Signature), tx.Signature)
	current := doubleSHA256(record)

	for _, node := range proof {
		if node.Left {
			current = doubleSHA256("\x01" + node.Hash + current)
		} else {
			current = doubleSHA256("\x01" + current + node.Hash)
		}
	}

	return current == root
}
