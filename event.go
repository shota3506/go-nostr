package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

type EventKind int64

const (
	EventKindSetMetadata     EventKind = 0
	EventKindTextNote        EventKind = 1
	EventKindRecommendServer EventKind = 2
)

type Tag []string

type Event struct {
	ID        string    `json:"id"`
	PubKey    string    `json:"pubkey"`
	CreatedAt int64     `json:"created_at"`
	Kind      EventKind `json:"kind"`
	Tags      []Tag     `json:"tags"`
	Content   string    `json:"content"`
	Sig       string    `json:"sig"`
}

func (e *Event) serialize() ([]byte, error) {
	b, err := json.Marshal([]any{
		0,
		e.PubKey,
		e.CreatedAt,
		e.Kind,
		e.Tags,
		e.Content,
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (e *Event) Sign(privKey string) error {
	s, err := hex.DecodeString(privKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}
	sk, pk := btcec.PrivKeyFromBytes(s)

	// public key
	e.PubKey = hex.EncodeToString(pk.SerializeCompressed()[1:])

	serial, err := e.serialize()
	if err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}
	serialHash := sha256.Sum256(serial)

	// id
	e.ID = hex.EncodeToString(serialHash[:])

	// signature
	sig, err := schnorr.Sign(sk, serialHash[:])
	if err != nil {
		return err
	}
	e.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}
