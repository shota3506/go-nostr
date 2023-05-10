package nostr_test

import (
	"context"
	"time"

	"github.com/shota3506/go-nostr"
)

func ExampleClient_Publish() {
	client, err := nostr.NewClient("URL") // TODO: Replace URL with a relay server URL.
	if err != nil {
		// TODO: Handle error.
	}
	defer client.Close()

	event := &nostr.Event{
		CreatedAt: time.Now().Unix(),
		Kind:      nostr.EventKindTextNote,
		Content:   "Hello, World!",
		Tags:      []nostr.Tag{},
	}
	event.Sign("PRIVATE_KEY") // TODO: Replace PRIVATE_KEY with your private key.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Publish(ctx, event)
	if err != nil {
		// TODO: Handle error.
	}

	_ = result // TODO: Check result.
}

func ExampleClient_Subscribe() {
	client, err := nostr.NewClient("URL") // TODO: Replace URL with a relay server URL.
	if err != nil {
		// TODO: Handle error.
	}
	defer client.Close()

	ctx := context.Background()
	sub, err := client.Subscribe(ctx, []nostr.Filter{{}})
	if err != nil {
		// TODO: Handle error.
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = sub.Receive(ctx, func(ctx context.Context, event *nostr.Event) {
		// TODO: Do something with event.
	})
}
