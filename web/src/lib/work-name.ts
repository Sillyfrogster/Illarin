export function workDisplayName(name: string): string {
  return name.trim() === "" ? "Untitled" : name;
}
