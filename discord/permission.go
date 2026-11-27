package discord

// https://docs.discord.com/developers/topics/permissions#permissions-bitwise-permission-flags
type PermissionFlags = BitSet

const (
	CreateInstantInvitePermissionFlag PermissionFlags = 1 << iota
	KickMembersPermissionFlag
	BanMembersPermissionFlag
	AdministratorPermissionFlag
	ManageChannelsPermissionFlag
	ManageGuildPermissionFlag
	AddReactionsPermissionFlag
	ViewAuditLogPermissionFlag
	PrioritySpeakerPermissionFlag
	StreamPermissionFlag
	ViewChannelPermissionFlag
	SendMessagesPermissionFlag
	SendTTSMessagesPermissionFlag
	ManageMessagesPermissionFlag
	EmbedLinksPermissionFlag
	AttachFilesPermissionFlag
	ReadMessageHistoryPermissionFlag
	MentionEveryonePermissionFlag
	UseExternalEmojisPermissionFlag
	ViewGuildInsightsPermissionFlag
	ConnectPermissionFlag
	SpeakPermissionFlag
	MuteMembersPermissionFlag
	DeafenMembersPermissionFlag
	MoveMembersPermissionFlag
	UseVADPermissionFlag
	ChangeNicknamePermissionFlag
	ManageNicknamesPermissionFlag
	ManageRolesPermissionFlag
	ManageWebhooksPermissionFlag
	ManageGuildExpressionsPermissionFlag
	UseApplicationCommandsPermissionFlag
	RequestToSpeakPermissionFlag
	ManageEventsPermissionFlag
	ManageThreadsPermissionFlag
	CreatePublicThreadsPermissionFlag
	CreatePrivateThreadsPermissionFlag
	UseExternalStickersPermissionFlag
	SendMessagesInThreadsPermissionFlag
	UseEmbeddedActivitiesPermissionFlag
	ModerateMembersPermissionFlag
	ViewCreatorMonetizationAnalyticsPermissionFlag
	UseSoundboardPermissionFlag
	CreateGuildExpressionsPermissionFlag
	CreateEventsPermissionFlag
	UseExternalSoundsPermissionFlag
	SendVoiceMessagesPermissionFlag
	_
	SetVoiceChannelStatusPermissionFlag
	SendPollsPermissionFlag
	UseExternalAppsPermissionFlag
	PinMessagesPermissionFlag
	BypassSlowmodePermissionFlag

	AllTextPermissionFlags = ViewChannelPermissionFlag |
		SendMessagesPermissionFlag |
		SendTTSMessagesPermissionFlag |
		ManageMessagesPermissionFlag |
		EmbedLinksPermissionFlag |
		AttachFilesPermissionFlag |
		ReadMessageHistoryPermissionFlag |
		MentionEveryonePermissionFlag |
		SendVoiceMessagesPermissionFlag |
		SendPollsPermissionFlag |
		UseExternalAppsPermissionFlag |
		PinMessagesPermissionFlag |
		BypassSlowmodePermissionFlag

	AllThreadPermissionFlags = ManageThreadsPermissionFlag |
		CreatePublicThreadsPermissionFlag |
		CreatePrivateThreadsPermissionFlag |
		SendMessagesInThreadsPermissionFlag

	AllVoicePermissionFlags = ViewChannelPermissionFlag |
		ConnectPermissionFlag |
		SpeakPermissionFlag |
		StreamPermissionFlag |
		MuteMembersPermissionFlag |
		DeafenMembersPermissionFlag |
		MoveMembersPermissionFlag |
		UseVADPermissionFlag |
		PrioritySpeakerPermissionFlag |
		UseSoundboardPermissionFlag |
		UseExternalSoundsPermissionFlag |
		RequestToSpeakPermissionFlag |
		UseEmbeddedActivitiesPermissionFlag |
		SetVoiceChannelStatusPermissionFlag |
		CreateGuildExpressionsPermissionFlag |
		CreateEventsPermissionFlag |
		ManageEventsPermissionFlag

	AllChannelPermissionFlags = AllTextPermissionFlags |
		AllThreadPermissionFlags |
		AllVoicePermissionFlags |
		CreateInstantInvitePermissionFlag |
		ManageChannelsPermissionFlag |
		AddReactionsPermissionFlag |
		UseExternalEmojisPermissionFlag |
		UseApplicationCommandsPermissionFlag |
		UseExternalStickersPermissionFlag

	AllPermissionFlags = AllChannelPermissionFlags |
		KickMembersPermissionFlag |
		BanMembersPermissionFlag |
		ManageGuildPermissionFlag |
		AdministratorPermissionFlag |
		ManageWebhooksPermissionFlag |
		ManageGuildExpressionsPermissionFlag |
		ViewCreatorMonetizationAnalyticsPermissionFlag |
		ViewGuildInsightsPermissionFlag |
		ViewAuditLogPermissionFlag |
		ManageRolesPermissionFlag |
		ChangeNicknamePermissionFlag |
		ManageNicknamesPermissionFlag |
		ModerateMembersPermissionFlag
)
