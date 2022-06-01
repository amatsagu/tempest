export interface SocketTls {
  /** Path to a file with cert chain in PEM format. */
  readonly cert: string;
  /** Path to a file with private key in PEM format. */
  readonly key: string;
}
