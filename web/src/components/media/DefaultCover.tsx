import {
  BookOpen,
  PackageOpen,
  Palette,
  SlidersHorizontal,
  UserRound,
} from "lucide-react";
import type { BrowseKind } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { KIND_LABELS } from "@/lib/kinds";

const ICONS = {
  character: UserRound,
  lorebook: BookOpen,
  preset: SlidersHorizontal,
  theme: Palette,
  pack: PackageOpen,
} as const;

const SHARD = "polygon(18% 0, 100% 0, 82% 100%, 0 100%)";

const STAR =
  "polygon(50% 0, 58% 42%, 100% 50%, 58% 58%, 50% 100%, 42% 58%, 0 50%, 42% 42%)";

const KINDS: Record<
  BrowseKind,
  { ground: string; ink: string; shard: string; star: string }
> = {
  character: {
    ground: "bg-media",
    ink: "text-on-media",
    shard: "-top-[13%] -right-[20%] h-[52%] w-[88%] rotate-[-16deg]",
    star: "-right-[27%] -bottom-[35%] w-[72%]",
  },
  lorebook: {
    ground: "bg-media brightness-[1.06]",
    ink: "text-on-media/75",
    shard: "-top-[8%] right-[35%] h-[52%] w-[88%] rotate-[18deg]",
    star: "-right-[18%] -bottom-[42%] w-[72%] rotate-[12deg]",
  },
  preset: {
    ground: "bg-media brightness-[1.12]",
    ink: "text-on-media",
    shard: "top-[14%] -right-[28%] h-[52%] w-[88%] rotate-[-8deg]",
    star: "-bottom-[38%] -left-[24%] w-[72%] rotate-45",
  },
  theme: {
    ground: "bg-media brightness-[1.18]",
    ink: "text-on-media/65",
    shard: "-top-[22%] -right-[8%] h-[52%] w-[68%] rotate-[24deg]",
    star: "-right-[34%] -bottom-[28%] w-[72%] rotate-[22.5deg]",
  },
  pack: {
    ground: "bg-media brightness-[1.09]",
    ink: "text-on-media",
    shard: "-top-[18%] -right-[2%] h-[52%] w-[58%] rotate-[34deg]",
    star: "-bottom-[32%] -left-[20%] w-[72%] rotate-[12deg]",
  },
};

export function DefaultCover({
  kind,
  compact = false,
}: {
  kind: BrowseKind;
  compact?: boolean;
}) {
  const Icon = ICONS[kind];
  const face = KINDS[kind];

  return (
    <span
      aria-hidden="true"
      className={cn(
        "group/cover absolute inset-0 grid content-center justify-items-center overflow-hidden [isolation:isolate]",
        face.ground,
        face.ink,
        compact ? "gap-0" : "gap-5",
      )}
    >
      <span
        className={cn("absolute z-0 bg-current/8", face.shard)}
        style={{ clipPath: SHARD }}
      />
      <span
        className={cn("absolute z-0 aspect-square bg-current/5", face.star)}
        style={{ clipPath: STAR }}
      />
      <span
        className={cn(
          "relative z-1 grid aspect-square place-items-center [isolation:isolate]",
          compact ? "w-[54%] max-w-[46px]" : "w-14 sm:w-24",
        )}
      >
        <span
          className="absolute -z-1 inset-0 bg-current/15 transition-transform duration-500 group-hover/cover:rotate-45 motion-reduce:transition-none"
          style={{ clipPath: STAR }}
        />
        <Icon size={compact ? 22 : 38} strokeWidth={1.25} />
      </span>
      {compact ? null : (
        <span className="relative z-1 text-label font-semibold tracking-[0.18em] uppercase">
          {KIND_LABELS[kind]}
        </span>
      )}
    </span>
  );
}
