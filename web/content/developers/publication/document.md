# Post document

Build a post body from paragraphs, headings, lists, pictures and other supported blocks.

The `document` field in a save request uses this format. Its `version` describes the format, not the working-copy edit number. If you prefer to write Markdown, use [Markdown import](/developers/publication/markdown).

## Shape

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "paragraph",
      "content": [{ "type": "text", "text": "Scenes now export as one file." }]
    }
  ]
}
```

| Field | Type | Rule |
| --- | --- | --- |
| `version` | integer | Currently `2`. See [Versions](#versions). |
| `content` | block[] | Top-level blocks, in order. |

Rules:

- Every block is an object with a `type`. Only the keys listed for that type are allowed.
- Unknown types, unknown keys and blocks in a disallowed position are refused. The error `field` is a path such as `document.content.2.content.0.type`.
- No HTML, MDX, scripts, styles or external image URLs.

## Text runs

A text run is a piece of text with optional formatting, such as bold or a link. Paragraphs and headings contain runs; table cells contain paragraphs.

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Install with " },
        { "type": "text", "text": "lantern update", "marks": [{ "type": "code" }] },
        { "type": "text", "text": " or " },
        {
          "type": "text",
          "text": "download the build",
          "marks": [
            { "type": "bold" },
            { "type": "link", "href": "https://paperlantern.example/download" }
          ]
        },
        { "type": "text", "text": "." }
      ]
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `type` | string | Always `text`. |
| `text` | string | Valid UTF-8. Max 5,000 bytes per run. |
| `marks` | mark[] | Optional. Any combination of the marks below. |

| Mark | Keys | Rule |
| --- | --- | --- |
| `bold` | | |
| `italic` | | |
| `strike` | | |
| `code` | | |
| `link` | `href` | `https://` or `mailto:`. Max 2,000 characters. No spaces or control characters. |

## Blocks

### paragraph

| Key | Type | Rule |
| --- | --- | --- |
| `content` | run[] | Text runs. |

### heading

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "heading",
      "level": 2,
      "anchor": "what-changed",
      "content": [{ "type": "text", "text": "What changed" }]
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `level` | integer | 2, 3 or 4. The post title is level 1. |
| `anchor` | string | Optional. `a-z`, `0-9`, single hyphens, max 80. Unique within the post. Generated from the text when omitted. Used for `#anchor` links and the contents list. |
| `content` | run[] | Must contain text. |

Keep anchors stable after publication. Readers may hold links to them.

### bulletList, orderedList

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "orderedList",
      "content": [
        {
          "type": "listItem",
          "content": [
            { "type": "paragraph", "content": [{ "type": "text", "text": "Open the library." }] }
          ]
        },
        {
          "type": "listItem",
          "content": [
            { "type": "paragraph", "content": [{ "type": "text", "text": "Choose Export." }] }
          ]
        }
      ]
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `content` | listItem[] | At least one item. |

A `listItem` has `content`: one or more of `paragraph`, `bulletList`, `orderedList`, `taskList`, `codeBlock`. Ordered lists always start at 1.

### taskList

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "taskList",
      "content": [
        {
          "type": "taskItem",
          "done": true,
          "content": [{ "type": "paragraph", "content": [{ "type": "text", "text": "Scene export" }] }]
        },
        {
          "type": "taskItem",
          "done": false,
          "content": [{ "type": "paragraph", "content": [{ "type": "text", "text": "Scene import" }] }]
        }
      ]
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `content` | taskItem[] | At least one item. |

A `taskItem` has `done` (boolean) and `content` with the same allowed blocks as `listItem`. Readers cannot change `done`.

### quote

| Key | Type | Rule |
| --- | --- | --- |
| `content` | block[] | One or more of `paragraph`, `bulletList`, `orderedList`, `codeBlock`. |

### codeBlock

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "codeBlock",
      "language": "bash",
      "source": "lantern export --all ~/scenes"
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `language` | string | One of `plain`, `bash`, `css`, `diff`, `go`, `html`, `javascript`, `json`, `markdown`, `python`, `rust`, `sql`, `toml`, `typescript`, `yaml`. |
| `source` | string | Non-empty. Max 20,000 bytes. Only newline and tab as control characters. CRLF is normalized to LF. Trailing newlines are trimmed. |

Highlighting is applied at render time. The document stores plain source only.

### table

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "table",
      "content": [
        {
          "type": "tableRow",
          "content": [
            { "type": "tableCell", "heading": true, "content": [{ "type": "paragraph", "content": [{ "type": "text", "text": "Platform" }] }] },
            { "type": "tableCell", "heading": true, "content": [{ "type": "paragraph", "content": [{ "type": "text", "text": "Build" }] }] }
          ]
        },
        {
          "type": "tableRow",
          "content": [
            { "type": "tableCell", "content": [{ "type": "paragraph", "content": [{ "type": "text", "text": "Linux" }] }] },
            { "type": "tableCell", "content": [{ "type": "paragraph", "content": [{ "type": "text", "text": "4.2.0-linux.tar.gz" }] }] }
          ]
        }
      ]
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `content` | tableRow[] | 1–60 rows. Every row has the same number of cells. |

A `tableRow` has `content`: 1–10 `tableCell` blocks. A `tableCell` has optional `heading` (boolean) and `content`: `paragraph` blocks only. Heading cells must fill the whole first row, the whole first column, or both.

### callout

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "callout",
      "kind": "warning",
      "content": [
        { "type": "paragraph", "content": [{ "type": "text", "text": "Back up your library before upgrading." }] }
      ]
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `kind` | string | `note`, `tip`, `important` or `warning`. |
| `content` | block[] | One or more of `paragraph`, `bulletList`, `orderedList`, `taskList`, `codeBlock`. |

### image

```json PostDocument
{
  "version": 2,
  "content": [
    {
      "type": "image",
      "mediaId": "d4e8f1a2-9b3c-4d5e-8a6f-2b1c3d4e5f60",
      "alt": "The new export dialog with every scene selected",
      "caption": "Export in 4.2"
    }
  ]
}
```

| Key | Type | Rule |
| --- | --- | --- |
| `mediaId` | uuid | A picture [uploaded to this post](/developers/publication/writing#upload-a-picture) with purpose `document`. |
| `alt` | string | Required. Plain text, one line, max 300 characters. |
| `caption` | string | Optional. Plain text, one line, max 300 characters. |

### gallery

| Key | Type | Rule |
| --- | --- | --- |
| `content` | galleryImage[] | 1–12 items. |

A `galleryImage` has the same keys as `image`.

### divider

No keys other than `type`.

## Nesting

| Container | Allowed blocks |
| --- | --- |
| document | `paragraph`, `heading`, `bulletList`, `orderedList`, `taskList`, `quote`, `codeBlock`, `table`, `callout`, `image`, `gallery`, `divider` |
| `listItem`, `taskItem` | `paragraph`, `bulletList`, `orderedList`, `taskList`, `codeBlock` |
| `quote` | `paragraph`, `bulletList`, `orderedList`, `codeBlock` |
| `callout` | `paragraph`, `bulletList`, `orderedList`, `taskList`, `codeBlock` |
| `tableCell` | `paragraph` |

Maximum nesting depth is 8.

## Limits

| Limit | Value |
| --- | --- |
| Nodes per document (blocks, rows, cells, runs) | 4,000 |
| Nesting depth | 8 |
| Text per run | 5,000 bytes |
| Text per document (runs, code, alt, captions) | 200,000 bytes |
| Code block source | 20,000 bytes |
| Table | 60 rows × 10 columns |
| Gallery | 12 pictures |

A document over any limit is refused as a whole.

## Versions

| Version | Change |
| --- | --- |
| 1 | Paragraphs, headings, lists, quotes, images, galleries, dividers. |
| 2 | Adds `taskList`, `codeBlock`, `table`, `callout` and heading `anchor`. |

The server reads versions 1 and 2 and writes 2. A document saved in version 1 is returned in version 2.

Compatibility promise:

- New versions only add blocks and keys. Existing ones keep their name and meaning.
- Every version ever written stays readable.
- A document sent in the current version is returned in the current version.
- The document format is defined here, not by the web editor. Editor internals are not part of the contract.
