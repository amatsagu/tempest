import { Rest } from "./rest.d.ts";
import { SocketTls } from "./util.d.ts";

export interface ClientOptions {
  /** The Rest Controller instance to use. */
  rest: Rest;
  /** Your app/bot's user id. */
  applicationId: bigint;
  /** Hash like key used to verify incoming payloads from Discord. */
  publicKey: string;
}

export interface Client extends ClientOptions {
  /** Starts application to listen incoming requests on selected port. */
  listen(port: number, encryption?: SocketTls): Promise<void>;
}
