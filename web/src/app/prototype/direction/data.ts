import type { StaticImageData } from "next/image";
import atlas from "./media/cover-atlas.webp";
import chorus from "./media/cover-chorus.webp";
import ember from "./media/cover-ember.webp";
import lastLight from "./media/cover-last-light.webp";
import verdant from "./media/cover-verdant.webp";
import plate from "./media/figure-plate.webp";
import pictures from "./media/header-pictures.webp";
import structure from "./media/header-structure.webp";

export type Category = "announcement" | "article" | "release";

export type Tint = { a: string; b: string; ink: string; dark: boolean };

export type Body =
  | { kind: "p"; text: string }
  | { kind: "h"; text: string }
  | { kind: "list"; items: string[] }
  | { kind: "quote"; text: string; cite?: string }
  | { kind: "figure"; image: StaticImageData; alt: string; caption: string }
  | { kind: "table"; head: string[]; rows: string[][] }
  | { kind: "code"; lines: string[] };

export type Post = {
  id: string;
  title: string;
  dek: string;
  category: Category;
  author: string;
  handle: string;
  date: string;
  updated?: string;
  minutes: number;
  weight: 1 | 2 | 3;
  image?: StaticImageData;
  tint?: Tint;
  body?: Body[];
};

const TINTS: Record<string, Tint> = {
  nightDesk: { a: "#2f5f8f", b: "#e8a851", ink: "#2b4f78", dark: true },
  lastLight: { a: "#8fa4b0", b: "#f0e6cf", ink: "#4a616f", dark: false },
  ember: { a: "#e8562f", b: "#7a1220", ink: "#8e2a1c", dark: true },
  verdant: { a: "#2f8f6d", b: "#7fbb6a", ink: "#1f6b52", dark: true },
  atlas: { a: "#b99a63", b: "#efe4cd", ink: "#7a5a22", dark: false },
  chorus: { a: "#8a5cf0", b: "#e0609a", ink: "#5b3ba6", dark: true },
  structure: { a: "#3d7fb5", b: "#1d3b5c", ink: "#2c5f8c", dark: true },
  pictures: { a: "#c58f5c", b: "#f3e6d4", ink: "#8a5a30", dark: false },
  plate: { a: "#6f7cff", b: "#1a1b22", ink: "#4550c8", dark: true },
};

export const DARK_TINT: Tint = {
  a: "#7f8fd0",
  b: "#d09aa4",
  ink: "#a9b6e8",
  dark: true,
};

const LONG_TITLE =
  "Everything a character card keeps when it moves between applications, and the three fields that still get lost on the way";

const STRUCTURE_BODY: Body[] = [
  {
    kind: "p",
    text: "A character page is read once, quickly, by someone deciding whether to spend an evening with it. That reader wants four things in order: who this is, how it speaks, what it needs from their app, and how to get it. Everything else is reference they will come back for later, if at all.",
  },
  {
    kind: "p",
    text: "Illarin used to give all four the same weight. The description, the greetings, the attribute table and the download all sat in equal boxes down the page, and the reader had to sort them out. **A page that treats everything as equally important has no hierarchy at all.**",
  },
  { kind: "h", text: "Reading order is a design decision" },
  {
    kind: "p",
    text: "The new page fixes the order and gives each part the room its content needs, instead of the room the grid happens to have. Prose gets a reading measure. A greeting gets the full width, because dialogue breaks badly in a narrow column. An attribute table gets a narrow column, because a table of short pairs looks absurd stretched across a page.",
  },
  {
    kind: "figure",
    image: structure,
    alt: "A page divided into four bands of differing width",
    caption:
      "Each band takes the width its content reads best at. The bands do not have to agree with each other.",
  },
  {
    kind: "quote",
    text: "Space is not a budget to spend evenly. It is the thing that tells a reader what to look at first.",
  },
  { kind: "h", text: "What a card actually carries" },
  {
    kind: "p",
    text: "Under the page is a file, and the file has a shape that other applications already agree on. Illarin keeps every field it receives, including the ones it does not display, so an export is the same object that came in.",
  },
  {
    kind: "table",
    head: ["Field", "Character Card V2", "Character Card V3", "Kept"],
    rows: [
      ["description", "yes", "yes", "always"],
      ["first_mes", "yes", "yes", "always"],
      ["alternate_greetings", "no", "yes", "always"],
      ["group_only_greetings", "no", "yes", "always"],
      ["extensions", "passthrough", "passthrough", "byte for byte"],
      ["creator_notes_multilingual", "no", "yes", "always"],
    ],
  },
  {
    kind: "p",
    text: "The rows marked _passthrough_ are the important ones. Illarin does not know what another application put there, so it does not touch it. An edit made on this page rewrites the fields the page shows and leaves the rest exactly as it found them.",
  },
  {
    kind: "code",
    lines: [
      '"extensions": {',
      '  "depth_prompt": { "depth": 4, "prompt": "..." },',
      '  "some_other_app": { "kept": true }',
      "}",
    ],
  },
  { kind: "h", text: "Three things still go missing" },
  {
    kind: "list",
    items: [
      "A tagline written in one application has no field in another, so it survives the round trip only as a note.",
      "Per-greeting labels are not part of any agreed format, so they are stored beside the file rather than in it.",
      "An avatar embedded in a PNG loses its colour profile in applications that re-encode on import.",
    ],
  },
  {
    kind: "figure",
    image: plate,
    alt: "A row of shelves under a single overhead light",
    caption: "The fields that survive a round trip, and the three that do not.",
  },
  {
    kind: "p",
    text: "None of the three is a reason to hold a card back. They are reasons to keep the original file, which Illarin does, and to offer it beside every generated export.",
  },
];

const PICTURES_BODY: Body[] = [
  {
    kind: "p",
    text: "Pictures in a post now behave like the rest of the page. A header picture spans the full width and the title crosses onto it. A picture inside the body breaks the reading measure deliberately, and its caption sits in the margin beside it rather than centred underneath.",
  },
  {
    kind: "figure",
    image: pictures,
    alt: "A warm interior lit from the upper right",
    caption:
      "A body picture at full width, with its caption held in the right margin.",
  },
  {
    kind: "p",
    text: "Alternative text is required before a post can be published. The editor will not let a picture through without it.",
  },
];

const SPARSE_BODY: Body[] = [
  {
    kind: "p",
    text: "Sealed prompts are available on presets from today. A sealed prompt keeps its text out of the export and out of the page, and names the applications allowed to run it.",
  },
  {
    kind: "p",
    text: "Nothing changes for presets that do not use one. Existing downloads keep working and no file is rewritten.",
  },
];

export const RICH_POSTS: Post[] = [
  {
    id: "p1",
    title: "What a character page owes its reader",
    dek: "Four questions in order, and the width each answer reads best at.",
    category: "article",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "6 September 2026",
    minutes: 9,
    weight: 3,
    image: structure,
    tint: TINTS.structure,
    body: STRUCTURE_BODY,
  },
  {
    id: "p2",
    title: "The lorebook index, and why it beats a wall of entries",
    dek: "Sixty-four entries used to render twenty thousand pixels tall. Now they render as an index beside the one entry you opened.",
    category: "article",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "2 September 2026",
    minutes: 6,
    weight: 2,
    image: verdant,
    tint: TINTS.verdant,
  },
  {
    id: "p3",
    title: "Presets carry their own sealed prompts now",
    dek: "A sealed prompt stays out of the export and names the applications allowed to run it.",
    category: "announcement",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "29 August 2026",
    minutes: 2,
    weight: 1,
    body: SPARSE_BODY,
  },
  {
    id: "p4",
    title: "Pictures in a post",
    dek: "What a header plate, a body picture and a gallery look like once they are placed.",
    category: "announcement",
    author: "Ilse Verrin",
    handle: "@ilse",
    date: "27 August 2026",
    minutes: 3,
    weight: 2,
    image: pictures,
    tint: TINTS.pictures,
    body: PICTURES_BODY,
  },
  {
    id: "p5",
    title: LONG_TITLE,
    dek: "A round trip through four applications, field by field, with the losses written down.",
    category: "article",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "24 August 2026",
    updated: "1 September 2026",
    minutes: 14,
    weight: 2,
    image: atlas,
    tint: TINTS.atlas,
  },
  {
    id: "p6",
    title: "Illarin keeps its own writing",
    dek: "Announcements and articles live here now, stored as structure rather than markup.",
    category: "announcement",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "20 August 2026",
    minutes: 4,
    weight: 1,
  },
  {
    id: "p7",
    title: "A pack is five things at once",
    dek: "Packs hold a character, its lorebook, a preset and a theme, and export each of them separately.",
    category: "release",
    author: "Ilse Verrin",
    handle: "@ilse",
    date: "16 August 2026",
    minutes: 5,
    weight: 2,
    image: chorus,
    tint: TINTS.chorus,
  },
  {
    id: "p8",
    title: "Reading a world without opening every door",
    dek: "Search, keys and the entry you actually wanted.",
    category: "article",
    author: "Marek Sten",
    handle: "@marek",
    date: "11 August 2026",
    minutes: 7,
    weight: 1,
    image: plate,
    tint: TINTS.plate,
  },
];

export const SPARSE_POSTS: Post[] = [
  {
    id: "s1",
    title: "Presets carry their own sealed prompts now",
    dek: "A sealed prompt stays out of the export and names the applications allowed to run it.",
    category: "announcement",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "29 August 2026",
    minutes: 2,
    weight: 3,
    body: SPARSE_BODY,
  },
  {
    id: "s2",
    title: LONG_TITLE,
    dek: "A round trip through four applications, field by field, with the losses written down.",
    category: "article",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "24 August 2026",
    minutes: 14,
    weight: 2,
  },
  {
    id: "s3",
    title: "Illarin keeps its own writing",
    dek: "Announcements and articles live here now, stored as structure rather than markup.",
    category: "announcement",
    author: "Wren Ashdown",
    handle: "@wren",
    date: "20 August 2026",
    minutes: 4,
    weight: 1,
  },
];

export const OTHER_WORK = [
  {
    name: "The West Shelf",
    kind: "Lorebook",
    cover: verdant,
    creator: "@wren",
  },
  {
    name: "Morrow, keeper of the last light",
    kind: "Character",
    cover: lastLight,
    creator: "@marek",
  },
  { name: "Ashfall chorus", kind: "Pack", cover: chorus, creator: "@ilse" },
  {
    name: "The long summer",
    kind: "Character",
    cover: ember,
    creator: "@ilse",
  },
];

export const CATEGORY_LABEL: Record<Category, string> = {
  announcement: "Announcement",
  article: "Article",
  release: "Release notes",
};

export const CATEGORY_PIGMENT: Record<Category, string> = {
  announcement: "var(--v-indigo)",
  article: "var(--v-oxide)",
  release: "var(--v-verdigris)",
};
