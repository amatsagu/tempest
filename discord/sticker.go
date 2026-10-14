package discord

// https://docs.discord.com/developers/resources/sticker#sticker-object-sticker-types
type StickerType uint8

const (
	StickerStandard StickerType = 1
	StickerGuild    StickerType = 2
)

// https://docs.discord.com/developers/resources/sticker#sticker-object-sticker-format-types
type StickerFormatType uint8

const (
	StickerFormatPNG    StickerFormatType = 1
	StickerFormatAPNG   StickerFormatType = 2
	StickerFormatLottie StickerFormatType = 3
	StickerFormatGIF    StickerFormatType = 4
)

// https://docs.discord.com/developers/resources/sticker#sticker-object-sticker-structure
type Sticker struct {
	User        *User             `json:"user,omitzero"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitzero"`
	Tags        string            `json:"tags"`
	PackID      Snowflake         `json:"pack_id,omitzero"`
	GuildID     GuildID           `json:"guild_id,omitzero"`
	ID          Snowflake         `json:"id"`
	SortValue   int               `json:"sort_value,omitzero"`
	Type        StickerType       `json:"type"`
	FormatType  StickerFormatType `json:"format_type"`
	Available   bool              `json:"available,omitzero"`
}

// https://docs.discord.com/developers/resources/sticker#sticker-item-object-sticker-item-structure
type StickerItem struct {
	Name       string            `json:"name"`
	ID         Snowflake         `json:"id"`
	FormatType StickerFormatType `json:"format_type"`
}

// https://docs.discord.com/developers/resources/sticker#sticker-pack-object-sticker-pack-structure
type StickerPack struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Stickers       []Sticker `json:"stickers"`
	ID             Snowflake `json:"id"`
	SkuID          SkuID     `json:"sku_id"`
	CoverStickerID Snowflake `json:"cover_sticker_id,omitzero"`
	BannerAssetID  Snowflake `json:"banner_asset_id,omitzero"`
}

// https://docs.discord.com/developers/resources/sticker#list-sticker-packs-response-structure
type StickerPacksResponse struct {
	StickerPacks []StickerPack `json:"sticker_packs"`
}
