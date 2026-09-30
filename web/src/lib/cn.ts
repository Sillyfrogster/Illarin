import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

const merge = extendTailwindMerge({
  extend: {
    theme: {
      spacing: [
        "control",
        "control-compact",
        "segment",
        "segment-compact",
        "flow",
        "group",
        "section",
        "chapter",
      ],
    },
    classGroups: {
      "font-size": [
        {
          text: [
            "hero",
            "display",
            "title",
            "section",
            "brand",
            "lede",
            "prose",
            "article",
            "ui",
            "meta",
            "label",
          ],
        },
      ],
      "font-family": [{ font: ["display", "ui", "prose"] }],
      rounded: [{ rounded: ["control", "plate"] }],
    },
  },
});

export function cn(...values: ClassValue[]) {
  return merge(clsx(values));
}

/** focusRing is the one keyboard focus ring every control draws in place of the global outline. */
export const focusRing =
  "outline-none focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/50";
