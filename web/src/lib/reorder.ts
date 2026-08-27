/** Returns the list with one item lifted out and put back at another index. */
export function moved<T>(items: T[], from: number, to: number): T[] {
  const next = items.slice();
  const [lifted] = next.splice(from, 1);
  next.splice(to, 0, lifted);
  return next;
}
