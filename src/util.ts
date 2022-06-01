export function hexToUint8Array(hex: string) {
  if (!hex || hex.length % 2 != 0) return;
  const res: number[] = [];
  const size = hex.length;
  for (let i = 0; i < size; i = i + 2) res.push(parseInt(hex.slice(i, i + 2), 16));
  return new Uint8Array(res);
}
