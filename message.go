package nostr

import (
	"encoding/json"
)

type MessageType string

const (
	MessageTypeEvent  MessageType = "EVENT"
	MessageTypeReq    MessageType = "REQ"
	MessageTypeClose  MessageType = "CLOSE"
	MessageTypeNotice MessageType = "NOTICE"
	MessageTypeEOSE   MessageType = "EOSE"
	MessageTypeOK     MessageType = "OK"
)

type EventMessage struct {
	SubscriptionID string // optional
	Event          *Event
}

func (m *EventMessage) MarshalJSON() ([]byte, error) {
	body := []any{MessageTypeEvent}
	if subID := m.SubscriptionID; subID != "" {
		body = append(body, subID)
	}
	body = append(body, m.Event)
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type ReqMessage struct {
	SubscriptionID string
	Filters        []Filter
}

func (m *ReqMessage) MarshalJSON() ([]byte, error) {
	body := []any{MessageTypeReq, m.SubscriptionID}
	for _, f := range m.Filters {
		body = append(body, f)
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type CloseMessage struct {
	SubscriptionID string
}

func (m *CloseMessage) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal([]any{"CLOSE", m.SubscriptionID})
	if err != nil {
		return nil, err
	}
	return b, nil
}

type NoticeMessage struct {
	Message string
}

type EOSEMessage struct {
	SubscriptionID string
}

type OKMessage struct {
	EventID string
	OK      bool
	Message string
}
