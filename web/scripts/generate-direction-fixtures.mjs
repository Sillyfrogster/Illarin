// Draws the synthetic cover and header art the visual direction prototype reads
import { mkdir, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const OUT = join(
  dirname(fileURLToPath(import.meta.url)),
  "..",
  "src",
  "app",
  "prototype",
  "direction",
  "media",
);

// A fixed sequence, so every run draws the same art
function sequence(seed) {
  let state = seed >>> 0;
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0;
    return state / 4294967296;
  };
}

function grain(width, height, strength, seed) {
  const next = sequence(seed);
  const pixels = Buffer.alloc(width * height * 4);
  for (let i = 0; i < width * height; i += 1) {
    const value = Math.round(next() * 255);
    pixels[i * 4] = value;
    pixels[i * 4 + 1] = value;
    pixels[i * 4 + 2] = value;
    pixels[i * 4 + 3] = strength;
  }
  return sharp(pixels, { raw: { width, height, channels: 4 } })
    .png()
    .toBuffer();
}

function blob(x, y, radius, colour, opacity, blur) {
  return `<ellipse cx="${x}" cy="${y}" rx="${radius}" ry="${radius * 0.82}"
    fill="${colour}" opacity="${opacity}" filter="url(#soft${blur})" />`;
}

function scene({ width, height, ground, sky, lights, figure }) {
  const blurs = [40, 90, 170]
    .map(
      (radius) =>
        `<filter id="soft${radius}" x="-60%" y="-60%" width="220%" height="220%">
          <feGaussianBlur stdDeviation="${radius}" /></filter>`,
    )
    .join("");
  const glow = lights
    .map(([x, y, radius, colour, opacity, blur]) =>
      blob(x * width, y * height, radius * width, colour, opacity, blur),
    )
    .join("");
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}">
    <defs>
      ${blurs}
      <linearGradient id="field" x1="0" y1="0" x2="0.35" y2="1">
        <stop offset="0" stop-color="${sky}" />
        <stop offset="1" stop-color="${ground}" />
      </linearGradient>
    </defs>
    <rect width="${width}" height="${height}" fill="url(#field)" />
    ${glow}
    ${figure ? figure(width, height) : ""}
  </svg>`;
}

// A standing figure, kept as a silhouette so no synthetic face is invented
function standing(tone, opacity) {
  return (width, height) => {
    const x = width * 0.52;
    const head = height * 0.28;
    return `<g fill="${tone}" opacity="${opacity}" filter="url(#soft40)">
      <circle cx="${x}" cy="${head}" r="${width * 0.105}" />
      <path d="M ${x - width * 0.2} ${height * 1.05}
        C ${x - width * 0.19} ${height * 0.52}, ${x - width * 0.13} ${head + height * 0.1}, ${x} ${head + height * 0.1}
        C ${x + width * 0.13} ${head + height * 0.1}, ${x + width * 0.19} ${height * 0.52}, ${x + width * 0.2} ${height * 1.05} Z" />
    </g>`;
  };
}

// A horizon of shelves, for the pieces that stand in for a book or a world
function shelves(tone, opacity) {
  return (width, height) => {
    const bars = [];
    for (let i = 0; i < 7; i += 1) {
      const w = width * (0.05 + (i % 3) * 0.018);
      const h = height * (0.3 + (i % 4) * 0.075);
      bars.push(
        `<rect x="${width * (0.12 + i * 0.108)}" y="${height - h}" width="${w}" height="${h}" rx="${w * 0.12}" />`,
      );
    }
    return `<g fill="${tone}" opacity="${opacity}" filter="url(#soft40)">${bars.join("")}</g>`;
  };
}

const PIECES = [
  {
    name: "cover-night-desk",
    width: 900,
    height: 1200,
    sky: "#101a2e",
    ground: "#05070f",
    lights: [
      [0.72, 0.34, 0.5, "#e8a851", 0.5, 170],
      [0.28, 0.66, 0.42, "#2f6f8f", 0.42, 170],
      [0.62, 0.24, 0.16, "#ffd9a0", 0.7, 90],
    ],
    figure: standing("#03050b", 0.62),
  },
  {
    name: "cover-last-light",
    width: 900,
    height: 1200,
    sky: "#c9d4d8",
    ground: "#5d6f79",
    lights: [
      [0.5, 0.2, 0.44, "#fdf6e6", 0.72, 170],
      [0.18, 0.78, 0.4, "#38505f", 0.5, 170],
      [0.5, 0.22, 0.1, "#ffffff", 0.85, 40],
    ],
    figure: standing("#28363f", 0.5),
  },
  {
    name: "cover-ember",
    width: 900,
    height: 1200,
    sky: "#3a0f0d",
    ground: "#120406",
    lights: [
      [0.44, 0.58, 0.46, "#e8562f", 0.55, 170],
      [0.66, 0.3, 0.28, "#f0a63c", 0.5, 90],
      [0.2, 0.82, 0.3, "#7a1220", 0.6, 170],
    ],
    figure: standing("#150305", 0.6),
  },
  {
    name: "cover-verdant",
    width: 1400,
    height: 1050,
    sky: "#0f2a24",
    ground: "#05100e",
    lights: [
      [0.3, 0.36, 0.34, "#2f8f6d", 0.5, 170],
      [0.74, 0.62, 0.3, "#7fbb6a", 0.34, 170],
      [0.5, 0.18, 0.18, "#d8f0c8", 0.34, 90],
    ],
    figure: shelves("#03110d", 0.5),
  },
  {
    name: "cover-atlas",
    width: 1600,
    height: 900,
    sky: "#efe4cd",
    ground: "#b99a63",
    lights: [
      [0.24, 0.3, 0.34, "#fff8e8", 0.7, 170],
      [0.78, 0.7, 0.34, "#8a6a35", 0.45, 170],
    ],
    figure: shelves("#4a3517", 0.32),
  },
  {
    name: "cover-chorus",
    width: 1100,
    height: 1100,
    sky: "#241436",
    ground: "#0a0512",
    lights: [
      [0.36, 0.32, 0.4, "#8a5cf0", 0.5, 170],
      [0.7, 0.66, 0.34, "#e0609a", 0.45, 170],
      [0.52, 0.5, 0.14, "#f4dcff", 0.6, 90],
    ],
    figure: standing("#0b0517", 0.55),
  },
  {
    name: "header-structure",
    width: 2000,
    height: 1125,
    sky: "#152233",
    ground: "#050810",
    lights: [
      [0.18, 0.28, 0.36, "#3d7fb5", 0.5, 170],
      [0.82, 0.72, 0.36, "#1d3b5c", 0.6, 170],
      [0.5, 0.44, 0.22, "#cfe6ff", 0.34, 90],
    ],
  },
  {
    name: "header-pictures",
    width: 2000,
    height: 1125,
    sky: "#f3e6d4",
    ground: "#c58f5c",
    lights: [
      [0.7, 0.3, 0.34, "#fff6e6", 0.75, 170],
      [0.22, 0.74, 0.34, "#9a5a35", 0.45, 170],
    ],
  },
  {
    name: "figure-plate",
    width: 1600,
    height: 1000,
    sky: "#1a1b22",
    ground: "#08080b",
    lights: [
      [0.5, 0.42, 0.4, "#6f7cff", 0.34, 170],
      [0.5, 0.42, 0.12, "#e6e9ff", 0.5, 90],
    ],
    figure: shelves("#04040a", 0.55),
  },
];

async function draw(piece) {
  const noise = await grain(
    piece.width,
    piece.height,
    12,
    piece.name.length * 7717,
  );
  const image = await sharp(Buffer.from(scene(piece)))
    .composite([{ input: noise, blend: "overlay" }])
    .webp({ quality: 82 })
    .toBuffer();
  await writeFile(join(OUT, `${piece.name}.webp`), image);
  return `${piece.name}.webp ${(image.length / 1024).toFixed(0)}kB`;
}

await mkdir(OUT, { recursive: true });
for (const piece of PIECES) console.log(await draw(piece));
