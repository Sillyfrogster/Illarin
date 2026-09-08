/** One run of text, and whether an update left it alone, took it out or put it in. */
export type DiffPiece = {
  kind: "same" | "removed" | "added";
  text: string;
};

// Beyond this many differing words a side is reported whole, because the comparison table grows with both sides multiplied together.
const WORD_LIMIT = 600;

/** What an update did to one piece of text, word by word, reporting a wholly rewritten passage as one removal and one addition rather than a trail of coincidental words. */
export function wordDiff(before: string, after: string): DiffPiece[] {
  const earlier = words(before);
  const later = words(after);

  let head = 0;
  while (
    head < earlier.length &&
    head < later.length &&
    earlier[head] === later[head]
  ) {
    head += 1;
  }
  let tail = 0;
  while (
    tail < earlier.length - head &&
    tail < later.length - head &&
    earlier[earlier.length - 1 - tail] === later[later.length - 1 - tail]
  ) {
    tail += 1;
  }

  const removed = earlier.slice(head, earlier.length - tail);
  const added = later.slice(head, later.length - tail);
  const middle =
    removed.length > WORD_LIMIT || added.length > WORD_LIMIT
      ? [
          { kind: "removed" as const, text: removed.join("") },
          { kind: "added" as const, text: added.join("") },
        ]
      : alignWords(removed, added);

  return merge([
    { kind: "same", text: earlier.slice(0, head).join("") },
    ...middle,
    { kind: "same", text: earlier.slice(earlier.length - tail).join("") },
  ]);
}

// words splits text into words and the spacing between them, so rejoining the pieces returns the original.
function words(text: string): string[] {
  return text.split(/(\s+)/).filter((piece) => piece !== "");
}

// alignWords keeps the longest run of words both sides share and reports the rest as taken out or put in.
function alignWords(removed: string[], added: string[]): DiffPiece[] {
  const shared: number[][] = Array.from({ length: removed.length + 1 }, () =>
    new Array(added.length + 1).fill(0),
  );
  for (let one = removed.length - 1; one >= 0; one -= 1) {
    for (let other = added.length - 1; other >= 0; other -= 1) {
      shared[one][other] =
        removed[one] === added[other]
          ? shared[one + 1][other + 1] + 1
          : Math.max(shared[one + 1][other], shared[one][other + 1]);
    }
  }

  const pieces: DiffPiece[] = [];
  let one = 0;
  let other = 0;
  while (one < removed.length && other < added.length) {
    if (removed[one] === added[other]) {
      pieces.push({ kind: "same", text: removed[one] });
      one += 1;
      other += 1;
    } else if (shared[one + 1][other] >= shared[one][other + 1]) {
      pieces.push({ kind: "removed", text: removed[one] });
      one += 1;
    } else {
      pieces.push({ kind: "added", text: added[other] });
      other += 1;
    }
  }
  for (; one < removed.length; one += 1) {
    pieces.push({ kind: "removed", text: removed[one] });
  }
  for (; other < added.length; other += 1) {
    pieces.push({ kind: "added", text: added[other] });
  }
  return pieces;
}

// merge joins neighbouring pieces of one kind and drops the empty ones.
function merge(pieces: DiffPiece[]): DiffPiece[] {
  const joined: DiffPiece[] = [];
  for (const piece of pieces) {
    if (piece.text === "") continue;
    const last = joined[joined.length - 1];
    if (last?.kind === piece.kind) {
      last.text += piece.text;
      continue;
    }
    joined.push({ ...piece });
  }
  return joined;
}
