package wallet

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/big"
)

type Wallet struct {
	PrivateKey     *ecdsa.PrivateKey
	PublicKeyBytes []byte
}

func marshalUncompressed(curve elliptic.Curve, x, y *big.Int) []byte {
	byteLen := (curve.Params().BitSize + 7) / 8
	ret := make([]byte, 1+2*byteLen)
	ret[0] = 4 // uncompressed point
	x.FillBytes(ret[1 : 1+byteLen])
	y.FillBytes(ret[1+byteLen : 1+2*byteLen])
	return ret
}

func unmarshalUncompressed(curve elliptic.Curve, data []byte) (x, y *big.Int) {
	// Map elliptic.Curve to its crypto/ecdh equivalent
	var ecdhCurve ecdh.Curve
	if curve == elliptic.P256() {
		ecdhCurve = ecdh.P256()
	} else if curve == elliptic.P384() {
		ecdhCurve = ecdh.P384()
	} else if curve == elliptic.P521() {
		ecdhCurve = ecdh.P521()
	}

	if ecdhCurve == nil {
		return nil, nil // Unsupported curve
	}

	// NewPublicKey verifies the SEC1 encoding shape and checks that the point is on the curve in constant-time
	_, err := ecdhCurve.NewPublicKey(data)
	if err != nil {
		return nil, nil
	}

	// Data is now guaranteed to be safe and correctly sized
	byteLen := (curve.Params().BitSize + 7) / 8
	x = new(big.Int).SetBytes(data[1 : 1+byteLen])
	y = new(big.Int).SetBytes(data[1+byteLen:])
	
	return x, y
}

// generate brand new random private key and derive matching public key. return both wrapped in a Wallet
func NewWallet() *Wallet {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pubKeyBytes := marshalUncompressed(privateKey.PublicKey.Curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)

	return &Wallet{
		PrivateKey:     privateKey,
		PublicKeyBytes: pubKeyBytes,
	}
}

// takes raw public key bytes and turn them back into usable ecdsa.Public key object
func BytesToPublicKey(pubKeyBytes []byte) *ecdsa.PublicKey {
	curve := elliptic.P256()
	x, y := unmarshalUncompressed(curve, pubKeyBytes)
	if x == nil {
		log.Println("Error parsing public key: invalid bytes")
		return nil
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
}

// generates a simple address string from public key bytes
func AddressFromPublicKey(pubKeyBytes []byte) string {
	hash := sha256.Sum256(pubKeyBytes)
	return hex.EncodeToString(hash[:])
}
