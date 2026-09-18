import type { BrowseType, StartWorkApp } from "./api/query";

export const TYPE_LABELS: Record<BrowseType, string> = {
  character: "Character",
  lorebook: "Lorebook",
  preset: "Preset",
  theme: "Theme",
  pack: "Pack",
  extension: "Extension",
};

export const BUILDABLE_TYPES: BrowseType[] = [
  "character",
  "lorebook",
  "preset",
  "theme",
  "pack",
];

export const TYPES_ASKING_FOR_AN_APP: BrowseType[] = ["preset", "theme"];

export const APP_CHOICES: { value: StartWorkApp; label: string }[] = [
  { value: "sillytavern", label: "SillyTavern" },
  { value: "lumiverse", label: "Lumiverse" },
];
