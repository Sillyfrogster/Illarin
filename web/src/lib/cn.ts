import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

// Without these the merger reads a named step such as text-lede as a colour and drops it
const merge = extendTailwindMerge({
  extend: {
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
    },
  },
});

/** Joins class names and keeps the last of any two that would fight */
export function cn(...values: ClassValue[]) {
  return merge(clsx(values));
}
