package nostr

import (
	"testing"
	"time"
)

func TestEventSign(t *testing.T) {
	event := &Event{
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Kind:      EventKindTextNote,
		Tags:      []Tag{},
		Content:   "short text note",
	}

	// example keys from https://github.com/nostr-protocol/nips/blob/01f90d105d995df7308ef6bea46cc93cdef16ec3/19.md
	privKey := "67dea2ed018072d675f5415ecfaed7d2597555e202d85b3d65ea4e58d2d92ffa"
	pubKey := "7e7e9c42a91bfef19fa929e5fda1b72e0ebc1a4c1141673e2794234d86addf4e"

	// expected values
	id := "f926f58579b974014c091f4d945e8e3de7f3f87bbc4a0b6a49f2b3d68be2c89d"
	sig := "7903b45c7863f053bb1e84e6308c0de6f2dd212a9496b2391c83859fec17a3f28427ce74e59deef34ff5c418d871601eb4b8c7a81390f4a3ccb08ba4bce55710"

	if err := event.Sign(privKey); err != nil {
		t.Fatalf("event.Sign() failed: %s", err)
	}

	if event.ID != id {
		t.Errorf("event.ID is %s, expected %s", event.ID, id)
	}
	if event.PubKey != pubKey {
		t.Errorf("event.PubKey is %s, expected %s", event.PubKey, pubKey)
	}
	if event.Sig != sig {
		t.Errorf("event.Sig is %s, expected %s", event.Sig, sig)
	}
}
