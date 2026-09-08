import type { BlockLayout, BlockWidth } from "@/lib/page-arrangement";

export type Item = {
  id: string;
  name: string;
  text: string;
  keys?: string;
  enabled?: boolean;
};
export type Element = {
  id: string;
  label: string;
  type: "prose" | "text_set" | "entry_table" | "dialogue_sample";
  role?: string;
  text: string;
  items: Item[];
};
export type Block = {
  id: string;
  title: string;
  definition: string;
  width: BlockWidth;
  layout: BlockLayout;
  hidden: boolean;
  elements: Element[];
};
export type Asset = {
  name: string;
  blurb: string;
  kind: "character" | "lorebook";
  version: string;
  nsfw: "no" | "yes" | "unanswered";
  blocks: Block[];
};
export type Notes = { summary: string; notes: string };
export type Change = {
  id: string;
  label: string;
  before: string;
  after: string;
  group: string;
};

function prose(
  id: string,
  label: string,
  text: string,
  role?: string,
): Element {
  return { id, label, text, role, type: "prose", items: [] };
}
function block(
  id: string,
  title: string,
  definition: string,
  elements: Element[],
  width: BlockWidth = "full",
  layout: BlockLayout = "single",
): Block {
  return { id, title, definition, elements, width, layout, hidden: false };
}

const description = `Morrow keeps the last lighthouse on a sea that has forgotten its tides. Each evening, they wind the lens by hand, write the weather in a book no one collects, and wait for a ship that has been missing for eleven years.

They are patient with people and impatient with machines. They mend things while they talk: a loose hinge, a split sleeve, a conversation that has gone quietly wrong. Ask about the missing ship and they will offer you tea. Ask twice and they will tell you the truth, though never all of it at once.

The lighthouse is a home before it is a mystery. There are damp coats beside the stove, jars of sea glass on the windowsill, and a third cup that Morrow sets out without thinking. Let ordinary moments have room. The strange things arrive on their own.

Morrow speaks plainly, with an occasional dry observation. They notice what a visitor carries, what they leave unsaid, and whether they look toward the water. They never decide what the visitor thinks or does. A scene should leave the visitor a real choice.`;

export function characterFixture(): Asset {
  return {
    name: "Morrow, keeper of the last light",
    blurb:
      "An impossible coastline. A patient keeper. A light left on for someone who may never return.",
    kind: "character",
    version: "1.0",
    nsfw: "no",
    blocks: [
      block(
        "core",
        "The character",
        "character_core",
        [
          prose("description", "Description", description, "description"),
          prose(
            "personality",
            "Personality",
            "Patient, practical, quietly stubborn. Offers hospitality before explanations.",
            "personality",
          ),
          prose(
            "scenario",
            "Scenario",
            "You arrive at the lighthouse at dusk, carrying a letter addressed to its former keeper.",
            "scenario",
          ),
        ],
        "two_thirds",
        "stack-3",
      ),
      block(
        "usage",
        "Before you arrive",
        "usage",
        [
          prose(
            "usage-text",
            "Usage notes",
            "Begin with any greeting. Slow scenes and small discoveries suit this character. Leave room for the visitor to choose their own history.",
          ),
        ],
        "third",
      ),
      block(
        "messages",
        "Ways to begin",
        "messages",
        [
          {
            id: "greetings",
            label: "Greetings",
            type: "text_set",
            role: "greetings",
            text: "",
            items: [
              {
                id: "arrival",
                name: "A letter at dusk",
                text: "The door opens before you knock. Morrow looks at the letter in your hand, then out at the empty road.\n\n‘I wondered when that would find its way here.’ They step aside. ‘Come in. You look like someone who has walked past the last sensible place to stop.’",
              },
              {
                id: "storm",
                name: "Shelter from the storm",
                text: "A lamp moves behind the rain-streaked window. A moment later, a voice calls from the doorway.\n\n‘The path floods in another minute. Whatever brought you here can wait until you are dry.’",
              },
              {
                id: "morning",
                name: "The morning after",
                text: "The kettle is already singing. Morrow has spread a chart across the table and anchored its corners with four mismatched cups.\n\n‘Tell me,’ they say, without looking up, ‘did you hear bells last night?’",
              },
            ],
          },
          {
            id: "dialogue",
            label: "Example dialogue",
            type: "dialogue_sample",
            role: "example_dialogue",
            text: "",
            items: [
              {
                id: "turn-one",
                name: "Visitor",
                text: "Does anyone still need the light?",
              },
              {
                id: "turn-two",
                name: "Morrow",
                text: "I would rather keep it lit and be wrong.",
              },
            ],
          },
          {
            id: "group-greetings",
            label: "Group greetings",
            type: "text_set",
            role: "group_greetings",
            text: "",
            items: [
              {
                id: "group-arrival",
                name: "Visitors together",
                text: "Morrow counts the figures on the path and reaches for more cups. ‘There is room by the stove for all of you.’",
              },
            ],
          },
        ],
        "full",
        "stack-3",
      ),
      block(
        "notes",
        "From the creator",
        "author_notes",
        [
          prose(
            "creator-note",
            "Creator notes",
            "A synthetic character made for trying the workspace. Every place, person and line of dialogue here is fictional.",
          ),
        ],
        "half",
      ),
      block(
        "empty",
        "Room for more",
        "custom_block",
        [prose("empty-prose", "Prose", "")],
        "half",
      ),
    ],
  };
}

export function lorebookFixture(): Asset {
  const places = [
    "The glass harbour",
    "The bell orchard",
    "The drowned observatory",
    "The salt road",
    "The lantern market",
    "The winter ferry",
    "The sleeping archive",
    "The tide garden",
  ];
  const entries = Array.from({ length: 64 }, (_, index) => ({
    id: `place-${index + 1}`,
    name: `${places[index % places.length]} · ${Math.floor(index / places.length) + 1}`,
    keys: `${places[index % places.length].slice(4).toLowerCase()}, coast ${index + 1}`,
    enabled: true,
    text: `Along the coast, ${places[index % places.length].toLowerCase()} is known for a custom: visitors leave a small object before asking for directions. The objects are not payment. They are how the residents remember a stranger.\n\nIn this part of the world, maps show promises rather than distances. A journey can shorten after an honest conversation, or grow longer when someone refuses to say goodbye. Introduce this detail through what a traveller sees and hears. Leave its explanation open.\n\nThis is fictional location ${index + 1}, written only for the workspace prototype.`,
  }));
  return {
    name: "An atlas of places between",
    blurb:
      "A coastal setting for stories about arrivals, departures, and everything that waits between them.",
    kind: "lorebook",
    version: "First edition",
    nsfw: "no",
    blocks: [
      block("atlas", "The coastal atlas", "lorebook_core", [
        {
          id: "entries",
          label: "Lorebook entries",
          type: "entry_table",
          role: "lorebook_entries",
          text: "",
          items: entries,
        },
      ]),
      block(
        "usage",
        "Using the atlas",
        "usage",
        [
          prose(
            "usage-text",
            "Usage notes",
            "Use entry keys to bring each place into a scene. All 64 entries are synthetic and can be edited freely.",
          ),
        ],
        "half",
      ),
      block(
        "notes",
        "From the creator",
        "author_notes",
        [
          prose(
            "creator-note",
            "Creator notes",
            "This collection tests browsing and writing with a substantial number of entries.",
          ),
        ],
        "half",
      ),
    ],
  };
}

export function changesBetween(before: Asset, after: Asset): Change[] {
  const changes: Change[] = [];
  function compare(
    id: string,
    label: string,
    a: string,
    b: string,
    group: string,
  ) {
    if (a !== b) changes.push({ id, label, before: a, after: b, group });
  }
  for (const key of ["name", "blurb", "version", "nsfw"] as const) {
    compare(
      key,
      {
        name: "Name",
        blurb: "Blurb",
        version: "Version label",
        nsfw: "Adult content",
      }[key],
      before[key],
      after[key],
      "Asset details",
    );
  }
  compare(
    "order",
    "Block order",
    before.blocks.map((b) => b.title).join(" → "),
    after.blocks.map((b) => b.title).join(" → "),
    "Page arrangement",
  );
  for (const current of after.blocks) {
    const previous = before.blocks.find((b) => b.id === current.id);
    if (!previous) continue;
    for (const key of ["title", "width", "layout", "hidden"] as const) {
      compare(
        `${current.id}-${key}`,
        `${current.title} · ${key}`,
        String(previous[key]),
        String(current[key]),
        "Page arrangement",
      );
    }
    const elementIds = new Set(
      [...previous.elements, ...current.elements].map((e) => e.id),
    );
    for (const elementId of elementIds) {
      const element = current.elements.find((e) => e.id === elementId);
      const old = previous.elements.find((e) => e.id === elementId);
      const label = element?.label ?? old?.label ?? "Element";
      if (element?.type === "prose" || old?.type === "prose")
        compare(
          elementId,
          label,
          old?.text ?? "",
          element?.text ?? "",
          current.title,
        );
      const ids = new Set(
        [...(old?.items ?? []), ...(element?.items ?? [])].map(
          (item) => item.id,
        ),
      );
      for (const id of ids) {
        const a = old?.items.find((item) => item.id === id);
        const b = element?.items.find((item) => item.id === id);
        const describe = (item?: Item) =>
          item
            ? [
                item.name,
                item.keys,
                item.enabled === undefined
                  ? ""
                  : item.enabled
                    ? "Enabled"
                    : "Disabled",
                item.text,
              ]
                .filter(Boolean)
                .join("\n")
            : "";
        compare(
          `${elementId}-${id}`,
          b?.name || a?.name || "Untitled item",
          describe(a),
          describe(b),
          label,
        );
      }
    }
  }
  return changes;
}

export function publicationIssues(
  asset: Asset,
  notes: Notes,
  changes: Change[],
): string[] {
  const issues: string[] = [];
  if (!asset.name.trim()) issues.push("Add an asset name in Asset details.");
  if (asset.nsfw === "unanswered")
    issues.push("Answer the adult-content question in Asset details.");
  if (!notes.summary.trim()) issues.push("Write a short update summary.");
  if (!changes.length)
    issues.push(
      "Make a content, detail or arrangement change before publishing an update.",
    );
  const elements = asset.blocks.flatMap((b) => b.elements);
  if (asset.kind === "character") {
    if (!elements.find((e) => e.role === "description")?.text.trim())
      issues.push("Add a description in The character.");
    if (
      !elements
        .find((e) => e.role === "greetings")
        ?.items.some((item) => item.text.trim())
    )
      issues.push("Write at least one greeting in Ways to begin.");
  } else if (
    !elements
      .find((e) => e.role === "lorebook_entries")
      ?.items.some((item) => item.text.trim())
  )
    issues.push("Write at least one lorebook entry in The coastal atlas.");
  return issues;
}

export const blockRules: Record<
  string,
  { required: boolean; hideable: boolean; layouts: BlockLayout[] }
> = {
  character_core: {
    required: true,
    hideable: true,
    layouts: ["stack-3", "trio"],
  },
  messages: {
    required: true,
    hideable: false,
    layouts: ["stack-2", "stack-3"],
  },
  lorebook_core: { required: true, hideable: false, layouts: ["single"] },
  usage: { required: false, hideable: true, layouts: ["single"] },
  author_notes: { required: false, hideable: true, layouts: ["single"] },
  custom_block: {
    required: false,
    hideable: true,
    layouts: ["single", "duo", "main-aside", "trio", "stack-2", "stack-3"],
  },
};

export function changedElementIds(before: Asset, after: Asset): Set<string> {
  const ids = new Set<string>();
  for (const key of ["name", "blurb", "version", "nsfw"] as const)
    if (before[key] !== after[key]) ids.add(key);
  for (const block of after.blocks) {
    const previous = before.blocks.find((b) => b.id === block.id);
    for (const element of block.elements) {
      const old = previous?.elements.find((e) => e.id === element.id);
      if (JSON.stringify(old) !== JSON.stringify(element)) ids.add(element.id);
    }
  }
  return ids;
}
