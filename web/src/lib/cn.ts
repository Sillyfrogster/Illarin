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
            "name-l",
            "name-m",
            "name-s",
          ],
        },
      ],
      "font-family": [{ font: ["display", "ui", "prose"] }],
      rounded: [{ rounded: ["chip", "control", "art", "plate", "card"] }],
    },
  },
});

export function cn(...values: ClassValue[]) {
  return merge(clsx(values));
}

/** focusRing is the one keyboard focus ring every control draws in place of the global outline. */
export const focusRing =
  "outline-none focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent";

/** navLink is a plain link in a run of navigation: ink at rest, a violet underline on hover, violet for the page you are on. */
export const navLink =
  "flex min-h-control items-center whitespace-nowrap text-ui font-medium text-ink decoration-accent underline-offset-4 hover:underline aria-[current=page]:text-accent";
