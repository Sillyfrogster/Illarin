export function moved<T>(items: T[], from: number, to: number): T[] {
  const next = items.slice();
  const [lifted] = next.splice(from, 1);
  next.splice(to, 0, lifted);
  return next;
}
