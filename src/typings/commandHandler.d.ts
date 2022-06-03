import { Command } from "./command.d.ts";

export interface CommandHandler<T extends Command> {
  /**
   * Adds command to the app's cache so it can be used automatically.
   * Provide main command name if you want to add subcommand.
   */
  add(command: T, mainCommand?: string): void;
  /** Deletes command & all related subcommands. */
  delete(command: string): void;
  get(command: string): Readonly<T & { subcommands: Record<string, T> }> | undefined;
  find(fn: (command: T & { subcommands: Record<string, T> }) => boolean): Readonly<T & { subcommands: Record<string, T> }> | undefined;
  filter(fn: (command: T & { subcommands: Record<string, T> }) => boolean, limit?: number): ReadonlyArray<T & { subcommands: Record<string, T> }>;
  /** Returns all registered commands from app's cache. */
  getCached(): ReadonlyArray<T & { subcommands: Record<string, T> }>;
  getCachedSize(): number;
}
