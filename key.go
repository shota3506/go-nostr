package nostr

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec/v2"
)

// NewPrivateKey generates a new private key that is suitable
// for use with secp256k1.
func NewPrivateKey() (string, error) {
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key.Serialize()), nil
}

// PublicKeyFromPrivateKey returns the public key
// that corresponds to the given private key.
func PublicKeyFromPrivateKey(privKey string) (string, error) {
	s, err := hex.DecodeString(privKey)
	if err != nil {
		return "", err
	}
	_, pk := btcec.PrivKeyFromBytes(s)
	return hex.EncodeToString(pk.SerializeCompressed()[1:]), nil
}
