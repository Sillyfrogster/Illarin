export type TextSelection = { start: number; end: number };

export type TextEdit = { text: string; selection: TextSelection };

type WrapAction = "bold" | "italic" | "code";

type PrefixAction = "bullet" | "numbered" | "quote";

export type MarkdownAction = WrapAction | PrefixAction | "link";

const WRAPS: Record<WrapAction, string> = {
  bold: "**",
  italic: "_",
  code: "`",
};

const LINE_PREFIXES: Record<PrefixAction, RegExp> = {
  bullet: /^(\s*)- /,
  numbered: /^(\s*)\d+\. /,
  quote: /^(\s*)> /,
};

const LINK_TEXT = "text";
const LINK_ADDRESS = "url";

/** Applies one formatting action to a selection and says where the caret lands */
export function applyMarkdown(
  action: MarkdownAction,
  text: string,
  selection: TextSelection,
): TextEdit {
  switch (action) {
    case "bold":
    case "italic":
    case "code":
      return toggleWrap(text, trimmed(text, selection), WRAPS[action]);
    case "link":
      return insertLink(text, trimmed(text, selection));
    default:
      return togglePrefix(text, selection, action);
  }
}

function trimmed(text: string, selection: TextSelection): TextSelection {
  let { start, end } = selection;
  while (start < end && /\s/.test(text[start])) start += 1;
  while (end > start && /\s/.test(text[end - 1])) end -= 1;
  return { end, start };
}

function toggleWrap(
  text: string,
  selection: TextSelection,
  marker: string,
): TextEdit {
  const { start, end } = selection;
  const width = marker.length;

  if (
    text.slice(start - width, start) === marker &&
    text.slice(end, end + width) === marker
  ) {
    return {
      selection: { end: end - width, start: start - width },
      text:
        text.slice(0, start - width) +
        text.slice(start, end) +
        text.slice(end + width),
    };
  }

  const inner = text.slice(start, end);
  if (
    inner.length >= width * 2 &&
    inner.startsWith(marker) &&
    inner.endsWith(marker)
  ) {
    const stripped = inner.slice(width, -width);
    return {
      selection: { end: start + stripped.length, start },
      text: text.slice(0, start) + stripped + text.slice(end),
    };
  }

  return {
    selection: { end: end + width, start: start + width },
    text: text.slice(0, start) + marker + inner + marker + text.slice(end),
  };
}

function insertLink(text: string, selection: TextSelection): TextEdit {
  const { start, end } = selection;
  const label = start === end ? LINK_TEXT : text.slice(start, end);
  const written = `[${label}](${LINK_ADDRESS})`;
  const addressStart = start + label.length + 3;
  return {
    selection: {
      end: addressStart + LINK_ADDRESS.length,
      start: addressStart,
    },
    text: text.slice(0, start) + written + text.slice(end),
  };
}

function togglePrefix(
  text: string,
  selection: TextSelection,
  action: PrefixAction,
): TextEdit {
  const pattern = LINE_PREFIXES[action];
  const from = text.lastIndexOf("\n", Math.max(selection.start - 1, 0)) + 1;
  const next = text.indexOf("\n", selection.end);
  const to = next === -1 ? text.length : next;

  const lines = text.slice(from, to).split("\n");
  const written = lines.every(
    (line) => line.trim() === "" || pattern.test(line),
  )
    ? lines.map((line) => line.replace(pattern, "$1"))
    : lines.map((line, index) => addPrefix(line, action, index));

  const body = written.join("\n");
  const moved = (offset: number) => movedOffset(offset, from, lines, written);
  return {
    selection: { end: moved(selection.end), start: moved(selection.start) },
    text: text.slice(0, from) + body + text.slice(to),
  };
}

function movedOffset(
  offset: number,
  from: number,
  lines: string[],
  written: string[],
): number {
  let was = from;
  let now = from;
  for (const [index, line] of lines.entries()) {
    if (offset <= was + line.length) {
      const within = offset - was;
      if (within === 0) return now;
      const shift = written[index].length - line.length;
      return now + Math.max(0, Math.min(written[index].length, within + shift));
    }
    was += line.length + 1;
    now += written[index].length + 1;
  }
  return now;
}

function addPrefix(line: string, action: PrefixAction, index: number): string {
  const indent = line.match(/^\s*/)?.[0] ?? "";
  const body = line.slice(indent.length);
  if (body === "" && action !== "quote") return line;
  const marker =
    action === "bullet" ? "- " : action === "quote" ? "> " : `${index + 1}. `;
  return `${indent}${marker}${body}`;
}
