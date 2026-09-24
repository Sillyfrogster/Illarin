export type ScanPart = {
  id: string;
  name: string;
  kind: "picture" | "card" | "data" | "file";
  size: number;
  note: string;
  picture?: string;
  lines?: string[];
};

const PICTURE = /\.(png|apng|jpe?g|webp|gif)$/i;
const MAX_PARTS = 200;
const MAX_PICTURES = 60;
const MAX_PICTURE_BYTES = 12 << 20;
const MAX_DATA_BYTES = 4 << 20;

/** scanFile lists what a chosen file holds, reading it in the browser so the page can show each part as it is found. */
export async function scanFile(file: File): Promise<ScanPart[]> {
  try {
    const head = new Uint8Array(await file.slice(0, 8).arrayBuffer());
    if (head[0] === 0x50 && head[1] === 0x4b) return await archiveParts(file);
    if (head[0] === 0x89 && head[1] === 0x50) return await pngParts(file);
    if (PICTURE.test(file.name) || file.type.startsWith("image/")) {
      return [pictureOf(file.name, file, file.size, 0)];
    }
    if (file.size <= MAX_DATA_BYTES) {
      const data = dataPart(file.name, await file.text(), file.size, 0);
      if (data) return [data];
    }
  } catch {}
  return [
    { id: "0", name: file.name, kind: "file", size: file.size, note: "File" },
  ];
}

/** revokeScan frees the pictures a scan made. */
export function revokeScan(parts: ScanPart[]) {
  for (const part of parts) if (part.picture) URL.revokeObjectURL(part.picture);
}

function pictureOf(
  name: string,
  body: Blob,
  size: number,
  index: number,
): ScanPart {
  return {
    id: String(index),
    name,
    kind: "picture",
    size,
    note: "Picture",
    picture: URL.createObjectURL(body),
  };
}

async function pngParts(file: File): Promise<ScanPart[]> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  const view = new DataView(bytes.buffer);
  const parts: ScanPart[] = [];
  const picture = pictureOf(file.name, file, file.size, 0);
  picture.note = `${view.getUint32(16)} × ${view.getUint32(20)} picture`;
  parts.push(picture);
  const latin = new TextDecoder("latin1");
  for (let at = 8; at + 12 <= bytes.length; ) {
    const length = view.getUint32(at);
    const type = latin.decode(bytes.subarray(at + 4, at + 8));
    const data = bytes.subarray(at + 8, at + 8 + length);
    if (type === "tEXt") {
      const split = data.indexOf(0);
      const keyword = latin.decode(data.subarray(0, split));
      if (keyword === "chara" || keyword === "ccv3") {
        const text = decodeBase64(latin.decode(data.subarray(split + 1)));
        const card = dataPart(
          `${keyword} text chunk`,
          text,
          length,
          parts.length,
        );
        if (card) parts.push(card);
      }
    }
    if (type === "IEND") break;
    at += 12 + length;
  }
  return parts;
}

type ArchiveEntry = {
  name: string;
  method: number;
  compressed: number;
  size: number;
  local: number;
};

// ponytail: reads plain ZIP only; a ZIP64 archive lists nothing here and still uploads
async function archiveParts(file: File): Promise<ScanPart[]> {
  const entries = (await archiveEntries(file))
    .filter((entry) => !entry.name.endsWith("/"))
    .sort(
      (a, b) => Number(b.name === "card.json") - Number(a.name === "card.json"),
    )
    .slice(0, MAX_PARTS);
  const parts: ScanPart[] = [];
  let pictures = 0;
  for (const [index, entry] of entries.entries()) {
    const readable =
      (entry.method === 0 || entry.method === 8) &&
      entry.size <= MAX_PICTURE_BYTES;
    if (PICTURE.test(entry.name) && readable && pictures < MAX_PICTURES) {
      pictures += 1;
      const body = await archivedBody(file, entry);
      const part = pictureOf(entry.name, body, entry.size, index);
      const size = await pngSize(body);
      if (size) part.note = `${size[0]} × ${size[1]} picture`;
      parts.push(part);
      continue;
    }
    if (
      /\.json$/i.test(entry.name) &&
      readable &&
      entry.size <= MAX_DATA_BYTES
    ) {
      const text = await (await archivedBody(file, entry)).text();
      const data = dataPart(entry.name, text, entry.size, index);
      if (data) {
        parts.push(data);
        continue;
      }
    }
    parts.push({
      id: String(index),
      name: entry.name,
      kind: PICTURE.test(entry.name) ? "picture" : "file",
      size: entry.size,
      note: PICTURE.test(entry.name) ? "Picture" : "File",
    });
  }
  return parts;
}

async function archiveEntries(file: File): Promise<ArchiveEntry[]> {
  const tailLength = Math.min(file.size, 65557);
  const tail = new DataView(
    await file.slice(file.size - tailLength).arrayBuffer(),
  );
  let end = -1;
  for (let at = tail.byteLength - 22; at >= 0; at -= 1) {
    if (tail.getUint32(at, true) === 0x06054b50) {
      end = at;
      break;
    }
  }
  if (end < 0) return [];
  const count = tail.getUint16(end + 10, true);
  const length = tail.getUint32(end + 12, true);
  const offset = tail.getUint32(end + 16, true);
  const directory = new DataView(
    await file.slice(offset, offset + length).arrayBuffer(),
  );
  const names = new TextDecoder();
  const entries: ArchiveEntry[] = [];
  for (
    let at = 0, read = 0;
    read < count && at + 46 <= directory.byteLength;
    read += 1
  ) {
    if (directory.getUint32(at, true) !== 0x02014b50) break;
    const nameLength = directory.getUint16(at + 28, true);
    entries.push({
      method: directory.getUint16(at + 10, true),
      compressed: directory.getUint32(at + 20, true),
      size: directory.getUint32(at + 24, true),
      local: directory.getUint32(at + 42, true),
      name: names.decode(new Uint8Array(directory.buffer, at + 46, nameLength)),
    });
    at +=
      46 +
      nameLength +
      directory.getUint16(at + 30, true) +
      directory.getUint16(at + 32, true);
  }
  return entries;
}

async function archivedBody(file: File, entry: ArchiveEntry): Promise<Blob> {
  const local = new DataView(
    await file.slice(entry.local, entry.local + 30).arrayBuffer(),
  );
  const start =
    entry.local + 30 + local.getUint16(26, true) + local.getUint16(28, true);
  const stored = file.slice(start, start + entry.compressed);
  if (entry.method === 0) return stored;
  return new Response(
    stored.stream().pipeThrough(new DecompressionStream("deflate-raw")),
  ).blob();
}

async function pngSize(body: Blob): Promise<[number, number] | null> {
  const head = new DataView(await body.slice(0, 24).arrayBuffer());
  if (head.byteLength < 24 || head.getUint32(0) !== 0x89504e47) return null;
  return [head.getUint32(16), head.getUint32(20)];
}

function decodeBase64(text: string): string {
  const binary = atob(text.trim());
  return new TextDecoder().decode(
    Uint8Array.from(binary, (letter) => letter.charCodeAt(0)),
  );
}

function dataPart(
  name: string,
  text: string,
  size: number,
  index: number,
): ScanPart | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return null;
  }
  if (!parsed || typeof parsed !== "object") return null;
  const root = parsed as Record<string, unknown>;
  const spec = typeof root.spec === "string" ? root.spec : "";
  if (spec.startsWith("chara_card")) {
    const data = (root.data ?? root) as Record<string, unknown>;
    return {
      id: String(index),
      name,
      kind: "card",
      size,
      note: `Card data · ${spec.replace("chara_card_", "").toUpperCase()}`,
      lines: cardLines(data),
    };
  }
  return {
    id: String(index),
    name,
    kind: "data",
    size,
    note: `Data · ${Object.keys(root).length} fields`,
    lines: Object.entries(root)
      .slice(0, 14)
      .map(([key, value]) => `${key}: ${glimpse(value)}`),
  };
}

function cardLines(data: Record<string, unknown>): string[] {
  const lines = [`name: ${glimpse(data.name)}`];
  for (const key of ["description", "personality", "scenario", "first_mes"]) {
    if (typeof data[key] === "string" && data[key]) {
      lines.push(`${key}: ${glimpse(data[key])}`);
    }
  }
  const greetings = data.alternate_greetings;
  if (Array.isArray(greetings) && greetings.length > 0) {
    lines.push(`alternate_greetings: ${greetings.length}`);
  }
  const book = data.character_book as { entries?: unknown[] } | undefined;
  if (Array.isArray(book?.entries)) {
    lines.push(`character_book: ${book.entries.length} entries`);
  }
  if (Array.isArray(data.tags) && data.tags.length > 0) {
    lines.push(`tags: ${data.tags.slice(0, 6).join(", ")}`);
  }
  if (Array.isArray(data.assets)) lines.push(`assets: ${data.assets.length}`);
  return lines;
}

function glimpse(value: unknown): string {
  if (typeof value === "string") {
    const flat = value.replaceAll(/\s+/g, " ").trim();
    return flat.length > 72 ? `${flat.slice(0, 72)}…` : flat || '""';
  }
  if (Array.isArray(value)) return `${value.length} items`;
  if (value && typeof value === "object") {
    return `${Object.keys(value).length} fields`;
  }
  return String(value);
}
