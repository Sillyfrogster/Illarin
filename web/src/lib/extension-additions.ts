/** One kind of thing an extension adds to its app, counting every addition while showing the first few. */
export type AdditionGroup = {
  name: string;
  total: number;
  shown: { key: string; name: string }[];
};

/** groupAdditions gathers what an extension's code registers under each kind of thing, in the order the code was read. */
export function groupAdditions(
  fields: readonly { name?: string; value: string }[],
  limit = fields.length,
): AdditionGroup[] {
  const groups = new Map<string, AdditionGroup>();
  fields.forEach((field, index) => {
    const name = field.name ?? "";
    const group = groups.get(name) ?? { name, total: 0, shown: [] };
    groups.set(name, group);
    group.total += 1;
    if (index < limit) group.shown.push({ key: `${index}`, name: field.value });
  });
  return [...groups.values()].filter((group) => group.shown.length > 0);
}
