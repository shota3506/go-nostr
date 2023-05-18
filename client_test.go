package nostr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "")

		for {
			if err := wsjson.Read(ctx, conn, &json.RawMessage{}); err != nil {
				var closeErr websocket.CloseError
				if errors.As(err, &closeErr) {
					break
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClientPublish(t *testing.T) {
	event := &Event{
		CreatedAt: time.Now().Unix(),
		Kind:      EventKindTextNote,
		Tags:      []Tag{},
		Content:   "short text note",
	}

	privKey, err := NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := event.Sign(privKey); err != nil {
		t.Fatalf("event.Sign() failed: %s", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "")

		for {
			var b json.RawMessage
			if err := wsjson.Read(ctx, conn, &b); err != nil {
				var closeErr websocket.CloseError
				if errors.As(err, &closeErr) {
					break
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			// emulate relay server
			var message []json.RawMessage
			if err = json.Unmarshal(b, &message); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var typ string
			if err = json.Unmarshal(message[0], &typ); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if typ != "EVENT" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if len(message) != 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var event Event
			if err = json.Unmarshal(message[1], &event); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := wsjson.Write(ctx, conn, []any{"OK", event.ID, true, "sample message"}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
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

func TestClientSubscribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "")

		for {
			var b json.RawMessage
			if err := wsjson.Read(ctx, conn, &b); err != nil {
				var closeErr websocket.CloseError
				if errors.As(err, &closeErr) {
					break
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			var message []json.RawMessage
			if err = json.Unmarshal(b, &message); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var typ string
			if err = json.Unmarshal(message[0], &typ); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			switch typ {
			case "REQ":
				if len(message) < 3 {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				var subscriptionID string
				if err = json.Unmarshal(message[1], &subscriptionID); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				go func(subscriptionID string) {
					for i := 0; i < 2; i++ {
						event := &Event{
							CreatedAt: time.Now().Unix(),
							Kind:      EventKindTextNote,
							Tags:      []Tag{},
							Content:   "short text note",
						}

						privKey, err := NewPrivateKey()
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						if err := event.Sign(privKey); err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						if err := wsjson.Write(ctx, conn, []any{"EVENT", subscriptionID, event}); err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
					}

					if err := wsjson.Write(ctx, conn, []any{"EOSE", subscriptionID}); err != nil {
						return
					}
				}(subscriptionID)
			case "CLOSE":
				if len(message) != 2 {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				var subscriptionID string
				if err = json.Unmarshal(message[1], &subscriptionID); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			default:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	sub, err := client.Subscribe(ctx, []Filter{{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var receivedEOSE atomic.Int32
	go func() {
		<-sub.EOSE()
		receivedEOSE.Add(1)
		cancel()
	}()

	var received atomic.Int32
	err = sub.Receive(ctx, func(_ context.Context, event *Event) {
		received.Add(1)
	})
	if err != nil {
		t.Fatal(err)
	}
	if count := received.Load(); count != 2 {
		t.Fatalf("unexpected number of received events: %d", count)
	}
	if count := receivedEOSE.Load(); count != 1 {
		t.Fatalf("unexpected number of received EOSE message: %d", count)
	}
}
