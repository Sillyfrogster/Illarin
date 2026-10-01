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

/** popupSurface is the raised plane every menu, list and anchored panel opens on, popping in toward its trigger. */
export const popupSurface =
  "z-90 rounded-art bg-plane font-ui text-ink shadow-popover ring-1 ring-ink/8 outline-none animate-pop data-[state=closed]:animate-leave";

/** popupRow is one choosable row in a popup, lit grey while the pointer or the arrow keys are on it. */
export const popupRow =
  "flex min-h-control cursor-pointer items-center gap-2 rounded-control px-2 text-ui outline-none select-none focus-visible:outline-none data-highlighted:bg-fill-hover data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0";
