import { Command } from "../../typings/command.d.ts";

/**
 * It's kinda like desugarizing process... - take out friendly struct and turn into discord like payload.
 * Below code is one of the worst parts of this codebase but it's ok since commands are rarely refreshed.
 */
export function processCommand(cmd: Command, isSubcommand?: boolean): Record<string, any> {
  const payload: any = {
    name: cmd.name,
    description: cmd.description,
    default_permission: true,
    options: []
  };

  if (isSubcommand) payload.type = 1; // Type = SUB_COMMAND

  if (Array.isArray(cmd.options)) {
    for (let i = 0; i < cmd.options.length; i++) {
      payload.options.push({
        name: cmd.options![i].name,
        description: cmd.options![i].description,
        required: cmd.options![i].required,
        type: (() => {
          switch (cmd.options![i].type) {
            case "string":
              return 3;
            case "int":
              return 4;
            case "boolean":
              return 5;
            case "user":
              return 6;
            case "channel":
              return 7;
            case "role":
              return 8;
            case "float":
              return 10;
          }
        })(),
        channel_types: cmd.options![i].channelTypes?.map((c) => {
          switch (c) {
            case "normal":
              return 0;
            case "category":
              return 4;
            case "crosspost":
              return 5;
          }
        }),
        choices: cmd.options![i].choices,
        min_value: cmd.options![i].minValue,
        max_value: cmd.options![i].maxValue,
        autocomplete: cmd.options![i].autoComplete
      });
    }
  }

  return payload;
}

export function processCommandsToDiscordStandard<T extends Command>(commands: ReadonlyArray<T & { subcommands: Record<string, T> }>) {
  const payloads: any[] = [];

  for (const command of commands) {
    if (Object.keys(command).length == 0 && !command.execute) throw new TypeError(`SlashCommand@${command.name} needs to have "execute" field when there are no sub commands attached within main command.`);

    const payload = processCommand(command);
    for (const sub in command.subcommands) payload.options.push(processCommand(command.subcommands[sub], true));
    payloads.push(payload);
  }

  return payloads;
}
