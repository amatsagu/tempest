export type ActionRowComponents = Button | SelectMenu;
export type Content = string | AdvancedMessage;

export interface Embed {
  title?: string;
  url?: string;
  author?: {
    iconUrl?: string;
    name: string;
    url?: string;
  };
  color?: number;
  description?: string;
  fields?: {
    inline?: boolean;
    name: string;
    value: string;
  }[];
  footer?: {
    iconUrl?: string;
    text: string;
  };
  imageUrl?: string;
  thumbnailUrl?: string;
  timestamp?: Date | string;
}

export interface PartialEmojiObject {
  id: string | null;
  name: string;
  animated?: boolean;
}

export interface Button {
  /** Set your own, unique string that you will catch later in code. */
  id?: string;
  type: "button";
  disabled?: boolean;
  emoji?: PartialEmojiObject;
  text?: string;
  color: "blurple" | "grey" | "green" | "red";
  redirectToUrl?: string;
}

export interface SelectMenu {
  /** Set your own, unique string that you will catch later in code. */
  id: string;
  type: "select menu";
  disabled?: boolean;
  maxValues?: number;
  minValues?: number;
  options: SelectMenuOptions[];
  placeholder?: string;
}

export interface SelectMenuOptions {
  default?: boolean;
  description?: string;
  emoji?: PartialEmojiObject;
  label: string;
  value: string;
}

export interface AdvancedMessage {
  allowedMentions?: {
    parse: ("everyone" | "users" | "roles")[];
    repliedUser?: boolean;
    users?: string[];
    roles?: string[];
  };
  content?: string;
  embeds?: Embed[];
  flags?: number;
  messageReference?: {
    channelId?: string;
    guildId?: string;
    messageId?: string;
  };
  tts?: boolean;
  components?: ActionRowComponents[][];
  stickerIds?: string[];
}
