package discord

// https://docs.discord.com/developers/resources/application#application-object-application-flags
type ApplicationFlags = BitSet

const (
	ApplicationFlagAutoModerationRuleCreateBadge ApplicationFlags = 1 << 6
	ApplicationFlagGatewayPresence               ApplicationFlags = 1 << 12
	ApplicationFlagGatewayPresenceLimited        ApplicationFlags = 1 << 13
	ApplicationFlagGatewayGuildMembers           ApplicationFlags = 1 << 14
	ApplicationFlagGatewayGuildMembersLimited    ApplicationFlags = 1 << 15
	ApplicationFlagVerificationPendingGuildLimit ApplicationFlags = 1 << 16
	ApplicationFlagEmbedded                      ApplicationFlags = 1 << 17
	ApplicationFlagGatewayMessageContent         ApplicationFlags = 1 << 18
	ApplicationFlagGatewayMessageContentLimited  ApplicationFlags = 1 << 19
	ApplicationFlagApplicationCommandBadge       ApplicationFlags = 1 << 23
)

// https://docs.discord.com/developers/resources/application#application-object-application-event-webhook-status
type ApplicationEventWebhookStatus uint8

const (
	ApplicationEventWebhookDisabled          ApplicationEventWebhookStatus = 1
	ApplicationEventWebhookEnabled           ApplicationEventWebhookStatus = 2
	ApplicationEventWebhookDisabledByDiscord ApplicationEventWebhookStatus = 3
)

// https://docs.discord.com/developers/resources/application#install-params-object
type InstallParams struct {
	Scopes      []string    `json:"scopes"`
	Permissions Permissions `json:"permissions,string,omitzero"`
}

// https://docs.discord.com/developers/resources/application#application-object-application-integration-type-configuration-object
type ApplicationIntegrationTypeConfiguration struct {
	OAuth2InstallParams *InstallParams `json:"oauth2_install_params,omitzero"`
}

// https://docs.discord.com/developers/topics/teams#data-models-membership-state-enum
type MembershipState uint8

const (
	MembershipInvited  MembershipState = 1
	MembershipAccepted MembershipState = 2
)

// https://docs.discord.com/developers/topics/teams#team-member-roles-team-member-role-types
type TeamMemberRole string

const (
	TeamMemberRoleAdmin     TeamMemberRole = "admin"
	TeamMemberRoleDeveloper TeamMemberRole = "developer"
	TeamMemberRoleReadOnly  TeamMemberRole = "read_only"
)

// https://docs.discord.com/developers/topics/teams#data-models-team-member-object
type TeamMember struct {
	Role            TeamMemberRole  `json:"role"`
	User            User            `json:"user"`
	TeamID          Snowflake       `json:"team_id"`
	MembershipState MembershipState `json:"membership_state"`
}

// https://docs.discord.com/developers/topics/teams#data-models-team-object
type Team struct {
	Icon        string       `json:"icon,omitzero"`
	Name        string       `json:"name"`
	Members     []TeamMember `json:"members"`
	ID          Snowflake    `json:"id"`
	OwnerUserID UserID       `json:"owner_user_id"`
}

// Partial guild object is intentionally skipped (see its own resource if needed).
//
// https://docs.discord.com/developers/resources/application#application-object
type Application struct {
	IntegrationTypesConfig            map[ApplicationIntegrationType]ApplicationIntegrationTypeConfiguration `json:"integration_types_config,omitzero"`
	Bot                               *User                                                                  `json:"bot,omitzero"`
	Owner                             *User                                                                  `json:"owner,omitzero"`
	Team                              *Team                                                                  `json:"team,omitzero"`
	InstallParams                     *InstallParams                                                         `json:"install_params,omitzero"`
	CustomInstallURL                  string                                                                 `json:"custom_install_url,omitzero"`
	Slug                              string                                                                 `json:"slug,omitzero"`
	EventWebhooksURL                  string                                                                 `json:"event_webhooks_url,omitzero"`
	RoleConnectionsVerificationURL    string                                                                 `json:"role_connections_verification_url,omitzero"`
	InteractionsEndpointURL           string                                                                 `json:"interactions_endpoint_url,omitzero"`
	FlagsNew                          string                                                                 `json:"flags_new,omitzero"`
	CoverImage                        string                                                                 `json:"cover_image,omitzero"`
	VerifyKey                         string                                                                 `json:"verify_key"`
	Name                              string                                                                 `json:"name"`
	Icon                              string                                                                 `json:"icon,omitzero"`
	Description                       string                                                                 `json:"description,omitzero"`
	TermsOfServiceURL                 string                                                                 `json:"terms_of_service_url,omitzero"`
	PrivacyPolicyURL                  string                                                                 `json:"privacy_policy_url,omitzero"`
	Tags                              []string                                                               `json:"tags,omitzero"`
	RPCOrigins                        []string                                                               `json:"rpc_origins,omitzero"`
	EventWebhooksTypes                []string                                                               `json:"event_webhooks_types,omitzero"`
	RedirectURIs                      []string                                                               `json:"redirect_uris,omitzero"`
	ID                                AppID                                                                  `json:"id"`
	PrimarySkuID                      SkuID                                                                  `json:"primary_sku_id,omitzero"`
	GuildID                           GuildID                                                                `json:"guild_id,omitzero"`
	ApproximateGuildCount             int                                                                    `json:"approximate_guild_count,omitzero"`
	ApproximateUserInstallCount       int                                                                    `json:"approximate_user_install_count,omitzero"`
	ApproximateUserAuthorizationCount int                                                                    `json:"approximate_user_authorization_count,omitzero"`
	Flags                             ApplicationFlags                                                       `json:"flags,omitzero"`
	EventWebhooksStatus               ApplicationEventWebhookStatus                                          `json:"event_webhooks_status,omitzero"`
	BotPublic                         bool                                                                   `json:"bot_public"`
	BotRequireCodeGrant               bool                                                                   `json:"bot_require_code_grant"`
}
