import { CommandChoice } from "./command.d.ts";
import { Content } from "./message.d.ts";
import { Target } from "./target.d.ts";

export interface CommandInteraction {
  /** Secret string based on interaction id. Use it for generating more unique button/menu ids. */
  secret: string;
  /** The id of either guild channel id or private, dm channel. */
  channelId: bigint;
  guildId?: bigint;
  /** The user/member who triggered interaction. It's a Member type if happened inside of guild channel. */
  target: Target;
  /** Used command name. */
  command: string;
  /** Used subcommand name if that's really a subcommand. */
  subCommand?: string;
  options: Record<string, string | number>;
  defer(ephemeral?: boolean): Promise<void>;
  /**
   * Acknowledges the interaction with a message.
   * Set second param to `true` to make message ephermal *(visible only for the target)*.
   *
   * **On fallback, it gonna try to create follow up message.**
   */
  sendReply(content: Content, ephemeral?: boolean): Promise<void>;
  editReply(content: Content, ephemeral?: boolean): Promise<void>;
  deleteReply(): Promise<void>;
  /**
   * Sends message without acknowledging callback. You have to defer or send reply first to acknowledge callback.
   * Set second param to `true` to make message ephermal *(visible only for the target)*.
   */
  sendFollowUp(content: Content, ephemeral?: boolean): Promise<void>;
}

export interface ButtonInteraction {
  /** The button id defined by user while creating new button. */
  id: string;
  target: Target;
  /** Silently acknowledges the interaction. If this interaction was a part of Client#awaitButton event - it will be automatically acknowledged. */
  acknowledge(): Promise<void>;
}

export interface AutoCompleteInteraction {
  command: string;
  subCommand?: string;
  option: string;
  value: string | number;
  suggest(choices: CommandChoice[]): Promise<void>;
}
