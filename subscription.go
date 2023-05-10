package nostr

import (
	"context"
	"time"
)

type Subscription struct {
	id string

	eventChan <-chan *Event
	eoseChan  <-chan struct{}

	trigger func(context.Context) error
	closer  func(context.Context) error
}

func (s *Subscription) ID() string {
	return s.id
}

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
