import type { BrowseType } from "./api/query";

export const WORK_TYPES: readonly BrowseType[] = [
  "character",
  "lorebook",
  "preset",
  "theme",
  "pack",
  "extension",
];

export const TYPE_LABELS: Record<BrowseType, string> = {
  character: "Character",
  lorebook: "Lorebook",
  preset: "Preset",
  theme: "Theme",
  pack: "Pack",
  extension: "Extension",
};

export const TYPE_PLURALS: Record<BrowseType, string> = {
  character: "Characters",
  lorebook: "Lorebooks",
  preset: "Presets",
  theme: "Themes",
  pack: "Packs",
  extension: "Extensions",
};

export function isWorkType(value: unknown): value is BrowseType {
  return WORK_TYPES.includes(value as BrowseType);
}
