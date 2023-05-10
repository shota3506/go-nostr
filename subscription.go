package nostr

import (
	"context"
	"time"
)

// A Subscription is a subscription to a channel.
type Subscription struct {
	id string

	eventChan <-chan *Event
	eoseChan  <-chan struct{}

	trigger func(context.Context) error
	closer  func(context.Context) error
}

// ID returns the subscription ID.
func (s *Subscription) ID() string {
	return s.id
}

// Receive calls f for each event received from the subscription.
// If ctx is done, Receive returns nil.
//
// The context passed to f will be canceled when ctx is Done or there is a fatal service error.
func (s *Subscription) Receive(ctx context.Context, f func(context.Context, *Event)) error {
	done := make(chan struct{})
	defer close(done)

	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
	outer:
		for {
			select {
			case event := <-s.eventChan:
				f(innerCtx, event)
			case <-done:
				break outer
			}
		}
	}()

	if err := s.trigger(ctx); err != nil {
		return err
	}

	defer func() {
		// use new context to close subscription
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		s.closer(closeCtx)
	}()

	<-ctx.Done()

	return nil
}
