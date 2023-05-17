package nostr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestClient(t *testing.T) {
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(websocket.StatusInternalError, "")

		for {
			var msg json.RawMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				var closeErr websocket.CloseError
				if errors.As(err, &closeErr) {
					break
				} else {
					t.Fatal(err)
				}
			}
		}

		close(done)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("websocket connection not closed")
	}
}

func TestClientPublish(t *testing.T) {
	event := &Event{
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Kind:      EventKindTextNote,
		Tags:      []Tag{},
		Content:   "short text note",
	}

	// example keys from https://github.com/nostr-protocol/nips/blob/01f90d105d995df7308ef6bea46cc93cdef16ec3/19.md
	privKey := "67dea2ed018072d675f5415ecfaed7d2597555e202d85b3d65ea4e58d2d92ffa"
	if err := event.Sign(privKey); err != nil {
		t.Fatalf("event.Sign() failed: %s", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(websocket.StatusInternalError, "")

		for {
			var b json.RawMessage
			if err := wsjson.Read(ctx, conn, &b); err != nil {
				var closeErr websocket.CloseError
				if errors.As(err, &closeErr) {
					break
				} else {
					t.Fatal(err)
				}
			}

			// emulate relay server
			var message []json.RawMessage
			if err = json.Unmarshal(b, &message); err != nil {
				t.Fatal(err)
			}
			var typ string
			if err = json.Unmarshal(message[0], &typ); err != nil {
				t.Fatal(err)
			}
			if typ != "EVENT" {
				t.Fatal("unexpected message type")
			}
			if len(message) != 2 {
				t.Fatalf("invalid message length: %d", len(message))
			}
			var event Event
			if err = json.Unmarshal(message[1], &event); err != nil {
				t.Fatal(err)
			}
			if err := wsjson.Write(ctx, conn, []any{"OK", event.ID, true, "sample message"}); err != nil {
				t.Fatal(err)
			}
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := client.Publish(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Error("command result not OK")
	}
	if result.Message != "sample message" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}
