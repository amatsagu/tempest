import { CommandHandler } from "./typings/commandHandler.d.ts";
import { Command } from "./typings/command.d.ts";
import { validate } from "../deps.ts";
import { COMMAND_SCHEMA } from "./constants.ts";

export function createCommandHandler<T extends Command>(): CommandHandler<T> {
  const cache = new Map<string, T & { subcommands: Record<string, T> }>();

  return {
    add(command, mainCommand) {
      validate(COMMAND_SCHEMA, command as unknown as Record<string, unknown>, `${mainCommand ? "SlashSubCommand" : "SlashCommand"}@${command.name.toLowerCase()}`);
      command.name = command.name.toLowerCase();
      mainCommand = mainCommand?.toLowerCase();

      if (mainCommand && !cache.has(mainCommand)) throw new TypeError(`SlashSubCommand@${command.name} cannot be loaded because origin command (parent, main) doesn't exist (or is just not loaded).`);
      else if (!mainCommand && cache.has(command.name)) throw new TypeError(`SlashCommand@${command.name} has been invalidated because you have already a command registered with such name.`);

      if (mainCommand) {
        const cmd = cache.get(mainCommand)!;
        if (Array.isArray(command.options) && cmd.options && cmd.options.length > 0) throw new TypeError(`SlashCommand@${mainCommand} cannot accept any subcommands if it has defined "options" property. Slash command with subcommands works like a category, namespace for them.`);

        if (cmd.subcommands) {
          if (!cmd.subcommands[command.name]) cmd.subcommands[command.name] = command;
          else throw new TypeError(`SlashSubCommand@${command.name} has been invalidated because you have already a subcommand registered with such name.`);
        } else cmd.subcommands = { [command.name]: command };
      } else {
        cache.set(command.name, { ...command, subcommands: {} });
      }
    },
    delete(command) {
      cache.delete(command);
    },
    get(command) {
      return cache.get(command);
    },
    find(fn) {
      const iter = cache.values();
      for (const value of iter) {
        if (fn(value)) return value;
      }
    },
    filter(fn, limit) {
      const result: (T & { subcommands: Record<string, T> })[] = [];
      const iter = cache.values();

      for (const value of iter) {
        if (fn(value)) {
          result.push(value);
          if (limit && result.length == limit) break;
        }
      }

      return result;
    },
    getCached() {
      return [...cache.values()];
    },
    getCachedSize() {
      return cache.size;
    }
  };
}
