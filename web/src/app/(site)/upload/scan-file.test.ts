import { expect, test } from "bun:test";
import { scanFile } from "./scan-file";

const card = JSON.stringify({
  spec: "chara_card_v3",
  data: { name: "Ana", alternate_greetings: ["a", "b"] },
});

function storedZip(files: Record<string, Uint8Array>): Uint8Array<ArrayBuffer> {
  const locals: number[] = [];
  const central: number[] = [];
  const u16 = (value: number) => [value & 255, value >> 8];
  const u32 = (value: number) => [...u16(value & 0xffff), ...u16(value >>> 16)];
  let count = 0;
  for (const [name, body] of Object.entries(files)) {
    const title = [...new TextEncoder().encode(name)];
    const offset = locals.length;
    const sizes = [...u32(0), ...u32(body.length), ...u32(body.length)];
    locals.push(
      ...u32(0x04034b50),
      ...u16(20),
      ...u16(0),
      ...u16(0),
      ...u32(0),
      ...sizes,
      ...u16(title.length),
      ...u16(0),
      ...title,
      ...body,
    );
    central.push(
      ...u32(0x02014b50),
      ...u16(20),
      ...u16(20),
      ...u16(0),
      ...u16(0),
      ...u32(0),
      ...sizes,
      ...u16(title.length),
      ...u16(0),
      ...u16(0),
      ...u16(0),
      ...u16(0),
      ...u32(0),
      ...u32(offset),
      ...title,
    );
    count += 1;
  }
  const end = [
    ...u32(0x06054b50),
    ...u16(0),
    ...u16(0),
    ...u16(count),
    ...u16(count),
    ...u32(central.length),
    ...u32(locals.length),
    ...u16(0),
  ];
  return new Uint8Array([...locals, ...central, ...end]);
}

function png(): Uint8Array {
  const bytes = new Uint8Array(33);
  bytes.set([
    0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 13, 73, 72, 68, 82,
  ]);
  bytes.set([0, 0, 1, 144, 0, 0, 2, 88], 16);
  return bytes;
}

test("a CharX scan lists the card data first and reads each picture's size", async () => {
  const archive = storedZip({
    "assets/icon/image/main.png": png(),
    "card.json": new TextEncoder().encode(card),
  });
  const parts = await scanFile(new File([archive], "ana.charx"));

  expect(parts.map((part) => [part.name, part.kind])).toEqual([
    ["card.json", "card"],
    ["assets/icon/image/main.png", "picture"],
  ]);
  expect(parts[0].lines).toContain("alternate_greetings: 2");
  expect(parts[1].note).toBe("400 × 600 picture");
});
