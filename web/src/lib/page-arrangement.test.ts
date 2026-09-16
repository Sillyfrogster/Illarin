import { describe, expect, test } from "bun:test";
import {
  contentItemCount,
  EXCERPT_DEFINITIONS,
  editsInTheRail,
  elementTracks,
  LAYOUTS,
  layoutChoiceIssue,
  NARROW_BLOCK_GRID_PX,
  packBlockRows,
  placeBlocks,
  suggestedBlockWidth,
  suggestionCandidateWidths,
  WIDTH_COLUMNS,
  WIDTH_FLOORS_PX,
  widthChoiceIssue,
  writesInPlace,
} from "./page-arrangement";

type TestBlock = {
  id: string;
  width: keyof typeof WIDTH_COLUMNS;
  hidden: boolean;
  empty: boolean;
};

function block(
  id: string,
  width: TestBlock["width"],
  hidden = false,
  empty = false,
): TestBlock {
  return { id, width, hidden, empty };
}

function spans(rows: ReturnType<typeof packBlockRows<TestBlock>>) {
  return rows.map((row) => row.map((item) => [item.block.id, item.columns]));
}

function placements(rows: ReturnType<typeof packBlockRows<TestBlock>>) {
  return rows.map((row) =>
    row.map((item) => [item.block.id, item.columns, item.startColumn]),
  );
}

describe("page arrangement", () => {
  test("the four named widths are twelve, eight, six and four columns", () => {
    expect(WIDTH_COLUMNS).toEqual({
      full: 12,
      two_thirds: 8,
      half: 6,
      third: 4,
    });
  });

  test("the six layouts have finite named slots and minimum widths", () => {
    expect(LAYOUTS).toEqual({
      single: { slots: ["main"], minimumColumns: 4 },
      duo: { slots: ["left", "right"], minimumColumns: 8 },
      "main-aside": { slots: ["main", "aside"], minimumColumns: 8 },
      trio: { slots: ["left", "middle", "right"], minimumColumns: 12 },
      "stack-2": { slots: ["top", "bottom"], minimumColumns: 4 },
      "stack-3": { slots: ["top", "middle", "bottom"], minimumColumns: 4 },
    });
  });

  test("blocks stay in page order and start a new row when they do not fit", () => {
    expect(
      spans(
        packBlockRows(
          [block("a", "two_thirds"), block("b", "half"), block("c", "third")],
          {},
        ),
      ),
    ).toEqual([
      [["a", 8]],
      [
        ["b", 6],
        ["c", 4],
      ],
    ]);
  });

  test("a block keeps its declared width and a short row stays short", () => {
    expect(
      spans(
        packBlockRows(
          [block("a", "third"), block("b", "half"), block("c", "third")],
          {},
        ),
      ),
    ).toEqual([
      [
        ["a", 4],
        ["b", 6],
      ],
      [["c", 4]],
    ]);
  });

  test("hidden and empty blocks keep their grid places when not rendered", () => {
    const blocks = [block("hidden", "half", true), block("shown", "half")];

    expect(placements(packBlockRows(blocks, {}))).toEqual([
      [
        ["hidden", 6, 1],
        ["shown", 6, 7],
      ],
    ]);

    const withEmpty = [
      block("first", "third"),
      block("empty", "half", false, true),
      block("last", "third"),
    ];
    expect(placements(packBlockRows(withEmpty, {}))).toEqual([
      [
        ["first", 4, 1],
        ["empty", 6, 5],
      ],
      [["last", 4, 1]],
    ]);
  });

  test("placement gives every block its grid row and its place in that row", () => {
    const blocks = [
      block("a", "half"),
      block("b", "third"),
      block("c", "two_thirds"),
      block("d", "full"),
    ];

    expect(
      placeBlocks(blocks, {}).map((item) => [
        item.block.id,
        item.row,
        item.place,
        item.startColumn,
        item.columns,
      ]),
    ).toEqual([
      ["a", 1, 0, 1, 6],
      ["b", 1, 1, 7, 4],
      ["c", 2, 0, 1, 8],
      ["d", 3, 0, 1, 12],
    ]);
  });

  test("every width combination keeps each block at its declared size", () => {
    const widths = Object.keys(WIDTH_COLUMNS) as TestBlock["width"][];
    for (const first of widths) {
      for (const second of widths) {
        for (const third of widths) {
          const chosen = [first, second, third];
          const rows = packBlockRows(
            [block("a", first), block("b", second), block("c", third)],
            {},
          );
          const packed = rows.flat();
          expect(packed).toHaveLength(3);
          packed.forEach((item, index) => {
            expect(item.columns).toBe(WIDTH_COLUMNS[chosen[index]]);
          });
          for (const row of rows) {
            const occupied = row.reduce(
              (total, item) => total + item.columns,
              0,
            );
            expect(occupied).toBeLessThanOrEqual(12);
          }
        }
      }
    }
  });

  test("each width promotes at its pixel floor through the same ladder", () => {
    expect(WIDTH_FLOORS_PX).toEqual({
      full: 280,
      two_thirds: 640,
      half: 440,
      third: 320,
    });

    const columnsAt = (width: TestBlock["width"], availableWidth: number) =>
      packBlockRows([block("one", width)], {
        availableWidth,
      })[0][0].columns;

    expect(columnsAt("third", 1000)).toBe(4);
    expect(columnsAt("third", 999)).toBe(6);
    expect(columnsAt("third", 899)).toBe(12);
    expect(columnsAt("half", 900)).toBe(6);
    expect(columnsAt("half", 899)).toBe(12);
    expect(columnsAt("two_thirds", 970)).toBe(8);
    expect(columnsAt("two_thirds", 969)).toBe(12);
    expect(columnsAt("full", 280)).toBe(12);
  });

  test("block packing never owns the measure of prose inside it", () => {
    const widths = ["full", "two_thirds", "half", "third"] as const;

    for (const width of widths) {
      expect(packBlockRows([block(width, width)], {})[0][0]).toEqual({
        block: block(width, width),
        columns: WIDTH_COLUMNS[width],
        startColumn: 1,
      });
    }
  });

  test("narrow packing ignores declared widths", () => {
    expect(
      spans(
        packBlockRows([block("a", "third"), block("b", "half")], {
          availableWidth: NARROW_BLOCK_GRID_PX,
        }),
      ),
    ).toEqual([[["a", 12]], [["b", 12]]]);
  });

  test("layout and width refusals name the first fix", () => {
    expect(
      layoutChoiceIssue("trio", "half", [
        "Description",
        "Personality",
        "Scenario",
      ]),
    ).toBe("Trio needs Full width. Widen this block first.");
    expect(widthChoiceIssue("trio", "half")).toBe(
      "Trio needs Full width. Choose another layout before narrowing this block.",
    );
    expect(
      layoutChoiceIssue("stack-2", "full", [
        "Greetings",
        "Example dialogue",
        "Group-only greetings",
      ]),
    ).toBe(
      "Stack 2 has no room for Group-only greetings. Move or remove it first.",
    );
  });

  test("suggests the smallest size that keeps the rendered content comfortable", () => {
    expect(
      suggestedBlockWidth({
        width: "full",
        layout: "single",
        availableWidth: 1240,
        renderedHeights: {
          third: 700,
          half: 520,
          two_thirds: 360,
          full: 250,
        },
      }),
    ).toBe("half");
    expect(
      suggestedBlockWidth({
        width: "third",
        layout: "single",
        availableWidth: 1240,
        renderedHeights: {
          third: 1100,
          half: 680,
          two_thirds: 490,
          full: 370,
        },
      }),
    ).toBe("two_thirds");
    expect(
      suggestedBlockWidth({
        width: "half",
        layout: "single",
        availableWidth: 1240,
        renderedHeights: {
          third: 650,
          half: 400,
          two_thirds: 300,
          full: 220,
        },
      }),
    ).toBeNull();
  });

  test("a suggestion respects layout minimums and the current rendering", () => {
    expect(
      suggestedBlockWidth({
        width: "full",
        layout: "duo",
        availableWidth: 1240,
        renderedHeights: {
          third: 100,
          half: 100,
          two_thirds: 180,
          full: 120,
        },
      }),
    ).toBe("two_thirds");
    expect(
      suggestedBlockWidth({
        width: "full",
        layout: "single",
        availableWidth: NARROW_BLOCK_GRID_PX,
        renderedHeights: { full: 180 },
      }),
    ).toBeNull();
    expect(suggestionCandidateWidths("single", 900)).toEqual([
      { width: "half", renderedWidth: 440 },
      { width: "full", renderedWidth: 900 },
    ]);
  });
});

describe("remove confirmation counts", () => {
  test("counts each element through its public content shape", () => {
    expect(
      contentItemCount({ type: "prose", content: { text: "One body" } }),
    ).toBe(1);
    expect(
      contentItemCount({
        type: "text_set",
        content: { texts: [{ text: "First" }, { text: "Second" }] },
      }),
    ).toBe(2);
    expect(
      contentItemCount({
        type: "dialogue_sample",
        content: { turns: [{ speaker: "A", text: "Hello" }] },
      }),
    ).toBe(1);
    expect(
      contentItemCount({
        type: "image_set",
        content: { images: [{ mediaId: "one" }, { mediaId: "two" }] },
      }),
    ).toBe(2);
  });

  test("empty content counts as nothing lost", () => {
    expect(contentItemCount({ type: "prose", content: { text: "" } })).toBe(0);
    expect(contentItemCount({ type: "text_set", content: { texts: [] } })).toBe(
      0,
    );
  });

  test("counts every catalog element through its named collection", () => {
    const examples = [
      ["field_list", { fields: [{}, {}] }, 2],
      ["entry_table", { entries: [{}, {}, {}] }, 3],
      ["link_list", { links: [{}] }, 1],
      ["prompt_list", { fragments: [{}, {}] }, 2],
      ["variable_schema", { variables: { name: {}, mood: {} } }, 2],
      ["setting_group", { settings: { temperature: 1 } }, 1],
      ["script_list", { scripts: [{}, {}] }, 2],
      [
        "color_set",
        {
          modes: [
            { name: "light", colors: [{ name: "ink", value: "#000" }] },
            { name: "dark", colors: [{ name: "ink", value: "#fff" }] },
          ],
        },
        2,
      ],
      [
        "stylesheet_set",
        { global: "body {}", stylesheets: { card: ".card {}" } },
        2,
      ],
      ["record_list", { records: [{}, {}, {}] }, 3],
    ] as const;
    for (const [type, content, count] of examples) {
      expect(contentItemCount({ type, content })).toBe(count);
    }
  });
});

describe("where an element is edited", () => {
  test("the page writes its own prose, greetings, turns, fields and links", () => {
    for (const type of [
      "prose",
      "text_set",
      "dialogue_sample",
      "field_list",
      "link_list",
    ]) {
      expect(writesInPlace(type)).toBe(true);
      expect(editsInTheRail(type)).toBe(false);
    }
  });

  test("everything else keeps the reader's rendering and opens in the rail", () => {
    for (const type of [
      "entry_table",
      "image_set",
      "prompt_list",
      "variable_schema",
      "setting_group",
      "script_list",
      "color_set",
      "stylesheet_set",
      "record_list",
    ]) {
      expect(editsInTheRail(type)).toBe(true);
      expect(writesInPlace(type)).toBe(false);
    }
  });

  test("every element type has exactly one of the two homes", () => {
    for (const type of Object.keys(EXCERPT_DEFINITIONS)) {
      expect(writesInPlace(type)).toBe(!editsInTheRail(type));
    }
  });
});

describe("the columns a block's elements arrange into", () => {
  test("a stack is one column whatever it holds", () => {
    expect(elementTracks("single", 1)).toBe("minmax(0, 1fr)");
    expect(elementTracks("stack-2", 2)).toBe("minmax(0, 1fr)");
    expect(elementTracks("stack-3", 3)).toBe("minmax(0, 1fr)");
  });

  test("a full row takes the columns its layout declares", () => {
    expect(elementTracks("duo", 2)).toBe("repeat(2, minmax(0, 1fr))");
    expect(elementTracks("trio", 3)).toBe("repeat(3, minmax(0, 1fr))");
    expect(elementTracks("main-aside", 2)).toBe(
      "minmax(0, 2fr) minmax(0, 1fr)",
    );
  });

  test("a row closes up around what it no longer renders", () => {
    expect(elementTracks("trio", 2)).toBe("repeat(2, minmax(0, 1fr))");
    expect(elementTracks("trio", 1)).toBe("minmax(0, 1fr)");
    expect(elementTracks("duo", 1)).toBe("minmax(0, 1fr)");
    expect(elementTracks("main-aside", 1)).toBe("minmax(0, 1fr)");
  });

  test("a block rendering nothing still declares one column", () => {
    expect(elementTracks("trio", 0)).toBe("minmax(0, 1fr)");
  });

  test("more elements than slots take the slots the layout has", () => {
    expect(elementTracks("duo", 5)).toBe("repeat(2, minmax(0, 1fr))");
  });
});
