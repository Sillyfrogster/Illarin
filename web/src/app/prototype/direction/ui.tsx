"use client";

import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { type ClassValue, clsx } from "clsx";
import type { ComponentProps, CSSProperties, ReactNode } from "react";
import { extendTailwindMerge } from "tailwind-merge";
import type { Category, Tint } from "./data";
import { CATEGORY_LABEL } from "./data";

// Without this the merger reads a named size such as text-lede as a colour and drops it
const merge = extendTailwindMerge({
  prefix: "vd",
  extend: {
    classGroups: {
      "font-size": [
        {
          text: [
            "hero",
            "display",
            "title",
            "section",
            "lede",
            "prose",
            "ui",
            "meta",
            "label",
          ],
        },
      ],
    },
  },
});
export function cn(...values: ClassValue[]) {
  return merge(clsx(values));
}

export type Direction = "vitrine" | "ambient" | "ledger";

/** The two hues a page borrows from its own media, plus the ink that stays readable on the field */
export function tintStyle(
  tint: Tint | undefined,
  dark: boolean,
): CSSProperties {
  if (!tint) return {};
  return {
    "--v-tint-a": tint.a,
    "--v-tint-b": tint.b,
    "--v-tint-ink": dark ? lighten(tint.a) : tint.ink,
  } as CSSProperties;
}

function lighten(hex: string) {
  return `color-mix(in oklab, ${hex} 62%, white)`;
}

const actionVariants = cva(
  "vd:relative vd:inline-flex vd:min-h-11 vd:shrink-0 vd:items-center vd:justify-center vd:gap-2 vd:rounded-control vd:px-5 vd:text-ui vd:font-semibold vd:tracking-tight vd:transition vd:duration-200 vd:disabled:cursor-not-allowed vd:disabled:opacity-40 vd:[&_svg]:size-4 vd:[&_svg]:shrink-0 vd:motion-reduce:transition-none",
  {
    variants: {
      variant: {
        primary:
          "vd:bg-strong vd:text-on-strong vd:hover:-translate-y-px vd:active:translate-y-0",
        secondary:
          "vd:bg-transparent vd:text-ink vd:shadow-[inset_0_0_0_1px_var(--v-rule)] vd:hover:bg-ink/6",
        tinted:
          "vd:text-on-strong vd:hover:-translate-y-px vd:active:translate-y-0",
        quiet: "vd:px-3 vd:text-ink vd:hover:bg-ink/8",
      },
      size: {
        default: "",
        large: "vd:min-h-13 vd:px-7 vd:text-base",
        icon: "vd:size-11 vd:px-0",
      },
    },
    defaultVariants: { variant: "secondary", size: "default" },
  },
);

export function Action({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ComponentProps<"button"> &
  VariantProps<typeof actionVariants> & { asChild?: boolean }) {
  const Component = asChild ? Slot : "button";
  return (
    <Component
      type={asChild ? undefined : "button"}
      className={cn(
        actionVariants({ variant, size }),
        variant === "tinted" &&
          "vd:bg-[var(--v-tint-a)] vd:text-white vd:mix-blend-normal",
        className,
      )}
      {...props}
    />
  );
}

export function Label({
  children,
  className,
  style,
}: {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <p
      style={style}
      className={cn(
        "vd:text-label vd:font-bold vd:uppercase vd:text-mute",
        className,
      )}
    >
      {children}
    </p>
  );
}

export function Rule({
  accent,
  className,
  style,
}: {
  accent?: boolean;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <div
      aria-hidden="true"
      style={style}
      className={cn(
        "vd:h-px vd:w-full",
        accent ? "v-accent-rule" : "vd:bg-rule",
        className,
      )}
    />
  );
}

/** One byline shape for every surface, so the same facts never land in a different order */
export function Byline({
  author,
  handle,
  category,
  date,
  updated,
  minutes,
  className,
  tone = "mute",
}: {
  author: string;
  handle?: string;
  category: Category;
  date: string;
  updated?: string;
  minutes?: number;
  className?: string;
  tone?: "mute" | "over";
}) {
  const parts = [
    CATEGORY_LABEL[category],
    date,
    updated && `updated ${updated}`,
    minutes && `${minutes} min`,
  ].filter(Boolean) as string[];
  return (
    <div
      className={cn(
        "vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-3 vd:gap-y-1 vd:text-meta",
        tone === "over" ? "vd:text-over-mute" : "vd:text-mute",
        className,
      )}
    >
      <span
        className={cn(
          "vd:font-semibold",
          tone === "over" ? "vd:text-over" : "vd:text-ink",
        )}
      >
        {author}
      </span>
      {handle && <span>{handle}</span>}
      <span aria-hidden="true" className="vd:opacity-40">
        /
      </span>
      {parts.map((part, index) => (
        <span key={part}>
          {part}
          {index < parts.length - 1 && (
            <span aria-hidden="true" className="vd:pl-3 vd:opacity-40">
              ·
            </span>
          )}
        </span>
      ))}
    </div>
  );
}

export function Tag({ children }: { children: ReactNode }) {
  return (
    <span className="vd:inline-flex vd:min-h-7 vd:items-center vd:rounded-control vd:px-2.5 vd:text-meta vd:text-mute vd:shadow-[inset_0_0_0_1px_var(--v-hair)]">
      {children}
    </span>
  );
}

/** A short inline convention, so synthetic copy can carry the emphasis real writing has */
export function Inline({ text }: { text: string }) {
  const parts = text.split(/(\*\*[^*]+\*\*|_[^_]+_)/g).filter(Boolean);
  return (
    <>
      {parts.map((part, index) => {
        if (part.startsWith("**") && part.endsWith("**")) {
          return (
            <strong
              // biome-ignore lint/suspicious/noArrayIndexKey: the split is stable for a fixed string
              key={index}
              className="vd:font-semibold vd:text-ink"
            >
              {part.slice(2, -2)}
            </strong>
          );
        }
        if (part.startsWith("_") && part.endsWith("_")) {
          // biome-ignore lint/suspicious/noArrayIndexKey: the split is stable for a fixed string
          return <em key={index}>{part.slice(1, -1)}</em>;
        }
        // biome-ignore lint/suspicious/noArrayIndexKey: the split is stable for a fixed string
        return <span key={index}>{part}</span>;
      })}
    </>
  );
}

export function Empty({
  title,
  detail,
  className,
}: {
  title: string;
  detail: string;
  className?: string;
}) {
  return (
    <div className={cn("vd:py-16", className)}>
      <p className="vd:font-display vd:text-title vd:text-mute">{title}</p>
      <p className="v-caption vd:mt-3 vd:text-ui vd:text-faint">{detail}</p>
    </div>
  );
}
