# Markdown import

Write a post body in Markdown and import it into the private working copy.

## Import

`POST /api/v1/publication/posts/{id}/import`

Replaces the working copy's entire body with the imported Markdown, converted into a post document. The server stores that document, not a second Markdown copy. Importing does not publish. Title, summary, slug, category and release are untouched. Validation is the same as for [Save a post](/developers/publication/writing#save-a-post).

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `version` | integer | yes | The working-copy version you read. |
| `markdown` | string | yes | Max 400,000 bytes. Request body max 1 MiB. |

Returns `200` with `{ "post": Post, "warnings": [{ "line": integer, "message": string }] }`.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | Unsupported Markdown. `refusals` lists every offending line. The working copy is unchanged. |
| 409 | `stale_version` | `version` is old. |
| 413 | `invalid` | Body over 1 MiB. |

```http
POST /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/import
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d

{
  "version": 2,
  "markdown": "## What changed\n\nScenes now export as one file.\n\n![The new export dialog](media:d4e8f1a2-9b3c-4d5e-8a6f-2b1c3d4e5f60 \"Export in 4.2\")\n"
}
```

```json PostImportRefusal
{
  "error": "This Markdown contains unsupported content. Review the reported lines.",
  "code": "invalid",
  "field": "markdown",
  "refusals": [
    { "line": 1, "message": "The post title is the page heading, so a body heading starts at level 2." },
    { "line": 9, "message": "Illarin does not carry HTML." }
  ]
}
```

The response lists up to 40 unsupported lines. If there are more, a final entry says so.

## Supported syntax

The subset below supports ordinary Markdown plus tables, task lists, strikethrough and GitHub-style alerts. Syntax outside this list may be refused.

| Markdown | Block |
| --- | --- |
| Paragraph | `paragraph` |
| `##` to `####` | `heading` level 2–4. Anchor is generated. |
| `- item`, `1. item` | `bulletList`, `orderedList` |
| `- [x] item`, `- [ ] item` | `taskList` |
| `> text` | `quote` |
| `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]` | `callout` |
| Fenced or indented code | `codeBlock`. The fence language is the label. |
| Pipe table | `table`. First row is the heading row. |
| `---` | `divider` |
| `![alt](media:<id> "caption")` alone on a line | `image` |
| `**bold**`, `*italic*`, `~~strike~~`, `` `code` `` | marks |
| `[text](https://…)`, `<https://…>`, `<user@example.com>` | `link` mark |

Images reference a picture already [uploaded to the post](/developers/publication/writing#upload-a-picture) by `media:<id>`. `alt` is required. The title becomes the caption. There is no gallery syntax; consecutive images become separate `image` blocks.

## Warnings

The import succeeds and saves the converted body. Review the returned warnings and the saved post before publishing.

| Input | Result |
| --- | --- |
| Hard line break inside a paragraph | Replaced with a space. |
| Ordered list not starting at 1 | Starts at 1. |
| Code fence language not in the [supported list](/developers/publication/document#codeblock) | Labelled `plain`. |
| Link with a title | Title dropped. |
| Formatting inside image alt text | Flattened to plain text. |

## Refusals

The import fails. Nothing is saved.

| Input |
| --- |
| Raw HTML, inline or block |
| A paragraph starting with `import` or `export` (MDX) |
| Footnotes |
| `#` headings, and headings deeper than `####` |
| An image not referencing `media:<id>`, without alt text, or inline with other text |
| A link that is not `https://` or `mailto:` |
| A block in a disallowed position, such as a quote inside a list |
| A task list where only some items are checkboxes |
| An empty list item, quote, callout or code block |
| A table whose rows have different cell counts |
| Any other syntax |
