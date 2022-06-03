export function hexToUint8Array(hex: string) {
  if (!hex || hex.length % 2 != 0) return;
  const res: number[] = [];
  const size = hex.length;
  for (let i = 0; i < size; i = i + 2) res.push(parseInt(hex.slice(i, i + 2), 16));
  return new Uint8Array(res);
}

/*
    STORE IDS AS BIGINTS
    - Idea/Trick from: https://github.com/discordeno/discordeno
*/

export function hashToBigInt(hash: string) {
  if (hash[0] == "a" && hash[1] == "_") hash = `a${hash.substring(2)}`;
  else hash = `b${hash}`;
  return BigInt(`0x${hash}`);
}

export function bigIntToHash(hashedValue: bigint) {
  if (!hashedValue) return;
  const hash = hashedValue.toString(16);
  return hash[0] == "a" ? `a_${hash.substring(1)}` : hash.substring(1);
}
