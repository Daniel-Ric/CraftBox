export function terminalInputChunks(text: string): string[] {
  const bytes = new TextEncoder().encode(text),
    decoder = new TextDecoder(),
    out: string[] = [];
  for (let start = 0; start < bytes.length;) {
    let end = Math.min(start + 32768, bytes.length);
    while (end < bytes.length && (bytes[end] & 0xc0) === 0x80) end--;
    out.push(decoder.decode(bytes.subarray(start, end)));
    start = end;
  }
  return out;
}
