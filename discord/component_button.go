package discord

// https://docs.discord.com/developers/components/reference#button-button-styles
type ButtonStyle uint8

const (
	ButtonPrimary ButtonStyle = iota + 1
	ButtonSecondary
	ButtonSuccess
	ButtonDanger
	ButtonLink
	ButtonPremium
)

// https://docs.discord.com/developers/components/reference#button-button-structure
type Button struct {
	Emoji    *Emoji        `json:"emoji,omitzero"`
	Label    string        `json:"label,omitzero"`
	CustomID string        `json:"custom_id,omitzero"`
	URL      string        `json:"url,omitzero"`
	SKUID    SkuID         `json:"sku_id,omitzero"`
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"`
	Style    ButtonStyle   `json:"style"`
	Disabled bool          `json:"disabled,omitzero"`
}

func NewButton(customID, label string, style ButtonStyle) Button {
	return Button{
		Type:     ComponentButton,
		Style:    style,
		Label:    label,
		CustomID: customID,
	}
}

func NewLinkButton(url, label string) Button {
	return Button{
		Type:  ComponentButton,
		Style: ButtonLink,
		Label: label,
		URL:   url,
	}
}

func NewPremiumButton(skuID SkuID) Button {
	return Button{
		Type:  ComponentButton,
		Style: ButtonPremium,
		SKUID: skuID,
	}
}
