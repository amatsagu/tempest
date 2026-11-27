package discord

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
)

func (msg *Message) UnmarshalJSON(data []byte) error {
	return unmarshalWithDecoder(data, msg.UnmarshalJSONFrom)
}

func (msg *Message) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	isNull, err := consumeJSONObjectStart(dec)
	if err != nil || isNull {
		if isNull {
			*msg = Message{}
		}
		return err
	}

	*msg = Message{}
	for dec.PeekKind() != '}' {
		key, err := readJSONKeyBytes(dec)
		if err != nil {
			return err
		}

		switch string(key) {
		case "message_reference":
			err = json.UnmarshalDecode(dec, &msg.MessageReference)
		case "interaction_metadata":
			err = json.UnmarshalDecode(dec, &msg.InteractionMetadata)
		case "poll":
			err = json.UnmarshalDecode(dec, &msg.Poll)
		case "member":
			err = json.UnmarshalDecode(dec, &msg.Member)
		case "resolved":
			err = json.UnmarshalDecode(dec, &msg.Resolved)
		case "author":
			err = json.UnmarshalDecode(dec, &msg.Author)
		case "timestamp":
			err = json.UnmarshalDecode(dec, &msg.Timestamp)
		case "edited_timestamp":
			err = json.UnmarshalDecode(dec, &msg.EditedTimestamp)
		case "referenced_message":
			err = json.UnmarshalDecode(dec, &msg.ReferencedMessage)
		case "content":
			err = json.UnmarshalDecode(dec, &msg.Content)
		case "nonce":
			err = msg.Nonce.UnmarshalJSONFrom(dec)
		case "attachments":
			err = json.UnmarshalDecode(dec, &msg.Attachments)
		case "mentions":
			err = json.UnmarshalDecode(dec, &msg.Mentions)
		case "mention_roles":
			err = json.UnmarshalDecode(dec, &msg.MentionRoles)
		case "mention_channels":
			err = json.UnmarshalDecode(dec, &msg.MentionChannels)
		case "embeds":
			err = json.UnmarshalDecode(dec, &msg.Embeds)
		case "sticker_items":
			err = json.UnmarshalDecode(dec, &msg.StickerItems)
		case "reactions":
			err = json.UnmarshalDecode(dec, &msg.Reactions)
		case "components":
			var list componentSlice[MessageComponent]
			if err = list.UnmarshalJSONFrom(dec); err != nil {
				return err
			}
			msg.Components = []MessageComponent(list)
		case "webhook_id":
			err = msg.WebhookID.UnmarshalJSONFrom(dec)
		case "application_id":
			err = msg.ApplicationID.UnmarshalJSONFrom(dec)
		case "guild_id":
			err = msg.GuildID.UnmarshalJSONFrom(dec)
		case "flags":
			msg.Flags, err = readJSONBitSet(dec)
		case "channel_id":
			err = msg.ChannelID.UnmarshalJSONFrom(dec)
		case "id":
			err = msg.ID.UnmarshalJSONFrom(dec)
		case "position":
			msg.Position, err = readJSONUint32(dec)
		case "type":
			var msgType uint8
			msgType, err = readJSONUint8(dec)
			msg.Type = MessageType(msgType)
		case "mention_everyone":
			msg.MentionEveryone, err = readJSONBool(dec)
		case "tts":
			msg.TTS, err = readJSONBool(dec)
		case "pinned":
			msg.Pinned, err = readJSONBool(dec)
		default:
			err = dec.SkipValue()
		}
		if err != nil {
			return err
		}
	}

	_, err = dec.ReadToken()
	return err
}
