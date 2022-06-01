import { ActionRowComponents, Content, Embed } from "../../typings/message.d.ts";

export function processEmbed(embed: Embed): Record<string, any> {
  return {
    title: embed.title,
    url: embed.url,
    author: {
      icon_url: embed.author?.iconUrl,
      name: embed.author?.name,
      url: embed.author?.url
    },
    color: embed.color,
    description: embed.description,
    fields: embed.fields,
    footer: {
      icon_url: embed.footer?.iconUrl,
      text: embed.footer?.text
    },
    image: {
      url: embed.imageUrl
    },
    thumbnail: {
      url: embed.thumbnailUrl
    },
    timestamp: embed.timestamp
  };
}

export function processComponentRow(row: ActionRowComponents[]) {
  const stack: Record<string, any>[] = [];

  for (const component of row) {
    if (component.type == "button") {
      stack.push({
        custom_id: component.id,
        disabled: component.disabled,
        emoji: component.emoji,
        label: component.text,
        type: 2,
        style: (() => {
          if (component.redirectToUrl) return 5;

          switch (component.color) {
            case "blurple":
              return 1;
            case "grey":
              return 2;
            case "green":
              return 3;
            case "red":
              return 4;
          }
        })(),
        url: component.redirectToUrl
      });
    } else if (component.type == "select menu") {
      stack.push({
        custom_id: component.id,
        disabled: component.disabled,
        max_values: component.maxValues,
        min_values: component.minValues,
        options: component.options,
        placeholder: component.placeholder,
        type: 3
      });
    }
  }

  return { type: 1, components: stack };
}

export function processContent(content: Content): Record<string, any> {
  if (typeof content == "string") return { content: content };

  return {
    allowed_mentions: content.allowedMentions,
    content: content.content,
    embeds: content.embeds?.map((embed) => processEmbed(embed)),
    flags: content.flags,
    message_reference: {
      channel_id: content.messageReference?.channelId,
      guild_id: content.messageReference?.guildId,
      message_id: content.messageReference?.messageId
    },
    tts: content.tts,
    components: content.components?.map((row) => processComponentRow(row)),
    sticker_ids: content.stickerIds
  };
}
