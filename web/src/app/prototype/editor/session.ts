import {
  type Asset,
  characterFixture,
  lorebookFixture,
  type Notes,
} from "./data";

export type Pane = "update" | "replacement" | "access";

export type Session = {
  draft: Asset;
  saved: Asset;
  published: Asset;
  notes: Notes;
  savedNotes: Notes;
  publishedSummary: string;
  number: number;
  baseRevision: number;
  serverRevision: number;
  newer?: Asset;
  conflict: boolean;
  editing: boolean;
  cursor: string | null;
  pane?: Pane;
  reviewed?: { asset: Asset; notes: Notes; revision: number };
  replacement?: { asset: Asset; revision: number };
  listed: boolean;
};

export function newSession(asset: Asset): Session {
  return {
    draft: asset,
    saved: asset,
    published: asset,
    notes: { summary: "", notes: "" },
    savedNotes: { summary: "", notes: "" },
    publishedSummary: "Initial recorded version",
    number: 1,
    baseRevision: 1,
    serverRevision: 1,
    editing: false,
    cursor: null,
    listed: true,
    conflict: false,
  };
}

export function initialSessions() {
  return {
    character: newSession(characterFixture()),
    lorebook: newSession(lorebookFixture()),
  };
}

export function sampleReplacement(asset: Asset): Asset {
  const incoming =
    asset.kind === "character" ? characterFixture() : lorebookFixture();
  const supplied =
    asset.kind === "character"
      ? ["description", "personality", "scenario", "greetings", "dialogue"]
      : ["entries"];
  const elements = incoming.blocks.flatMap((b) => b.elements);
  return {
    ...asset,
    blocks: asset.blocks.map((block) => ({
      ...block,
      elements: block.elements.map((element) => {
        if (!supplied.includes(element.id)) return element;
        const source = elements.find((e) => e.id === element.id);
        if (!source) return element;
        if (element.id === "description")
          return {
            ...source,
            text: `${source.text}\n\nA second light has appeared beyond the harbour. Morrow has begun keeping two weather books.`,
          };
        if (element.id === "greetings")
          return {
            ...source,
            items: [
              source.items[0],
              source.items[2],
              {
                id: "replacement-greeting",
                name: "The other light",
                text: "Morrow hands you the telescope. ‘Before you tell me I need more sleep, look at the headland. Tell me you see it too.’",
              },
            ],
          };
        if (element.id === "entries")
          return {
            ...source,
            items: [
              ...source.items.slice(0, -1),
              {
                id: "new-place",
                name: "The other lighthouse",
                keys: "other lighthouse, second light",
                enabled: true,
                text: "Across the harbour stands a lighthouse absent from every chart. Its lamp turns against the wind.",
              },
            ],
          };
        return source;
      }),
    })),
  };
}
