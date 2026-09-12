export function assetDisplayName(name: string): string {
  return name.trim() === "" ? "Untitled" : name;
}
