import { ClientOptions, Client } from "./typings/client.d.ts";
import { Command } from "./typings/command.d.ts";
import { createCommandHandler } from "./commandHandler.ts";
import { hexToUint8Array } from "./util.ts";
import { sign } from "../deps.ts";
import { processCommandsToDiscordStandard } from "./blueprints/outgoing/command.ts";

export function createClient<T extends Command>(options: ClientOptions): Client {
  const commandHandler = createCommandHandler<T>();
  const request = options.rest.request;
  const encoder = new TextEncoder();
  const key = hexToUint8Array(options.publicKey)!;

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
        case 1:
          respondWith(new Response('{"type": 1}', { headers: { "Content-Type": "application/json;" } }));
          break;
        default:
          console.log(payload);
      }
    }
  }

  return {
    applicationId: options.applicationId,
    request,
    async getLatency() {
      const startedAt = new Date().valueOf();
      await options.rest.request("GET", "/gateway", undefined, true); // This endpoint has no rate limits.
      return new Date().valueOf() - startedAt;
    },
    async syncCommands(extra) {
      const commands = extra && Array.isArray(extra.whitelist) ? commandHandler.filter((cmd) => extra.whitelist!.includes(cmd.name)) : commandHandler.getCached();

      try {
        if (extra?.guildId) await options.rest.request("PUT", `/applications/${options.applicationId}/guilds/${extra.guildId}/commands`, processCommandsToDiscordStandard(commands), true);
        else await options.rest.request("PUT", `/applications/${options.applicationId}/commands`, processCommandsToDiscordStandard(commands), true);
      } catch {
        throw new Error("Failed to bulk update discord cache. Your app probably reached limit of 100 global command updates per day, try again later.");
      }
    },
    async listen(port, encryption) {
      const socket = encryption ? Deno.listenTls({ hostname: "0.0.0.0", transport: "tcp", port, ...encryption }) : Deno.listen({ hostname: "0.0.0.0", transport: "tcp", port });
      for await (const connection of socket) handleConnection(connection);
    }
  };
}
