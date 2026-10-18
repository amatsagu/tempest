package tempest

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/amatsagu/tempest/discord"
)

// SKUs returns the application's premium offerings. Discord returns a
// SUBSCRIPTION_GROUP SKU next to each SUBSCRIPTION one; use the subscription.
// https://docs.discord.com/developers/resources/sku#list-skus
func (c *Client) SKUs(ctx context.Context) ([]discord.SKU, error) {
	path, err := c.appPath("/skus")
	if err != nil {
		return nil, err
	}

	var skus []discord.SKU
	if err := c.rest.RequestInto(ctx, http.MethodGet, path, nil, "", &skus); err != nil {
		return nil, err
	}
	return skus, nil
}

// Entitlements returns the application's entitlements, active and expired. A nil
// filter asks for all of them with Discord's defaults.
// https://docs.discord.com/developers/resources/entitlement#list-entitlements
func (c *Client) Entitlements(ctx context.Context, filter *discord.EntitlementFilter) ([]discord.Entitlement, error) {
	path, err := c.appPath("/entitlements")
	if err != nil {
		return nil, err
	}

	var entitlements []discord.Entitlement
	if err := c.rest.RequestInto(ctx, http.MethodGet, path+entitlementQuery(filter), nil, "", &entitlements); err != nil {
		return nil, err
	}
	return entitlements, nil
}

// Entitlement returns one entitlement.
// https://docs.discord.com/developers/resources/entitlement#get-entitlement
func (c *Client) Entitlement(ctx context.Context, id discord.EntitlementID) (discord.Entitlement, error) {
	var entitlement discord.Entitlement

	path, err := c.appPath("/entitlements/" + id.String())
	if err != nil {
		return entitlement, err
	}

	err = c.rest.RequestInto(ctx, http.MethodGet, path, nil, "", &entitlement)
	return entitlement, err
}

// ConsumeEntitlement marks a one-time purchase as used up, which is what a
// consumable SKU needs after the item is delivered.
// https://docs.discord.com/developers/resources/entitlement#consume-an-entitlement
func (c *Client) ConsumeEntitlement(ctx context.Context, id discord.EntitlementID) error {
	path, err := c.appPath("/entitlements/" + id.String() + "/consume")
	if err != nil {
		return err
	}

	_, err = c.rest.Request(ctx, http.MethodPost, path, nil, "")
	return err
}

// CreateTestEntitlement grants a SKU to a user or guild for testing, without a
// purchase. Discord returns a partial entitlement; reload the client to see it.
// https://docs.discord.com/developers/resources/entitlement#create-test-entitlement
func (c *Client) CreateTestEntitlement(ctx context.Context, payload discord.TestEntitlementPayload) (discord.Entitlement, error) {
	var entitlement discord.Entitlement

	path, err := c.appPath("/entitlements")
	if err != nil {
		return entitlement, err
	}

	err = c.rest.RequestInto(ctx, http.MethodPost, path, payload, "", &entitlement)
	return entitlement, err
}

// DeleteTestEntitlement removes a test entitlement, so the user or guild loses
// access again.
// https://docs.discord.com/developers/resources/entitlement#delete-test-entitlement
func (c *Client) DeleteTestEntitlement(ctx context.Context, id discord.EntitlementID) error {
	path, err := c.appPath("/entitlements/" + id.String())
	if err != nil {
		return err
	}

	_, err = c.rest.Request(ctx, http.MethodDelete, path, nil, "")
	return err
}

// Subscriptions returns the subscriptions that contain a SKU, filtered by user.
// UserID is required by Discord unless the token is an OAuth2 token.
// https://docs.discord.com/developers/resources/subscription#list-sku-subscriptions
func (c *Client) Subscriptions(ctx context.Context, skuID discord.SkuID, filter *discord.SubscriptionFilter) ([]discord.Subscription, error) {
	if c.rest == nil {
		return nil, errors.New("tempest: Config.Token is required to call the Discord API")
	}

	path := "/skus/" + skuID.String() + "/subscriptions" + subscriptionQuery(filter)

	var subscriptions []discord.Subscription
	if err := c.rest.RequestInto(ctx, http.MethodGet, path, nil, "", &subscriptions); err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// Subscription returns one subscription by its ID.
// https://docs.discord.com/developers/resources/subscription#get-sku-subscription
func (c *Client) Subscription(ctx context.Context, skuID discord.SkuID, id discord.SubscriptionID) (discord.Subscription, error) {
	var subscription discord.Subscription

	if c.rest == nil {
		return subscription, errors.New("tempest: Config.Token is required to call the Discord API")
	}

	path := "/skus/" + skuID.String() + "/subscriptions/" + id.String()
	err := c.rest.RequestInto(ctx, http.MethodGet, path, nil, "", &subscription)
	return subscription, err
}

// appPath builds an application-scoped REST path, reporting the two things that
// make one impossible.
func (c *Client) appPath(suffix string) (string, error) {
	if c.rest == nil {
		return "", errors.New("tempest: Config.Token is required to call the Discord API")
	}

	if c.appID.Snowflake == 0 {
		return "", errors.New("tempest: application ID is unknown; set Config.ApplicationID")
	}
	return "/applications/" + c.appID.String() + suffix, nil
}

// entitlementQuery encodes a filter, omitting everything unset.
func entitlementQuery(filter *discord.EntitlementFilter) string {
	if filter == nil {
		return ""
	}

	query := url.Values{}
	if filter.UserID.Snowflake != 0 {
		query.Set("user_id", filter.UserID.String())
	}

	if filter.GuildID.Snowflake != 0 {
		query.Set("guild_id", filter.GuildID.String())
	}

	if filter.Before.Snowflake != 0 {
		query.Set("before", filter.Before.String())
	}

	if filter.After.Snowflake != 0 {
		query.Set("after", filter.After.String())
	}

	if len(filter.SkuIDs) > 0 {
		query.Set("sku_ids", joinIDs(filter.SkuIDs))
	}

	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}

	if filter.ExcludeEnded {
		query.Set("exclude_ended", "true")
	}

	// Discord excludes deleted entitlements, so asking for them means saying so.
	if filter.IncludeDeleted {
		query.Set("exclude_deleted", "false")
	}

	return encodeQuery(query)
}

// subscriptionQuery encodes a filter, omitting everything unset.
func subscriptionQuery(filter *discord.SubscriptionFilter) string {
	if filter == nil {
		return ""
	}

	query := url.Values{}
	if filter.UserID.Snowflake != 0 {
		query.Set("user_id", filter.UserID.String())
	}

	if filter.Before.Snowflake != 0 {
		query.Set("before", filter.Before.String())
	}

	if filter.After.Snowflake != 0 {
		query.Set("after", filter.After.String())
	}

	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}

	return encodeQuery(query)
}

func encodeQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}
	return "?" + query.Encode()
}

// joinIDs renders snowflake IDs as Discord's comma-delimited set.
func joinIDs(ids []discord.SkuID) string {
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(id.String())
	}
	return b.String()
}
