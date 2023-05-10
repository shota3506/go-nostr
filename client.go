package nostr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

type subChannelGroup struct {
	eventChan chan<- *Event
	eoseChan  chan<- struct{}
}

type eventChannelGroup struct {
	okChan chan<- *CommandResult
}

// A Client is a Nostr client that connects to a relay server.
type Client struct {
	conn *websocket.Conn

	done chan struct{}

	noticeChan chan string
	subMap     sync.Map // map[string]*subChannelGroup
	eventMap   sync.Map // map[string]*subChannelGroup

	closeOnce sync.Once
	closeErr  error
}

// NewClient creates a new Nostr client.
// It establishes a websocket connection to the relay server.
func NewClient(url string) (*Client, error) {
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket connection error: %w", err)
	}
	// disable read limit
	conn.SetReadLimit(math.MaxInt64 - 1)

	done := make(chan struct{})
	client := &Client{
		conn:       conn,
		done:       done,
		noticeChan: make(chan string, 100),
	}

	go func() {
	outer:
		for {
			select {
			case <-done:
				break outer
			default:
			}

			if err := client.readMessage(ctx); err != nil {
				continue
			}
		}
	}()

	return client, nil
}

// Publish submits an event to the relay server and waits for the command result.
func (c *Client) Publish(ctx context.Context, event *Event) (*CommandResult, error) {
	id := event.ID
	okChan := make(chan *CommandResult, 1)

	c.eventMap.Store(id, &eventChannelGroup{
		okChan: okChan,
	})
	defer c.eventMap.Delete(id)

	if err := c.writeMessage(ctx, &EventMessage{Event: event}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("missing command result: %w", ctx.Err())
	case result := <-okChan:
		return result, nil
	}
}

// Subscribe creates a subscription to the relay server with the given filters.
func (c *Client) Subscribe(ctx context.Context, filters []Filter) (*Subscription, error) {
	if len(filters) == 0 {
		return nil, errors.New("at least one filter is required")
	}

	id := uuid.New().String()
	eventChan := make(chan *Event)
	eoseChan := make(chan struct{}, 1)

	trigger := func(ctx context.Context) error {
		// register subscription to client
		c.subMap.Store(id, &subChannelGroup{
			eventChan: eventChan,
			eoseChan:  eoseChan,
		})

		req := ReqMessage{
			SubscriptionID: id,
			Filters:        filters,
		}
		if err := c.writeMessage(ctx, &req); err != nil {
			// unregister subscription from client
			c.subMap.Delete(id)
			return err
		}
		return nil
	}

	closer := func(ctx context.Context) error {
		req := CloseMessage{SubscriptionID: id}
		if err := c.writeMessage(ctx, &req); err != nil {
			return err
		}

		// unregister subscription from client
		c.subMap.Delete(id)
		return nil
	}

	return &Subscription{
		id:        id,
		eventChan: eventChan,
		eoseChan:  eoseChan,
		trigger:   trigger,
		closer:    closer,
	}, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)

		err := c.conn.Close(websocket.StatusNormalClosure, "")
		if err != nil {
			c.closeErr = err
			return
		}
	})
	return c.closeErr
}

func (c *Client) writeMessage(ctx context.Context, message json.Marshaler) error {
	body, err := message.MarshalJSON()
	if err != nil {
		return err
	}
	err = c.conn.Write(ctx, websocket.MessageText, body)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) readMessage(ctx context.Context) error {
	_, b, err := c.conn.Read(ctx)
	if err != nil {
		return err
	}

	var message []json.RawMessage
	if err = json.Unmarshal(b, &message); err != nil {
		return err
	}
	if len(message) == 0 {
		return errors.New("empty message")
	}
	var typ string
	if err = json.Unmarshal(message[0], &typ); err != nil {
		return err
	}

	switch typ {
	case string(MessageTypeNotice):
		if len(message) != 2 {
			return fmt.Errorf("invalid notice message length: %d", len(message))
		}
		var m string
		if err = json.Unmarshal(message[1], &m); err != nil {
			return err
		}
		c.handleNoticeMessage(&NoticeMessage{Message: m})
		return nil
	case string(MessageTypeEvent):
		if len(message) != 3 {
			return fmt.Errorf("invalid event message length: %d", len(message))
		}
		var subID string
		if err = json.Unmarshal(message[1], &subID); err != nil {
			return err
		}
		var event Event
		if err = json.Unmarshal(message[2], &event); err != nil {
			return err
		}
		c.handleEventMessage(&EventMessage{
			SubscriptionID: subID,
			Event:          &event,
		})
		return nil
	case string(MessageTypeEOSE):
		if len(message) != 2 {
			return fmt.Errorf("invalid EOSE message length: %d", len(message))
		}
		var subID string
		if err = json.Unmarshal(message[1], &subID); err != nil {
			return err
		}
		c.handleEOSEMessage(&EOSEMessage{SubscriptionID: subID})
		return nil
	case string(MessageTypeOK):
		if len(message) != 4 {
			return fmt.Errorf("invalid OK message length: %d", len(message))
		}
		var eventID string
		if err = json.Unmarshal(message[1], &eventID); err != nil {
			return err
		}
		var ok bool
		if err = json.Unmarshal(message[2], &ok); err != nil {
			return err
		}
		var m string
		if err = json.Unmarshal(message[3], &m); err != nil {
			return err
		}
		c.handleOKMessage(&OKMessage{
			EventID: eventID,
			OK:      ok,
			Message: m,
		})
		return nil
	}

	return fmt.Errorf("unsupported message type: %s", typ)
}

func (c *Client) handleNoticeMessage(m *NoticeMessage) error {
	select {
	case c.noticeChan <- m.Message:
	default:
		// drop message
	}
	return nil
}

func (c *Client) handleEventMessage(m *EventMessage) error {
	value, ok := c.subMap.Load(m.SubscriptionID)
	if !ok {
		return fmt.Errorf("unaddressed event message: subscription id: %s", m.SubscriptionID)
	}
	group, ok := value.(*subChannelGroup)
	if !ok {
		return errors.New("invalid value in subsciption map")
	}

	select {
	case group.eventChan <- m.Event:
	default:
		// drop message
	}
	return nil
}

func (c *Client) handleEOSEMessage(m *EOSEMessage) error {
	value, ok := c.subMap.Load(m.SubscriptionID)
	if !ok {
		return fmt.Errorf("unaddressed EOSE message: subscription id: %s", m.SubscriptionID)
	}
	group, ok := value.(*subChannelGroup)
	if !ok {
		return errors.New("invalid value in subsciption map")
	}

	select {
	case group.eoseChan <- struct{}{}:
	default:
		// drop message
	}
	return nil
}

func (c *Client) handleOKMessage(m *OKMessage) error {
	value, ok := c.eventMap.Load(m.EventID)
	if !ok {
		return fmt.Errorf("unaddressed OK message: event id: %s", m.EventID)
	}
	group, ok := value.(*eventChannelGroup)
	if !ok {
		return errors.New("invalid value in event map")
	}

	select {
	case group.okChan <- &CommandResult{
		OK:      m.OK,
		Message: m.Message,
	}:
	default:
		// drop message
	}
	return nil
}
