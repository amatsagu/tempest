package tempest

import (
	"encoding/json"
	"time"
)

// https://discord.com/developers/docs/events/webhook-events#webhook-types
type WebhookType uint8

const (
	PING_WEBHOOK_TYPE WebhookType = iota
	EVENT_WEBHOOK_TYPE
)

// https://discord.com/developers/docs/events/webhook-events#event-types
type EventType string

const (
	APPLICATION_AUTHORIZED_EVENT_TYPE   EventType = "APPLICATION_AUTHORIZED"
	APPLICATION_DEAUTHORIZED_EVENT_TYPE EventType = "APPLICATION_DEAUTHORIZED"
	ENTITLEMENT_CREATE_EVENT_TYPE       EventType = "ENTITLEMENT_CREATE"
)

// https://discord.com/developers/docs/events/webhook-events
type WebhookEvent struct {
	// version is skipped (docs says it's always 1, read-only property)
	ApplicationID Snowflake        `json:"application_id"` // Your bot/app ID
	Type          WebhookType      `json:"type"`
	Event         *EventBodyObject `json:"event,omitempty"` // Only available when webhook type is = event.
}

// https://discord.com/developers/docs/events/webhook-events#event-body-object
type EventBodyObject struct {
	Type      EventType       `json:"type"`
	Timestamp *time.Time      `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// This event can be used to detect when Discord Application/Bot is added to server or as user application.
//
// https://discord.com/developers/docs/events/webhook-events#application-authorized
type ApplicationAuthorizedEvent struct {
	IntegrationType ApplicationIntegrationType `json:"integration_type,omitempty"`
	User            User                       `json:"user"`            // The user who invited (authenticated) bot/app to either server or own user account.
	Scopes          []string                   `json:"scopes"`          // https://discord.com/developers/docs/topics/oauth2#shared-resources-oauth2-scopes
	Guild           *Guild                     `json:"guild,omitempty"` // Only available if integration type is 0 (guild/server authorization).
	Client          *Client                    `json:"-"`
}

// This event can only be used to detect when Discord Application/Bot is removed from user application.
// It's confusing in docs but it doesn't seem to work with servers at this moment.
//
// https://discord.com/developers/docs/events/webhook-events#application-deauthorized
type ApplicationDeauthorizedEvent struct {
	User   User    `json:"user"`
	Client *Client `json:"-"`
}

// https://discord.com/developers/docs/events/webhook-events#entitlement-create
type EntitlementCreationEvent struct {
	Entitlement
	Client *Client `json:"-"`
}
