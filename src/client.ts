import { ClientOptions, Client } from "./typings/client.d.ts";
import { Rest } from "./typings/rest.d.ts";

export function createClient(options: ClientOptions): Client {
  return {
    ...options,
    async listen() {}
  };
}
