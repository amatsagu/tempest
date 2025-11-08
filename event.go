package tempest

import (
	"encoding/json"
	"time"
)

// https://discord.com/developers/docs/events/webhook-events#event-types
type EventType string

const (
	APPLICATION_AUTHORIZED_EVENT_TYPE   EventType = "APPLICATION_AUTHORIZED"
	APPLICATION_DEAUTHORIZED_EVENT_TYPE EventType = "APPLICATION_DEAUTHORIZED"
	ENTITLEMENT_CREATE_EVENT_TYPE       EventType = "ENTITLEMENT_CREATE"
)

// https://discord.com/developers/docs/events/webhook-events#event-body-object
type EventObject struct {
	Type      EventType       `json:"type"`
	Timestamp *time.Time      `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}
