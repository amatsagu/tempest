import { ButtonInteraction, CommandInteraction } from "./interaction.d.ts";
import { CommandHandler } from "./commandHandler.d.ts";
import { Target, User } from "./target.d.ts";
import { Command } from "./command.d.ts";
import { Rest } from "./rest.d.ts";
import { Content } from "./message.d.ts";

export interface ClientOptions {
  /** The Rest Controller instance to use. */
  rest: Rest;
  /** Your app/bot's user id. */
  applicationId: bigint;
  /** Hash like key used to verify incoming payloads from Discord. */
  publicKey: string;
  /** Settings related to cooldown on commands. Leave this object undefined to disable cooldowns. */
  cooldown?: {
    /** The cooldown between command usage in milliseconds. */
    duration: number;
    /** An array of user IDs representing users that are not affected by cooldowns. */
    exclusedUserIds?: Set<bigint>;
    /** A function that returns a string or Content to return to user. */
    cooldownMessage: (target: Target, timeLeft: number) => Content;
    /** Whether message should appear only for rate limited user. */
    hidden?: boolean;
    /** Whether or not to restart a command's cooldown every time it's used. */
    restartCooldown?: boolean;
    /**
     * The maximum amount of commands to execute before sweeping cooldown cache. Setting it too low may cause extra lag and too high lead to unnecessary memory usage. It's recommended to set somewhere between 100 and 5000 based on how frequently your application is used.
     * @default 100
     * */
    maxCommandsBeforeSweep?: number;
  };
}

export interface SyncOptions {
  /**
   * When provided - client gonna update commands only for this specific server (locally).
   *
   * **Warning!** While it's good for developing new commands, remember that you may create duplicates by mistake.
   * Discord global & local (per server) commands are counted differently and you may see 2x the same commands in specific guild(s).
   * */
  guildId?: bigint;
  /**
   * Include only listed commands. Syncing process gonna first erase all commands and add **only** those whose names you added into this array.
   * In order to "reset", provide empty array, that will delete all commands cached by discord.
   */
  whitelist?: string[];
}

export interface AwaitButtonBucket {
  code: string;
  buttonIds: string[];
  filter(reactor: Target): boolean;
  resolve: Function;
  resolvesAt: number;
}

export interface Client {
  /** Your app/bot's user id. */
  applicationId: bigint;
  commands: CommandHandler<Command>;
  /** User bot object. Use this to get bot name, icon url, etc. It will be available after launching bot (after Client#listen). */
  user?: User;
  /** Measures latency (ping) by sending test payload to Discord API and waiting for return message. */
  getLatency(): Promise<number>;
  /**
   * Sync currently cached commands to discord API.
   *
   * **Warning!** Global update has limit to 100 commands daily and it can take up to an hour to see changes.
   * Local (per server) changes should be instant.
   * */
  syncCommands(options?: SyncOptions): Promise<void>;
  /** Creates promise that you can await to aknowledge moment when any of listened buttons gets clicked by matching target. It will resolve with no value after timeout. */
  listenButtons(buttonIds: string[], filter: (reactor: Target) => boolean, timeout?: number): Promise<ButtonInteraction | undefined>;
  /** @returns {bigint} Id of created message. */
  sendMessage(channelId: bigint, content: Content): Promise<bigint>;
  editMessage(channelId: bigint, messageId: bigint, content: Content): Promise<void>;
  deleteMessage(channelId: bigint, messageId: bigint): Promise<void>;
  /** Publishes message if it was sent on Announcement Channel. */
  crosspostMessage(channelId: bigint, messageId: bigint): Promise<void>;
  /** Triggers specified function on each slash command. */
  onCommand?: <T extends Command>(ctx: CommandInteraction, command: T & { execute: (ctx: CommandInteraction, client: Client) => Promise<any> | any }) => Promise<any> | any;
  /** Emits all unused component payloads. */
  onInteractionComponent?: (ctx: ButtonInteraction) => Promise<any> | any;
  /** Starts application to listen incoming requests on selected port. */
  launch(port: number): Promise<void>;
}
