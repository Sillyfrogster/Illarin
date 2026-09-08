import type { StaticImageData } from "next/image";
import type { Tint } from "./data";
import atlas from "./media/cover-atlas.webp";
import chorus from "./media/cover-chorus.webp";
import ember from "./media/cover-ember.webp";
import lastLight from "./media/cover-last-light.webp";
import nightDesk from "./media/cover-night-desk.webp";
import verdant from "./media/cover-verdant.webp";

export type Kind = "character" | "lorebook" | "preset" | "theme" | "pack";

export type Width = "full" | "two_thirds" | "half" | "third";

export type Layout =
  | "single"
  | "duo"
  | "main-aside"
  | "trio"
  | "stack-2"
  | "stack-3";

export type LoreEntry = {
  id: string;
  name: string;
  keys: string[];
  secondaryKeys?: string[];
  body: string;
  enabled: boolean;
  constant?: boolean;
  order: number;
  position: string;
  recursion?: "blocked" | "prevented" | "delayed";
};

export type Fragment = {
  id: string;
  name: string;
  role: "system" | "user" | "assistant";
  placement: "relative" | "absolute";
  depth?: number;
  enabled: boolean;
  group?: string;
  body: string;
};

export type Variable = {
  id: string;
  name: string;
  type: "string" | "number" | "boolean" | "choice";
  fallback: string;
  choices?: string[];
  note: string;
};

export type Script = {
  id: string;
  name: string;
  find: string;
  replace: string;
  targets: string[];
  enabled: boolean;
};

export type Lumia = {
  id: string;
  lumiaName: string;
  lumiaDefinition: string;
  lumiaPersonality: string;
  lumiaBehavior: string;
  avatar: StaticImageData;
  genderIdentity: string;
  authorName: string;
  version: string;
};

export type Element =
  | {
      id: string;
      type: "prose";
      label: string;
      role?: string;
      display: "rich" | "verbatim";
      body: string;
    }
  | {
      id: string;
      type: "text_set";
      label: string;
      role?: string;
      items: { id: string; name: string; body: string }[];
    }
  | {
      id: string;
      type: "field_list";
      label: string;
      role?: string;
      fields: { name: string; value: string }[];
    }
  | {
      id: string;
      type: "dialogue_sample";
      label: string;
      role?: string;
      turns: { speaker: "char" | "user"; line: string }[];
    }
  | {
      id: string;
      type: "entry_table";
      label: string;
      role?: string;
      total: number;
      enabledTotal: number;
      entries: LoreEntry[];
    }
  | {
      id: string;
      type: "image_set";
      label: string;
      role?: string;
      itemSize: "small" | "medium" | "large";
      images: { id: string; src: StaticImageData; name?: string }[];
    }
  | {
      id: string;
      type: "link_list";
      label: string;
      role?: string;
      links: { id: string; label: string; note: string; kind: Kind }[];
    }
  | {
      id: string;
      type: "prompt_list";
      label: string;
      role?: string;
      total: number;
      fragments: Fragment[];
    }
  | {
      id: string;
      type: "variable_schema";
      label: string;
      role?: string;
      variables: Variable[];
    }
  | {
      id: string;
      type: "setting_group";
      label: string;
      role?: string;
      settings: { name: string; value: string }[];
    }
  | {
      id: string;
      type: "script_list";
      label: string;
      role?: string;
      scripts: Script[];
    }
  | {
      id: string;
      type: "color_set";
      label: string;
      role?: string;
      tokens: { name: string; value: string }[];
    }
  | {
      id: string;
      type: "stylesheet_set";
      label: string;
      role?: string;
      sheets: { id: string; name: string; size: string; lines: string[] }[];
    }
  | {
      id: string;
      type: "record_list";
      label: string;
      role?: string;
      schema: "lumia";
      records: Lumia[];
    };

export type Block = {
  id: string;
  definition: string;
  title: string;
  layout: Layout;
  width: Width;
  elements: Element[];
};

export type Format = {
  label: string;
  note: string;
  recommended?: boolean;
  verdict: "carried" | "reduced" | "blocked";
  drops?: string[];
};

export type Release = {
  version: string;
  date: string;
  summary: string;
  changes: string[];
  additions: number;
  removals: number;
};

export type Asset = {
  id: string;
  kind: Kind;
  name: string;
  nickname?: string;
  assetVersion: string;
  creditedAuthor: string;
  blurb?: string;
  creator: string;
  shared: string;
  tags: string[];
  cover?: StaticImageData;
  tint: Tint;
  blocks: Block[];
  formats: Format[];
  releases: Release[];
};

export const KIND_LABEL: Record<Kind, string> = {
  character: "Character",
  lorebook: "Lorebook",
  preset: "Preset",
  theme: "Theme",
  pack: "Pack",
};

export const WIDTH_SPAN: Record<Width, number> = {
  full: 12,
  two_thirds: 8,
  half: 6,
  third: 4,
};

const TINTS: Record<string, Tint> = {
  nightDesk: { a: "#2f4d78", b: "#8a6a4a", ink: "#22364f", dark: true },
  atlas: { a: "#3c5a4a", b: "#7d6a48", ink: "#25402f", dark: true },
  chorus: { a: "#5a4a78", b: "#8a5a6a", ink: "#3d3054", dark: true },
  ember: { a: "#7a4030", b: "#8a6a3a", ink: "#5a2f24", dark: true },
  verdant: { a: "#2f5f4c", b: "#5a7a3a", ink: "#1f4536", dark: true },
  lastLight: { a: "#4a5a7a", b: "#8a7060", ink: "#33415c", dark: true },
};

const DESCRIPTION =
  "Marisol keeps the night desk at a record office that no department will admit to owning. She has worked the same twelve hours for eleven years, and in that time she has read most of the building.\n\nShe is dry, unhurried and entirely unbothered by rank. She will find you anything on the first three floors within four minutes. She will not discuss the fourth shelf, and she will not explain why she will not discuss it.";

const GREETINGS: { id: string; name: string; body: string }[] = [
  {
    id: "g1",
    name: "The first night",
    body: 'The desk lamp is the only light on the floor. She does not look up when the door goes; she finishes the line she is on, caps the pen, and turns the ledger around so you can read it upside down.\n\n"You want the third floor. Everybody who comes in at this hour wants the third floor."',
  },
  {
    id: "g2",
    name: "Closing time",
    body: '"We shut at six. It is nine." She says it without any particular urgency, and does not move to shut anything.',
  },
  {
    id: "g3",
    name: "The fourth shelf",
    body: 'You are two steps down the wrong aisle when her voice arrives behind you, level as ever.\n\n"Not that one."',
  },
  {
    id: "g4",
    name: "A returned book",
    body: 'She turns the returned volume over twice, checks the spine against her ledger, and makes a small mark. "Eleven days late. I have written down that it was nine."',
  },
];

const GROUP_GREETINGS: { id: string; name: string; body: string }[] = [
  {
    id: "gg1",
    name: "Two at the desk",
    body: 'She looks at the pair of you the way she looks at a mis-filed box. "One of you talks. I do not care which."',
  },
];

const ENTRIES: LoreEntry[] = [
  {
    id: "e1",
    name: "The fourth shelf",
    keys: ["fourth shelf", "shelf four", "the fourth"],
    secondaryKeys: ["basement"],
    body: "Nothing on the fourth shelf has a card. Marisol has never filed anything there and has never removed anything from it. She will change the subject twice and then stop answering.",
    enabled: true,
    constant: false,
    order: 100,
    position: "before character",
    recursion: "blocked",
  },
  {
    id: "e2",
    name: "Verrin municipal record office",
    keys: ["record office", "verrin", "the building"],
    body: "Four floors, one working lift, and a heating system that answers to nobody. The building was requisitioned twice and returned once. The paperwork for the second requisition is on the fourth shelf.",
    enabled: true,
    constant: true,
    order: 90,
    position: "after character",
  },
  {
    id: "e3",
    name: "The ledger",
    keys: ["ledger", "the book", "her book"],
    body: "Marisol keeps her own ledger in parallel with the official one. Where the two disagree, hers is right, and she has never once been asked to prove it.",
    enabled: true,
    order: 80,
    position: "after character",
    recursion: "prevented",
  },
  {
    id: "e4",
    name: "Aldon Reeve",
    keys: ["reeve", "aldon", "the inspector"],
    body: "An inspector who came for three days in the spring and stayed eleven. He signed for a box he did not take. Marisol still has the slip.",
    enabled: false,
    order: 70,
    position: "before character",
  },
];

const EXAMPLE: { speaker: "char" | "user"; line: string }[] = [
  { speaker: "user", line: "Is there a catalogue?" },
  {
    speaker: "char",
    line: "There are four. Two are wrong, one is in the basement, and one is me.",
  },
  { speaker: "user", line: "Which one should I use?" },
  { speaker: "char", line: "You are using it." },
];

export const CHARACTER_ASSET: Asset = {
  id: "a1",
  kind: "character",
  name: "Marisol of the Night Desk",
  nickname: "Marisol",
  assetVersion: "1.4",
  creditedAuthor: "Wren Ashdown",
  blurb:
    "The night desk keeper at a record office nobody signed for, and she will not discuss the fourth shelf.",
  creator: "@wren",
  shared: "20 August 2026",
  tags: [
    "original character",
    "library",
    "slow burn",
    "night shift",
    "archivist",
    "mystery",
  ],
  cover: nightDesk,
  tint: TINTS.nightDesk,
  formats: [
    {
      label: "Character Card V3",
      note: "Carries every element on this page",
      recommended: true,
      verdict: "carried",
    },
    {
      label: "Character Card V2",
      note: "Older card format",
      verdict: "reduced",
      drops: [
        "the group-only greeting",
        "the gallery",
        "the expression set",
        "entry recursion flags",
      ],
    },
    {
      label: "Lumiverse character",
      note: "Cross-application format",
      verdict: "reduced",
      drops: ["the image prompts", "the changelog"],
    },
  ],
  blocks: [
    {
      id: "b1",
      definition: "character_core",
      title: "The character",
      layout: "stack-3",
      width: "two_thirds",
      elements: [
        {
          id: "b1e1",
          type: "prose",
          label: "Description",
          role: "description",
          display: "rich",
          body: DESCRIPTION,
        },
        {
          id: "b1e2",
          type: "prose",
          label: "Personality",
          role: "personality",
          display: "rich",
          body: "Dry, patient, unhurried. Never raises her voice. Answers questions she was not asked and ignores the ones she was.",
        },
        {
          id: "b1e3",
          type: "prose",
          label: "Scenario",
          role: "scenario",
          display: "rich",
          body: "The record office after closing, one lamp lit at the desk and the rest of the floor dark.",
        },
      ],
    },
    {
      id: "b2",
      definition: "attributes",
      title: "Attributes",
      layout: "single",
      width: "third",
      elements: [
        {
          id: "b2e1",
          type: "field_list",
          label: "Details",
          fields: [
            { name: "Age", value: "34" },
            { name: "Species", value: "Human" },
            { name: "Occupation", value: "Night-desk archivist" },
            { name: "Height", value: "168 cm" },
            { name: "Eyes", value: "Grey" },
            { name: "Pronouns", value: "she / her" },
            { name: "Speech", value: "Level, unhurried" },
            { name: "Carries", value: "One coat, one pen" },
          ],
        },
      ],
    },
    {
      id: "b3",
      definition: "messages",
      title: "Messages",
      layout: "stack-3",
      width: "full",
      elements: [
        {
          id: "b3e1",
          type: "text_set",
          label: "Greetings",
          role: "greetings",
          items: GREETINGS,
        },
        {
          id: "b3e2",
          type: "dialogue_sample",
          label: "Example dialogue",
          role: "example_dialogue",
          turns: EXAMPLE,
        },
        {
          id: "b3e3",
          type: "text_set",
          label: "Group-only greetings",
          role: "group_greetings",
          items: GROUP_GREETINGS,
        },
      ],
    },
    {
      id: "b4",
      definition: "lorebook",
      title: "The world she works in",
      layout: "single",
      width: "two_thirds",
      elements: [
        {
          id: "b4e1",
          type: "entry_table",
          label: "Entries",
          role: "lorebook_entries",
          total: 34,
          enabledTotal: 31,
          entries: ENTRIES,
        },
      ],
    },
    {
      id: "b5",
      definition: "author_notes",
      title: "Author's notes",
      layout: "single",
      width: "third",
      elements: [
        {
          id: "b5e1",
          type: "prose",
          label: "How to play her",
          role: "creator_notes",
          display: "rich",
          body: "Slow scenes suit her. Let the building answer before she does. She is more interesting when she is refusing something than when she is helping.",
        },
      ],
    },
    {
      id: "b6",
      definition: "gallery",
      title: "Gallery",
      layout: "single",
      width: "half",
      elements: [
        {
          id: "b6e1",
          type: "image_set",
          label: "Images",
          role: "gallery",
          itemSize: "medium",
          images: [
            { id: "i1", src: nightDesk, name: "At the desk" },
            { id: "i2", src: lastLight, name: "The third floor, late" },
            { id: "i3", src: atlas, name: "The ledger" },
            { id: "i4", src: ember, name: "Closing" },
          ],
        },
      ],
    },
    {
      id: "b7",
      definition: "expressions",
      title: "Expressions",
      layout: "single",
      width: "half",
      elements: [
        {
          id: "b7e1",
          type: "image_set",
          label: "Set",
          role: "expressions",
          itemSize: "small",
          images: [
            { id: "x1", src: nightDesk, name: "neutral" },
            { id: "x2", src: ember, name: "unimpressed" },
            { id: "x3", src: atlas, name: "amused" },
            { id: "x4", src: chorus, name: "closing the ledger" },
            { id: "x5", src: verdant, name: "not that one" },
            { id: "x6", src: lastLight, name: "tired" },
          ],
        },
      ],
    },
    {
      id: "b8",
      definition: "image_prompts",
      title: "How the art was made",
      layout: "stack-3",
      width: "two_thirds",
      elements: [
        {
          id: "b8e1",
          type: "field_list",
          label: "Parameters",
          fields: [
            { name: "Model", value: "local, 30 steps" },
            { name: "Sampler", value: "dpmpp_2m" },
            { name: "Seed", value: "774120" },
            { name: "Size", value: "832 x 1216" },
          ],
        },
        {
          id: "b8e2",
          type: "prose",
          label: "Prompt",
          display: "verbatim",
          body: "night archive desk, single lamp, grey coat, ledger open, deep shadow, muted palette, film grain",
        },
        {
          id: "b8e3",
          type: "prose",
          label: "Negative prompt",
          display: "verbatim",
          body: "bright daylight, saturated colour, modern office, crowd",
        },
      ],
    },
    {
      id: "b9",
      definition: "model_instructions",
      title: "Model instructions",
      layout: "stack-2",
      width: "half",
      elements: [
        {
          id: "b9e1",
          type: "prose",
          label: "System prompt",
          role: "system_prompt",
          display: "verbatim",
          body: "Stay in Marisol's voice. Keep replies short. Never volunteer what is on the fourth shelf.",
        },
        {
          id: "b9e2",
          type: "prose",
          label: "Post-history instructions",
          role: "post_history_instructions",
          display: "verbatim",
          body: "If asked directly about the fourth shelf, change the subject once, then decline plainly.",
        },
      ],
    },
    {
      id: "b10",
      definition: "relationships",
      title: "Related work",
      layout: "single",
      width: "third",
      elements: [
        {
          id: "b10e1",
          type: "link_list",
          label: "Links",
          links: [
            {
              id: "l1",
              label: "The West Shelf",
              note: "The lorebook her ledger lives in",
              kind: "lorebook",
            },
            {
              id: "l2",
              label: "Night desk preset",
              note: "The prompt settings she was written against",
              kind: "preset",
            },
            {
              id: "l3",
              label: "Record office pack",
              note: "Her and four colleagues",
              kind: "pack",
            },
          ],
        },
      ],
    },
    {
      id: "b11",
      definition: "usage",
      title: "How to run her",
      layout: "single",
      width: "half",
      elements: [
        {
          id: "b11e1",
          type: "prose",
          label: "Notes",
          display: "rich",
          body: "She was written against a 16k context. Below that, drop the lorebook to the eight constant entries and she still holds. Temperature above 1.1 makes her chatty, which is wrong for her.",
        },
      ],
    },
    {
      id: "b12",
      definition: "changelog",
      title: "Changelog",
      layout: "single",
      width: "half",
      elements: [
        {
          id: "b12e1",
          type: "text_set",
          label: "Entries",
          items: [
            {
              id: "c1",
              name: "1.4",
              body: "Two new greetings and a rewritten scenario.",
            },
            {
              id: "c2",
              name: "1.2",
              body: "Attributes tightened, one attribute removed.",
            },
            {
              id: "c3",
              name: "1.0",
              body: "First published version, taken from the uploaded file.",
            },
          ],
        },
      ],
    },
  ],
  releases: [
    {
      version: "1.4",
      date: "6 September 2026",
      summary: "Two new greetings and a rewritten scenario.",
      changes: [
        "Added the greeting Closing time",
        "Added the greeting The fourth shelf",
        "Rewrote Scenario",
      ],
      additions: 3,
      removals: 0,
    },
    {
      version: "1.2",
      date: "28 August 2026",
      summary: "Attributes tightened, one attribute removed.",
      changes: ["Edited Description", "Removed the attribute Hair"],
      additions: 1,
      removals: 1,
    },
    {
      version: "1.0",
      date: "20 August 2026",
      summary: "The first recorded version, taken from the uploaded file.",
      changes: ["Recorded from the creator's own file"],
      additions: 0,
      removals: 0,
    },
  ],
};

export const SPARSE_ASSET: Asset = {
  id: "a2",
  kind: "character",
  name: "Halden",
  assetVersion: "0.01",
  creditedAuthor: "sy",
  creator: "@sy",
  shared: "4 September 2026",
  tags: [],
  tint: TINTS.lastLight,
  formats: [
    {
      label: "Character Card V2",
      note: "The file this was uploaded as",
      recommended: true,
      verdict: "carried",
    },
  ],
  blocks: [
    {
      id: "s1",
      definition: "character_core",
      title: "The character",
      layout: "stack-3",
      width: "two_thirds",
      elements: [
        {
          id: "s1e1",
          type: "prose",
          label: "Description",
          role: "description",
          display: "rich",
          body: "A courier. Reliable. Does not ask what is in the box.",
        },
      ],
    },
    {
      id: "s2",
      definition: "messages",
      title: "Messages",
      layout: "stack-2",
      width: "full",
      elements: [
        {
          id: "s2e1",
          type: "text_set",
          label: "Greetings",
          role: "greetings",
          items: [
            {
              id: "sg1",
              name: "First meeting",
              body: '"Sign here. And here. That one\'s for me."',
            },
          ],
        },
      ],
    },
  ],
  releases: [],
};

export const LOREBOOK_ASSET: Asset = {
  id: "a3",
  kind: "lorebook",
  name: "The West Shelf",
  assetVersion: "3",
  creditedAuthor: "Wren Ashdown",
  blurb:
    "Everything the Verrin record office has lost, catalogued by the people who lost it.",
  creator: "@wren",
  shared: "12 July 2026",
  tags: ["worldbuilding", "records", "verrin", "companion book"],
  cover: atlas,
  tint: TINTS.atlas,
  formats: [
    {
      label: "Lumiverse lorebook",
      note: "Carries every entry and its recursion settings",
      recommended: true,
      verdict: "carried",
    },
    {
      label: "Character Card V3 book",
      note: "Embedded in a character card",
      verdict: "reduced",
      drops: ["the delayed-recursion setting on 14 entries"],
    },
  ],
  blocks: [
    {
      id: "lb1",
      definition: "lorebook_core",
      title: "Entries",
      layout: "single",
      width: "full",
      elements: [
        {
          id: "lb1e1",
          type: "entry_table",
          label: "Entries",
          role: "lorebook_entries",
          total: 285,
          enabledTotal: 271,
          entries: ENTRIES,
        },
      ],
    },
    {
      id: "lb2",
      definition: "usage",
      title: "How to run it",
      layout: "single",
      width: "half",
      elements: [
        {
          id: "lb2e1",
          type: "prose",
          label: "Notes",
          display: "rich",
          body: "Scan depth 4 is enough. Above that the building entries start firing on the word floor, which is not what anybody wants.",
        },
      ],
    },
    {
      id: "lb3",
      definition: "attributes",
      title: "At a glance",
      layout: "single",
      width: "third",
      elements: [
        {
          id: "lb3e1",
          type: "field_list",
          label: "Details",
          fields: [
            { name: "Entries", value: "285" },
            { name: "Enabled", value: "271" },
            { name: "Constant", value: "8" },
            { name: "Scan depth", value: "4" },
            { name: "Token budget", value: "1200" },
          ],
        },
      ],
    },
  ],
  releases: [
    {
      version: "3",
      date: "12 July 2026",
      summary: "Sixty-one entries added for the fourth floor.",
      changes: ["Added 61 entries", "Disabled 4 duplicate entries"],
      additions: 61,
      removals: 4,
    },
  ],
};

export const PRESET_ASSET: Asset = {
  id: "a4",
  kind: "preset",
  name: "Night desk",
  assetVersion: "main",
  creditedAuthor: "Wren Ashdown",
  blurb:
    "A quiet, low-temperature preset for slow scenes and characters who do not fill silence.",
  creator: "@wren",
  shared: "2 June 2026",
  tags: ["roleplay", "low temperature", "slow burn"],
  cover: chorus,
  tint: TINTS.chorus,
  formats: [
    {
      label: "Lumiverse preset",
      note: "Carries the fragments, variables and scripts",
      recommended: true,
      verdict: "carried",
    },
    {
      label: "Plain settings file",
      note: "Sampler settings only",
      verdict: "reduced",
      drops: ["all 28 prompt fragments", "the variables", "the two scripts"],
    },
  ],
  blocks: [
    {
      id: "p1",
      definition: "preset_core",
      title: "Prompt fragments",
      layout: "single",
      width: "full",
      elements: [
        {
          id: "p1e1",
          type: "prompt_list",
          label: "Fragments",
          role: "prompt_fragments",
          total: 28,
          fragments: [
            {
              id: "f1",
              name: "Main",
              role: "system",
              placement: "relative",
              enabled: true,
              group: "Core",
              body: "Write in past tense. Keep replies under 120 words unless the scene needs more.",
            },
            {
              id: "f2",
              name: "Restraint",
              role: "system",
              placement: "absolute",
              depth: 4,
              enabled: true,
              group: "Core",
              body: "Do not narrate what {{user}} feels or decides. Let silences stand.",
            },
            {
              id: "f3",
              name: "Scene anchor",
              role: "user",
              placement: "absolute",
              depth: 1,
              enabled: true,
              group: "Scene",
              body: "[The scene so far: {{scene}}]",
            },
            {
              id: "f4",
              name: "Jailbreak",
              role: "system",
              placement: "relative",
              enabled: false,
              group: "Off",
              body: "Unused. Kept for reference.",
            },
          ],
        },
      ],
    },
    {
      id: "p2",
      definition: "settings",
      title: "Settings",
      layout: "trio",
      width: "full",
      elements: [
        {
          id: "p2e1",
          type: "setting_group",
          label: "Sampler",
          role: "sampler_settings",
          settings: [
            { name: "temperature", value: "0.82" },
            { name: "top_p", value: "0.9" },
            { name: "top_k", value: "40" },
            { name: "repetition_penalty", value: "1.05" },
          ],
        },
        {
          id: "p2e2",
          type: "setting_group",
          label: "Completion",
          role: "completion_settings",
          settings: [
            { name: "max_tokens", value: "320" },
            { name: "stop", value: "\\n{{user}}:" },
            { name: "stream", value: "true" },
          ],
        },
        {
          id: "p2e3",
          type: "setting_group",
          label: "Advanced",
          role: "advanced_settings",
          settings: [
            { name: "context_size", value: "16384" },
            { name: "trim_incomplete", value: "true" },
          ],
        },
      ],
    },
    {
      id: "p3",
      definition: "variables",
      title: "Variables",
      layout: "single",
      width: "two_thirds",
      elements: [
        {
          id: "p3e1",
          type: "variable_schema",
          label: "Schema",
          role: "prompt_variables",
          variables: [
            {
              id: "v1",
              name: "scene",
              type: "string",
              fallback: "an empty room",
              note: "Written into the scene anchor at depth 1",
            },
            {
              id: "v2",
              name: "pace",
              type: "choice",
              fallback: "slow",
              choices: ["slow", "even", "quick"],
              note: "Selects one of three restraint fragments",
            },
            {
              id: "v3",
              name: "narrate_weather",
              type: "boolean",
              fallback: "false",
              note: "Adds one sentence of weather to each scene change",
            },
          ],
        },
      ],
    },
    {
      id: "p4",
      definition: "scripts",
      title: "Scripts",
      layout: "single",
      width: "third",
      elements: [
        {
          id: "p4e1",
          type: "script_list",
          label: "Regex",
          role: "regex_scripts",
          scripts: [
            {
              id: "r1",
              name: "Strip stage directions",
              find: "\\*[^*]+\\*",
              replace: "",
              targets: ["assistant"],
              enabled: true,
            },
            {
              id: "r2",
              name: "Collapse ellipses",
              find: "\\.{4,}",
              replace: "...",
              targets: ["assistant", "user"],
              enabled: true,
            },
          ],
        },
      ],
    },
  ],
  releases: [
    {
      version: "main",
      date: "2 June 2026",
      summary: "Restraint fragment moved to depth 4.",
      changes: ["Moved Restraint to absolute depth 4", "Disabled Jailbreak"],
      additions: 1,
      removals: 1,
    },
  ],
};

export const THEME_ASSET: Asset = {
  id: "a5",
  kind: "theme",
  name: "Verdant record",
  assetVersion: "1.1",
  creditedAuthor: "sy",
  blurb: "A green reading theme for long sessions and dim rooms.",
  creator: "@sy",
  shared: "18 August 2026",
  tags: ["dark", "green", "reading"],
  cover: verdant,
  tint: TINTS.verdant,
  formats: [
    {
      label: "Lumiverse theme",
      note: "Carries the palette, the controls and the stylesheet",
      recommended: true,
      verdict: "carried",
    },
    {
      label: "Palette only",
      note: "Colour tokens as JSON",
      verdict: "reduced",
      drops: ["the 16 KB stylesheet", "the seven controls"],
    },
  ],
  blocks: [
    {
      id: "t1",
      definition: "theme_core",
      title: "Palette",
      layout: "duo",
      width: "full",
      elements: [
        {
          id: "t1e1",
          type: "color_set",
          label: "Tokens",
          role: "theme_tokens",
          tokens: [
            { name: "background", value: "#0f1512" },
            { name: "surface", value: "#16201b" },
            { name: "raised", value: "#1e2b24" },
            { name: "text", value: "#e4ece6" },
            { name: "muted", value: "#8fa79a" },
            { name: "accent", value: "#5fb894" },
            { name: "accent-soft", value: "#2c5c48" },
            { name: "warning", value: "#d8a04a" },
            { name: "danger", value: "#c9585c" },
            { name: "border", value: "#2a3a32" },
          ],
        },
        {
          id: "t1e2",
          type: "setting_group",
          label: "Controls",
          role: "theme_controls",
          settings: [
            { name: "Message radius", value: "10px" },
            { name: "Avatar shape", value: "Rounded square" },
            { name: "Chat width", value: "72ch" },
            { name: "Font", value: "Source Serif" },
            { name: "Blur behind panels", value: "On" },
            { name: "Compact timestamps", value: "On" },
            { name: "Animated background", value: "Off" },
          ],
        },
      ],
    },
    {
      id: "t2",
      definition: "stylesheet",
      title: "Stylesheet",
      layout: "single",
      width: "full",
      elements: [
        {
          id: "t2e1",
          type: "stylesheet_set",
          label: "Sheets",
          role: "stylesheets",
          sheets: [
            {
              id: "sh1",
              name: "theme.css",
              size: "14.2 KB",
              lines: [
                ":root {",
                "  --chat-bg: #0f1512;",
                "  --chat-surface: #16201b;",
                "  --chat-accent: #5fb894;",
                "}",
                "",
                ".message { border-radius: 10px; }",
              ],
            },
            {
              id: "sh2",
              name: "fonts.css",
              size: "1.8 KB",
              lines: [
                "@font-face {",
                '  font-family: "Source Serif";',
                "  font-display: swap;",
                "}",
              ],
            },
          ],
        },
      ],
    },
  ],
  releases: [
    {
      version: "1.1",
      date: "18 August 2026",
      summary: "Accent lightened, border contrast raised.",
      changes: ["Changed accent", "Changed border"],
      additions: 2,
      removals: 0,
    },
  ],
};

export const PACK_ASSET: Asset = {
  id: "a6",
  kind: "pack",
  name: "The record office",
  assetVersion: "5",
  creditedAuthor: "Wren Ashdown",
  blurb: "Marisol and the four people she shares a building with.",
  creator: "@wren",
  shared: "1 September 2026",
  tags: ["ensemble", "verrin", "group chat"],
  cover: ember,
  tint: TINTS.ember,
  formats: [
    {
      label: "Lumia pack",
      note: "All five members with their avatars",
      recommended: true,
      verdict: "carried",
    },
    {
      label: "Five character cards",
      note: "One file per member",
      verdict: "reduced",
      drops: ["the pack ordering", "the shared author note"],
    },
  ],
  blocks: [
    {
      id: "k1",
      definition: "pack_core",
      title: "Members",
      layout: "single",
      width: "full",
      elements: [
        {
          id: "k1e1",
          type: "record_list",
          label: "Members",
          role: "pack_items",
          schema: "lumia",
          records: [
            {
              id: "m1",
              lumiaName: "Marisol",
              lumiaDefinition:
                "Night-desk archivist. Eleven years on the same twelve hours.",
              lumiaPersonality: "Dry, patient, unhurried.",
              lumiaBehavior: "Answers late and exactly.",
              avatar: nightDesk,
              genderIdentity: "she / her",
              authorName: "Wren Ashdown",
              version: "1.4",
            },
            {
              id: "m2",
              lumiaName: "Aldon Reeve",
              lumiaDefinition:
                "An inspector who came for three days and stayed eleven.",
              lumiaPersonality: "Genial, immovable.",
              lumiaBehavior: "Asks the same question four ways.",
              avatar: lastLight,
              genderIdentity: "he / him",
              authorName: "Wren Ashdown",
              version: "1.0",
            },
            {
              id: "m3",
              lumiaName: "Tobi",
              lumiaDefinition: "Day desk. Twenty-two. Wants the night shift.",
              lumiaPersonality: "Fast, cheerful, careless.",
              lumiaBehavior: "Files things where he can reach them.",
              avatar: verdant,
              genderIdentity: "they / them",
              authorName: "Wren Ashdown",
              version: "1.0",
            },
            {
              id: "m4",
              lumiaName: "Mrs Odell",
              lumiaDefinition: "Building manager. Has the only lift key.",
              lumiaPersonality: "Brisk. Unbribable.",
              lumiaBehavior: "Says no first and reconsiders in writing.",
              avatar: atlas,
              genderIdentity: "she / her",
              authorName: "Wren Ashdown",
              version: "1.1",
            },
            {
              id: "m5",
              lumiaName: "The fourth floor",
              lumiaDefinition:
                "Not a person. Included because the group chat needs it.",
              lumiaPersonality: "Silent.",
              lumiaBehavior: "Answers once, at the end.",
              avatar: chorus,
              genderIdentity: "—",
              authorName: "Wren Ashdown",
              version: "0.3",
            },
          ],
        },
      ],
    },
    {
      id: "k2",
      definition: "author_notes",
      title: "Author's notes",
      layout: "single",
      width: "third",
      elements: [
        {
          id: "k2e1",
          type: "prose",
          label: "On running the group",
          role: "creator_notes",
          display: "rich",
          body: "Four is the right number in a scene. The fourth floor goes last or not at all.",
        },
      ],
    },
  ],
  releases: [
    {
      version: "5",
      date: "1 September 2026",
      summary: "Mrs Odell added, avatars redrawn.",
      changes: ["Added Mrs Odell", "Replaced five avatars"],
      additions: 6,
      removals: 0,
    },
  ],
};

export const ASSETS: Record<string, Asset> = {
  character: CHARACTER_ASSET,
  lorebook: LOREBOOK_ASSET,
  preset: PRESET_ASSET,
  theme: THEME_ASSET,
  pack: PACK_ASSET,
  sparse: SPARSE_ASSET,
};
