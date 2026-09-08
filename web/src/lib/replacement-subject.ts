/** What a replacement preview calls one part of an asset, in the words the page uses. */
export function replacementSubjectLabel(subject: string): string {
  if (subject === "images") return "Pictures";
  if (subject === "opaque_data") return "Preserved data";
  const words = subject.replaceAll("_", " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/** One part of the asset and what an uploaded file does to it. */
export type ReplacementSummary = {
  subject: string;
  label: string;
  detail: string;
  replacesYourEdit: boolean;
};

const KINDS = [
  { kind: "addition", one: "Added", many: "added" },
  { kind: "change", one: "Changed", many: "changed" },
  { kind: "removal", one: "Removed", many: "removed" },
];

/** Counts a preview's changes into one line for each part of the asset. */
export function summariseReplacement(
  changes: { kind: string; subject: string }[],
): ReplacementSummary[] {
  const subjects: string[] = [];
  for (const change of changes) {
    if (!subjects.includes(change.subject)) subjects.push(change.subject);
  }
  return subjects.map((subject) => {
    const own = changes.filter((change) => change.subject === subject);
    const counted = KINDS.map(({ kind, one, many }) => {
      const count = own.filter((change) => change.kind === kind).length;
      if (count === 0) return "";
      return count === 1 && own.length === 1 ? one : `${count} ${many}`;
    }).filter((part) => part !== "");
    return {
      subject,
      label: replacementSubjectLabel(subject),
      detail: `${counted.join(", ")} by the file`,
      replacesYourEdit: own.some((change) => change.kind === "conflict"),
    };
  });
}
