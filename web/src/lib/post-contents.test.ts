import { expect, test } from "bun:test";
import {
  POST_BODY_VERSION,
  type PostBlock,
  type PostBody,
} from "@/lib/post-body";
import { postContents } from "@/lib/post-contents";

function heading(
  level: number,
  text: string,
  anchor?: string,
): Extract<PostBlock, { type: "heading" }> {
  return { type: "heading", level, anchor, content: [{ type: "text", text }] };
}

function document(...content: PostBlock[]): PostBody {
  return { version: POST_BODY_VERSION, content };
}

test("three headings derive the contents in reading order", () => {
  const contents = postContents(
    document(
      heading(2, "What is stored", "what-is-stored"),
      { type: "paragraph", content: [{ type: "text", text: "Prose." }] },
      heading(2, "What is refused", "what-is-refused"),
      heading(3, "Unknown nodes", "unknown-nodes"),
    ),
  );
  expect(contents).toEqual([
    { anchor: "what-is-stored", label: "What is stored", level: 2 },
    { anchor: "what-is-refused", label: "What is refused", level: 2 },
    { anchor: "unknown-nodes", label: "Unknown nodes", level: 3 },
  ]);
});

test("a shorter structure derives no contents at all", () => {
  const contents = postContents(
    document(
      heading(2, "What is stored", "what-is-stored"),
      heading(2, "What is refused", "what-is-refused"),
    ),
  );
  expect(contents).toEqual([]);
});

test("a heading label joins every span it is written from", () => {
  const contents = postContents(
    document(
      {
        type: "heading",
        level: 2,
        anchor: "one",
        content: [
          { type: "text", text: "Reading " },
          { type: "text", text: "and", marks: [{ type: "italic" }] },
          { type: "text", text: " sharing" },
        ],
      },
      heading(2, "Two", "two"),
      heading(2, "Three", "three"),
    ),
  );
  expect(contents[0]).toEqual({
    anchor: "one",
    label: "Reading and sharing",
    level: 2,
  });
});

test("a fourth-level heading is too deep to reach the contents", () => {
  const contents = postContents(
    document(
      heading(2, "One", "one"),
      heading(2, "Two", "two"),
      heading(3, "Three", "three"),
      heading(4, "Four", "four"),
    ),
  );
  expect(contents.map((entry) => entry.anchor)).toEqual([
    "one",
    "two",
    "three",
  ]);
});

test("a heading kept from before Illarin stored addresses is not linkable", () => {
  const contents = postContents(
    document(
      heading(2, "One", "one"),
      heading(2, "Two"),
      heading(2, "Three", "three"),
      heading(2, "Four", "four"),
    ),
  );
  expect(contents.map((entry) => entry.anchor)).toEqual([
    "one",
    "three",
    "four",
  ]);
});

test("three headings without addresses derive nothing rather than a dead list", () => {
  const contents = postContents(
    document(heading(2, "One"), heading(2, "Two"), heading(2, "Three")),
  );
  expect(contents).toEqual([]);
});

test("an empty heading is not a place a reader can be sent", () => {
  const contents = postContents(
    document(
      heading(2, "One", "one"),
      heading(2, "   ", "blank"),
      heading(2, "Two", "two"),
      heading(2, "Three", "three"),
    ),
  );
  expect(contents.map((entry) => entry.anchor)).toEqual([
    "one",
    "two",
    "three",
  ]);
});

test("a heading nested inside a quote or a callout is not article structure", () => {
  const contents = postContents(
    document(
      heading(2, "One", "one"),
      heading(2, "Two", "two"),
      {
        type: "callout",
        kind: "note",
        content: [heading(2, "Aside", "aside")],
      },
      {
        type: "quote",
        content: [heading(2, "Quoted", "quoted")],
      },
    ),
  );
  expect(contents).toEqual([]);
});

test("a document with no headings derives nothing", () => {
  expect(postContents(document())).toEqual([]);
});
