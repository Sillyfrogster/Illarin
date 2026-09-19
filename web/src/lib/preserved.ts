/** describePreservedLabels joins the labels the API gave each namespace into one phrase, naming each once. */
export function describePreservedLabels(labels: readonly string[]) {
  const unique = [...new Set(labels)];
  if (unique.length === 1) return unique[0];
  if (unique.length === 2) return unique.join(" and ");
  return `${unique.slice(0, -1).join(", ")}, and ${unique.at(-1)}`;
}
