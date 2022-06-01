import { sign } from "../deps.ts";
import { ClientOptions, Client } from "./typings/client.d.ts";
import { hexToUint8Array } from "./util.ts";

export function createClient(options: ClientOptions): Client {
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
    ...options,
    async listen(port, encryption) {
      const socket = encryption ? Deno.listenTls({ hostname: "0.0.0.0", transport: "tcp", port, ...encryption }) : Deno.listen({ hostname: "0.0.0.0", transport: "tcp", port });
      for await (const connection of socket) handleConnection(connection);
    }
  };
}
