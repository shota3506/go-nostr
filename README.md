# go-nostr

go-nostr is a package that provides a simple interface to access nostr relay servers.

[![Go Reference](https://pkg.go.dev/badge/github.com/shota3506/go-nostr.svg)](https://pkg.go.dev/github.com/shota3506/go-nostr)

## Installation

```bash
go get -u github.com/shota3506/go-nostr
```

## Example Usage

Publish event.

```go
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
```

Subscribe events.

```go
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
```
