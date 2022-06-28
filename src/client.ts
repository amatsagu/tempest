import { ClientOptions, Client, AwaitButtonBucket } from "./typings/client.d.ts";
import { Command } from "./typings/command.d.ts";
import { createCommandHandler } from "./commandHandler.ts";
import { hexToUint8Array } from "./util.ts";
import { sign } from "../deps.ts";
import { processCommandInteraction } from "./blueprints/incoming/commandInteraction.ts";
import { processAutoCompleteInteraction } from "./blueprints/incoming/autoCompleteInteraction.ts";
import { processCommandsToDiscordStandard } from "./blueprints/outgoing/command.ts";
import { processUser } from "./blueprints/incoming/user.ts";
import { processButtonInteraction } from "./blueprints/incoming/buttonInteraction.ts";
import { processContent } from "./blueprints/outgoing/message.ts";

export function createClient<T extends Command>(options: ClientOptions): Client {
  const commandHandler = createCommandHandler<T>();
  const restRequest = options.rest.request;
  const applicationId = options.applicationId;
  const buttonCollectors = new Map<string, AwaitButtonBucket>();
  const cdr = options.cooldown;
  const cooldowns = (cdr && new Map<bigint, number>()) as Map<bigint, number>;
  let running = false;
  let cdrTrg = 0; // Trigger cooldowns cache to be bleaned every XXX commands.

  const client: Client = {
    applicationId,
    commands: commandHandler,
    async getLatency() {
      const startedAt = new Date().valueOf();
      await restRequest("GET", "/gateway", undefined, true); // This endpoint has no rate limits.
      return new Date().valueOf() - startedAt;
    },
    async listenButtons(buttonIds, filter, timeout = 60000) {
      return new Promise((resolve) => {
        const code = buttonIds.join("-");
        buttonCollectors.set(code, { code, buttonIds, filter, resolve, resolvesAt: Date.now() + timeout });
        setTimeout(resolve, timeout);
      });
    },
    async sendMessage(channelId, content) {
      const res = await restRequest("POST", `/channels/${channelId}/messages`, processContent(content));
      return BigInt(res.id as string);
    },
    async editMessage(channelId, messageId, content) {
      await restRequest("PATCH", `/channels/${channelId}/messages/${messageId}`, processContent(content), true);
    },
    async deleteMessage(channelId, messageId) {
      await restRequest("DELETE", `/channels/${channelId}/messages/${messageId}`, undefined, true);
    },
    async crosspostMessage(channelId, messageId) {
      await restRequest("POST", `/channels/${channelId}/messages/${messageId}/crosspost`, undefined, true);
    },
    async syncCommands(extra) {
      const commands = extra && Array.isArray(extra.whitelist) ? commandHandler.filter((cmd) => extra.whitelist!.includes(cmd.name)) : commandHandler.getCached();

      try {
        if (extra?.guildId) await restRequest("PUT", `/applications/${applicationId}/guilds/${extra.guildId}/commands`, processCommandsToDiscordStandard(commands), true);
        else await restRequest("PUT", `/applications/${applicationId}/commands`, processCommandsToDiscordStandard(commands), true);
      } catch {
        throw new Error("Failed to bulk update discord cache. Your app probably reached limit of 100 global command updates per day, try again later.");
      }
    },
    async onCommand(ctx, _command) {
      await ctx.sendReply("Rejected action because your Client#onCommand handler is still not configured!", true);
      throw new Error(`Received ${ctx.command}${ctx.subCommand ? `@${ctx.subCommand} subcommand` : " command"} but it got rejected because your Client#onCommand handler is not yet configured!`);
    },
    async launch(port) {
      if (running) throw new Error("Client's web server is already running!");
      running = true;
      this.user = processUser(await restRequest("GET", `/users/${applicationId}`));
      initSocket(port);
    }
  };

  async function initSocket(port: number) {
    const socket = Deno.listen({ hostname: "0.0.0.0", port, transport: "tcp" });
    const key = hexToUint8Array(options.publicKey)!;
    const encoder = new TextEncoder();

    while (true) {
      try {
        const http = Deno.serveHttp(await socket.accept());
        const event = await http.nextRequest();
        if (!event || event.request.method != "POST") continue;

        const req = event.request;
        const signature = hexToUint8Array(req.headers.get("X-Signature-Ed25519")!);
        const timestamp = req.headers.get("X-Signature-Timestamp")!;
        if (!signature || !timestamp) continue;

        const body = await req.text();
        const valid = sign.detached.verify(encoder.encode(timestamp + body), signature, key);
        if (!valid) continue;

        const payload = JSON.parse(body);
        switch (payload.type) {
          // PING
          case 1:
            event.respondWith(new Response('{"type": 1}', { headers: { "Content-Type": "application/json;" } }));
            break;
          // APPLICATION_COMMAND
          case 2: {
            let command: any = commandHandler.get(payload.data.name);
            if (!command) break;

            const ctx = processCommandInteraction(payload, applicationId, restRequest);
            command = ctx.subCommand ? command.subcommands[ctx.subCommand] : command;

            if (cdr && !cdr.exclusedUserIds?.has(ctx.target.id)) {
              const now = Date.now();
              let resv = cooldowns.get(ctx.target.id) ?? 0;
              const timeLeft = resv - now;

              if (timeLeft > 0) {
                if (cdr.restartCooldown) cooldowns.set(ctx.target.id, now + cdr.duration);
                ctx.sendReply(cdr.cooldownMessage(ctx.target, cdr.restartCooldown ? cdr.duration : timeLeft), cdr.hidden);
                break;
              } else {
                cdrTrg++;
                if (cdrTrg % cdr.maxCommandsBeforeSweep == 0) cooldowns.clear(); // To simplify - just clear all no matter whether it was expired or not.
                cooldowns.set(ctx.target.id, now + cdr.duration);
              }
            }

            command.execute && client.onCommand && client.onCommand(ctx, command);
            break;
          }
          // MESSAGE_COMPONENT (BUTTON, SELECT MENU, ETC.)
          case 3: {
            // BUTTON
            if (payload.data.component_type == 2 && payload.data.custom_id) {
              const iter = buttonCollectors.values();
              let collector!: AwaitButtonBucket;

              for (const v of iter) {
                if (v.buttonIds.includes(payload.data.custom_id)) collector = v;
              }

              const button = processButtonInteraction(payload, restRequest);
              if (!collector) break;

              if (collector.filter(button.target)) {
                buttonCollectors.delete(collector.code);
                button.acknowledge();
                collector.resolve(button);
              }
            }

            if (client.onInteractionComponent) client.onInteractionComponent(payload);
            break;
          }
          // AUTO_COMPLETE
          case 4: {
            let command = commandHandler.get(payload.data.name);
            if (!command) break;

            const ctx = processAutoCompleteInteraction(payload, restRequest);
            if (!ctx.value || ctx.value == "") break; // Avoid empty loops.

            if (ctx.subCommand) {
              const subcommand = command.subcommands[ctx.subCommand];
              if (subcommand) subcommand.autoComplete?.(ctx, client);
            } else command.autoComplete?.(ctx, client);

            break;
          }
          default: {
            console.log(payload);
          }
        }
      } catch (err) {
        const type = err.constructor;
        // @ts-ignore Unstable Deno core...
        if (type !== Deno.core.Interrupted && type !== Deno.core.BadResource) throw err;
      }
    }
  }

  return client;
}
