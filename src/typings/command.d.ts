import { AutoCompleteInteraction, CommandInteraction } from "./interaction.d.ts";
import { Client } from "./client.d.ts";

export interface Command {
  name: string;
  description: string;
  options?: CommandOptionsField[];
  autoComplete?: (ctx: AutoCompleteInteraction, client: Client) => Promise<any> | any;
  /** Main command's body. The function that should be used when command has been triggered. */
  execute?: (ctx: CommandInteraction, client: Client) => Promise<any> | any;
}

export interface CommandOptionsField {
  name: string;
  description: string;
  required?: boolean;
  type: "string" | "int" | "boolean" | "user" | "channel" | "role" | "float"; // Yeet "mentionable" away cause it's silly
  /** Only available when the option type is set to **channel**. */
  channelTypes?: ("normal" | "category" | "crosspost")[]; // Yeet "DM", "Voice" & "Stage" away cause app can't use it anyway.
  /** Only available when the option type is set to either **int** or **float**. */
  minValue?: number;
  /** Only available when the option type is set to either **int** or **float** */
  maxValue?: number;
  /** Only available when the option type is set to **int**, **float** or **string** and this command has defined **autoComplete** function in SlashCommand body.  */
  autoComplete?: boolean;
  choices?: CommandChoice[];
}

export interface CommandChoice {
  name: string;
  value: string | number | boolean;
}
