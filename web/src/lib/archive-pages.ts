const REACH = 1;

export type ArchiveStep = number | "gap";

export function archiveSteps(page: number, pages: number): ArchiveStep[] {
  if (pages < 2) return [];
  const shown = new Set<number>([1, pages]);
  for (let near = page - REACH; near <= page + REACH; near++) {
    if (near >= 1 && near <= pages) shown.add(near);
  }
  const steps: ArchiveStep[] = [];
  let last = 0;
  for (const number of [...shown].sort((a, b) => a - b)) {
    if (number === last + 2) steps.push(last + 1);
    else if (last !== 0 && number > last + 1) steps.push("gap");
    steps.push(number);
    last = number;
  }
  return steps;
}
