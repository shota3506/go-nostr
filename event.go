package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// EventKind is the kind of an event.
type EventKind int64

const (
	EventKindSetMetadata             EventKind = 0    // NIP-01
	EventKindTextNote                EventKind = 1    // NIP-01
	EventKindRecommendServer         EventKind = 2    // NIP-01
	EventKindContacts                EventKind = 3    // NIP-02
	EventKindEncryptedDirectMessages EventKind = 4    // NIP-04
	EventKindEventDeletion           EventKind = 5    // NIP-09
	EventKindReposts                 EventKind = 6    // NIP-18
	EventKindReaction                EventKind = 7    // NIP-25
	EventKindBadgeAward              EventKind = 8    // NIP-58
	EventKindChannelCreation         EventKind = 40   // NIP-28
	EventKindChannelMetadata         EventKind = 41   // NIP-28
	EventKindChannelMessage          EventKind = 42   // NIP-28
	EventKindChannelHideMessage      EventKind = 43   // NIP-28
	EventKindChannelMuteUser         EventKind = 44   // NIP-28
	EventKindFileMetadata            EventKind = 1063 // NIP-94
	EventKindReporting               EventKind = 1984 // NIP-56
	EventKindZapRequest              EventKind = 9734 // NIP-57
	EventKindZap                     EventKind = 9735 // NIP-57
)

// Tag is a tag of an event.
type Tag []string

// Event is an Nostr event.
type Event struct {
	ID        string    `json:"id"`
	PubKey    string    `json:"pubkey"`
	CreatedAt int64     `json:"created_at"`
	Kind      EventKind `json:"kind"`
	Tags      []Tag     `json:"tags"`
	Content   string    `json:"content"`
	Sig       string    `json:"sig"`
}

// Sing signs the event with the given private key.
// It sets the ID, PubKey, and Sig fields.
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
