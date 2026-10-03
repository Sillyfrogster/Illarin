const TAG_PREFIX = "tag:";
const AUTHOR_PREFIX = "author:";
const PLAIN_MINIMUM = 2;

/** tagQuery writes a tag as the search word that finds it, quoting a tag with a space. */
export function tagQuery(value: string) {
  return `${TAG_PREFIX}${value.includes(" ") ? `"${value}"` : value}`;
}

/** tagSearchHref links to the Browse search for one tag. */
export function tagSearchHref(value: string) {
  return `/browse?q=${encodeURIComponent(tagQuery(value))}`;
}

// A quoted phrase counts as one word, and -1 means the text ends after a finished word.
function lastWordStart(written: string) {
  let start = -1;
  let quoted = false;
  for (let index = 0; index < written.length; index++) {
    const character = written[index];
    if (character === '"') {
      quoted = !quoted;
      if (start < 0) start = index;
    } else if (!quoted && /\s/.test(character)) {
      start = -1;
    } else if (start < 0) {
      start = index;
    }
  }
  return start;
}

/** tagFragment returns the part of a tag the reader is typing, or null when the last word is not one. */
export function tagFragment(written: string) {
  const start = lastWordStart(written);
  if (start < 0) return null;
  const word = written.slice(start);
  const lower = word.toLowerCase();
  if (lower.startsWith(AUTHOR_PREFIX)) return null;
  const tagged = lower.startsWith(TAG_PREFIX);
  const typed = (tagged ? word.slice(TAG_PREFIX.length) : word)
    .replaceAll('"', "")
    .trim()
    .toLowerCase();
  return typed.length >= (tagged ? 1 : PLAIN_MINIMUM) ? typed : null;
}

/** chooseTag swaps the word being typed for the chosen tag's search word. */
export function chooseTag(written: string, value: string) {
  const start = lastWordStart(written);
  return (start < 0 ? written : written.slice(0, start)) + tagQuery(value);
}
