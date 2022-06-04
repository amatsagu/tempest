import { ClientOptions, Client } from "./typings/client.d.ts";
import { Command } from "./typings/command.d.ts";
import { createCommandHandler } from "./commandHandler.ts";
import { hexToUint8Array } from "./util.ts";
import { sign } from "../deps.ts";
import { processCommandInteraction } from "./blueprints/incoming/commandInteraction.ts";
import { processAutoCompleteInteraction } from "./blueprints/incoming/autoCompleteInteraction.ts";
import { processCommandsToDiscordStandard } from "./blueprints/outgoing/command.ts";

export function createClient<T extends Command>(options: ClientOptions): Client {
  const commandHandler = createCommandHandler<T>();
  const restRequest = options.rest.request;
  const applicationId = options.applicationId;
  const encoder = new TextEncoder();
  const key = hexToUint8Array(options.publicKey)!;

  const onCommand =
    options.onCommand ||
    async function (ctx, cmd) {
      return !!cmd.execute;
    };

  const client: Client = {
    applicationId,
    commands: commandHandler,
    async getLatency() {
      const startedAt = new Date().valueOf();
      await restRequest("GET", "/gateway", undefined, true); // This endpoint has no rate limits.
      return new Date().valueOf() - startedAt;
    },
    async syncCommands(extra) {
      const commands = extra && Array.isArray(extra.whitelist) ? commandHandler.filter((cmd) => extra.whitelist!.includes(cmd.name)) : commandHandler.getCached();

      try {
        if (extra?.guildId) await restRequest("PUT", `/applications/${options.applicationId}/guilds/${extra.guildId}/commands`, processCommandsToDiscordStandard(commands), true);
        else await restRequest("PUT", `/applications/${options.applicationId}/commands`, processCommandsToDiscordStandard(commands), true);
      } catch {
        throw new Error("Failed to bulk update discord cache. Your app probably reached limit of 100 global command updates per day, try again later.");
      }
    },
    async listen(port, encryption) {
      const socket = encryption ? Deno.listenTls({ hostname: "0.0.0.0", transport: "tcp", port, ...encryption }) : Deno.listen({ hostname: "0.0.0.0", transport: "tcp", port });
      for await (const connection of socket) handleConnection(connection);
    }
  };

  async function handleConnection(connection: Deno.Conn) {
    const http = Deno.serveHttp(connection);

    for await (const { request, respondWith } of http) {
      if (request.method != "POST") continue;

      const signature = hexToUint8Array(request.headers.get("X-Signature-Ed25519")!);
      const timestamp = request.headers.get("X-Signature-Timestamp")!;
      if (!signature || !timestamp) continue;

      const body = await request.text();
      const valid = sign.detached.verify(encoder.encode(timestamp + body), signature, key);
      if (!valid) continue;

      const payload = JSON.parse(body);
      switch (payload.type) {
        // PING
        case 1:
          respondWith(new Response('{"type": 1}', { headers: { "Content-Type": "application/json;" } }));
          break;
        // APPLICATION_COMMAND
        case 2: {
          let command: any = commandHandler.get(payload.data.name);
          if (!command) break;

          const ctx = processCommandInteraction(payload, applicationId, restRequest);
          command = ctx.subCommand ? command.subcommands[ctx.subCommand] : command;
          const allowed = onCommand(ctx, command);
          if (allowed) command.execute && command.execute(ctx, client).catch((res: Error) => console.error(res));

          break;
        }
        // AUTO_COMPLETE
        case 4: {
          const command = commandHandler.get(payload.data.name);
          if (!command) break;

          const ctx = processAutoCompleteInteraction(payload, restRequest);
          if (!ctx.value || ctx.value == "") break; // Avoid empty loops.

          if (ctx.subCommand) {
            const subcommand = command.subcommands[ctx.subCommand];
            if (subcommand) subcommand.autoComplete?.(ctx, client);
          } else command.autoComplete?.(ctx, client);

          break;
        }
        default:
          console.log(payload);
      }
    }
  }

  return client;
}
