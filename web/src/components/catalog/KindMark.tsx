import {
  BookOpen,
  PackageOpen,
  Palette,
  Puzzle,
  SlidersHorizontal,
  UserRound,
} from "lucide-react";
import type { BrowseKind } from "@/lib/api/query";

export const KIND_MARKS = {
  character: UserRound,
  lorebook: BookOpen,
  preset: SlidersHorizontal,
  theme: Palette,
  pack: PackageOpen,
  extension: Puzzle,
} as const satisfies Record<BrowseKind, unknown>;

export function KindMark({
  className,
  kind,
}: {
  className?: string;
  kind: BrowseKind;
}) {
  const Mark = KIND_MARKS[kind];
  return <Mark aria-hidden="true" className={className} strokeWidth={1.5} />;
}
