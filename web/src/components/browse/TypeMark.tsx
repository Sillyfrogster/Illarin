import {
  BookOpen,
  PackageOpen,
  Palette,
  Puzzle,
  SlidersHorizontal,
  UserRound,
} from "lucide-react";
import type { BrowseType } from "@/lib/api/query";

export const TYPE_MARKS = {
  character: UserRound,
  lorebook: BookOpen,
  preset: SlidersHorizontal,
  theme: Palette,
  pack: PackageOpen,
  extension: Puzzle,
} as const satisfies Record<BrowseType, unknown>;

export function TypeMark({
  className,
  type,
}: {
  className?: string;
  type: BrowseType;
}) {
  const Mark = TYPE_MARKS[type];
  return <Mark aria-hidden="true" className={className} strokeWidth={1.5} />;
}
