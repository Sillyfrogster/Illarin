import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

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

export function cn(...values: ClassValue[]) {
  return merge(clsx(values));
}
