"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { DefaultCover } from "@/components/media/DefaultCover";
import type { BrowseType } from "@/lib/api/query";
import type {
  ColorSetContent,
  SettingGroupContent,
  TextSetContent,
  WorkBlock,
  WorkElement,
} from "@/lib/api/shapes";
import { cn } from "@/lib/cn";
import { nameSlot } from "@/lib/preset-slots";
import { themeColorName } from "@/lib/theme-colors";
import { TYPE_LABELS } from "@/lib/work-types";

const EASE = [0.16, 1, 0.3, 1] as const;

const SPANS: Record<string, string> = {
  full: "col-span-12",
  two_thirds: "col-span-12 sm:col-span-8",
  half: "col-span-12 sm:col-span-6",
  third: "col-span-12 sm:col-span-4",
};

const SHOWN_ITEMS = 6;

/** DraftPreview draws the page an empty draft opens with, from the same blocks the draft is made from. */
export function DraftPreview({
  type,
  blocks,
}: {
  type: BrowseType;
  blocks: WorkBlock[];
}) {
  const label = TYPE_LABELS[type].toLowerCase();
  return (
    <div
      aria-hidden="true"
      className="relative overflow-hidden rounded-plate bg-field p-5 shadow-[0_24px_60px_-28px_rgb(0_0_0/0.45)] ring-1 ring-rule/70 sm:p-7"
    >
      <div className="grid grid-cols-[minmax(0,1fr)_5.5rem] items-center gap-5 sm:grid-cols-[minmax(0,1fr)_7rem_minmax(0,1fr)]">
        <div className="min-w-0">
          <p className="font-display text-[1.6rem] leading-[1.05] font-medium tracking-[-0.03em] text-mute/70 italic">
            Name this page
          </p>
          <p className="mt-2 flex flex-wrap items-center gap-1.5 text-label text-mute">
            <span className="size-1.5 rounded-full bg-accent" />
            {TYPE_LABELS[type]}
            <span className="rounded-control bg-accent-wash px-1.5 py-0.5 font-medium text-accent">
              Private draft
            </span>
          </p>
        </div>
        <span className="relative block aspect-[5/6] overflow-hidden rounded-control shadow-[0_14px_30px_-16px_rgb(0_0_0/0.5)]">
          <DefaultCover compact type={type} />
        </span>
        <div className="col-span-2 sm:col-span-1">
          <p className="text-label font-medium text-ink">Blurb</p>
          <p className="mt-1 rounded-control bg-deep px-2.5 py-2 text-label text-mute">
            Describe this {label} in a few words
          </p>
        </div>
      </div>

      <p className="mt-6 flex flex-wrap gap-x-4 gap-y-1 border-b border-rule/70 pb-2.5 text-label text-mute">
        <span>On this page</span>
        {blocks.map((block, index) => (
          <span className={cn(index === 0 && "text-accent")} key={block.id}>
            {block.title}
          </span>
        ))}
      </p>

      <div className="mt-5 grid grid-cols-12 gap-x-5 gap-y-6">
        {blocks.map((block, index) => (
          <motion.section
            animate={{ opacity: 1, y: 0 }}
            className={cn("min-w-0", SPANS[block.width] ?? "col-span-12")}
            initial={{ opacity: 0, y: 10 }}
            key={block.id}
            transition={{ delay: 0.06 * index, duration: 0.35, ease: EASE }}
          >
            <h4 className="font-display text-ui font-medium text-ink">
              {block.title}
            </h4>
            <div
              className={cn(
                "mt-2.5 grid gap-3",
                block.elements.length > 1 &&
                  block.layout === "trio" &&
                  "sm:grid-cols-3",
                block.elements.length > 1 &&
                  block.layout === "duo" &&
                  "sm:grid-cols-2",
              )}
            >
              {block.elements.map((element) => (
                <div className="min-w-0" key={element.id}>
                  <p className="text-label text-mute">{element.label}</p>
                  <ElementSketch element={element} />
                </div>
              ))}
            </div>
          </motion.section>
        ))}
      </div>

      <span className="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-field to-transparent" />
    </div>
  );
}

function ElementSketch({ element }: { element: WorkElement }) {
  switch (element.type) {
    case "setting_group":
      return (
        <Items
          names={(element.content as SettingGroupContent).settings.map(
            (setting) => setting.label ?? nameSlot(setting.name).name,
          )}
          shape="setting"
        />
      );
    case "color_set":
      return (
        <Items
          names={(element.content as ColorSetContent).modes.flatMap((mode) =>
            mode.colors.map((color) => themeColorName(color.name)),
          )}
          shape="color"
        />
      );
    case "text_set": {
      const named = (element.content as TextSetContent).texts.filter(
        (text) => text.name,
      );
      return named.length > 0 ? (
        <Items
          names={named.map((text) => nameSlot(text.name ?? "").name)}
          shape="text"
        />
      ) : (
        <Bubbles />
      );
    }
    case "dialogue_sample":
      return <Bubbles alternate />;
    case "entry_table":
      return <Rows columns />;
    case "prompt_list":
    case "record_list":
      return <Rows />;
    default:
      return <Lines />;
  }
}

/** Items lists the named slots an app's draft arrives with, so switching app swaps them in place. */
function Items({
  names,
  shape,
}: {
  names: string[];
  shape: "setting" | "color" | "text";
}) {
  const still = useReducedMotion();
  const shown = names.slice(0, SHOWN_ITEMS);
  return (
    <ul
      className={cn(
        "mt-1.5 grid list-none gap-1 p-0",
        shape === "color" && "grid-cols-2",
      )}
    >
      <AnimatePresence initial={false} mode="popLayout">
        {shown.map((name, index) => (
          <motion.li
            animate={{ opacity: 1, x: 0 }}
            className="flex min-w-0 items-center gap-1.5 rounded-[6px] bg-inset px-2 py-1"
            exit={{ opacity: 0, x: 8 }}
            initial={still ? false : { opacity: 0, x: -8 }}
            key={`${index}-${name}`}
            layout={!still}
            transition={{ duration: 0.25, ease: EASE }}
          >
            {shape === "color" ? (
              <span className="size-3 shrink-0 rounded-full bg-[repeating-linear-gradient(45deg,var(--v-rule)_0_2px,transparent_2px_4px)] ring-1 ring-rule" />
            ) : null}
            <span className="min-w-0 flex-1 truncate text-label text-ink">
              {name}
            </span>
            {shape === "setting" ? (
              <span className="h-2.5 w-7 shrink-0 rounded-full bg-deep" />
            ) : null}
          </motion.li>
        ))}
      </AnimatePresence>
      {names.length > SHOWN_ITEMS ? (
        <li className="px-2 text-label text-mute">
          {names.length - SHOWN_ITEMS} more
        </li>
      ) : null}
    </ul>
  );
}

function Lines() {
  return (
    <span className="mt-1.5 grid gap-1.5 rounded-control bg-inset p-2.5">
      <span className="h-1.5 w-11/12 rounded-full bg-deep" />
      <span className="h-1.5 w-4/5 rounded-full bg-deep" />
      <span className="h-1.5 w-2/3 rounded-full bg-deep" />
    </span>
  );
}

function Bubbles({ alternate = false }: { alternate?: boolean }) {
  return (
    <span className="mt-1.5 grid gap-1.5">
      <span className="h-5 w-3/4 rounded-control rounded-bl-[3px] bg-inset" />
      <span
        className={cn(
          "h-5 w-1/2 rounded-control bg-inset",
          alternate
            ? "justify-self-end rounded-br-[3px] bg-accent-wash"
            : "rounded-bl-[3px]",
        )}
      />
    </span>
  );
}

function Rows({ columns = false }: { columns?: boolean }) {
  return (
    <span className="mt-1.5 grid gap-1">
      {[0, 1, 2].map((row) => (
        <span
          className="grid grid-cols-[1fr_2fr] items-center gap-2 rounded-[6px] bg-inset px-2 py-1.5"
          key={row}
        >
          <span
            className={cn(
              "h-1.5 rounded-full bg-deep",
              columns ? "w-full" : "col-span-2 w-5/6",
            )}
          />
          {columns ? (
            <span className="h-1.5 w-4/5 rounded-full bg-deep" />
          ) : null}
        </span>
      ))}
    </span>
  );
}
