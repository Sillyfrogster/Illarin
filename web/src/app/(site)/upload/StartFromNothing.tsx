"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useRouter } from "next/navigation";
import { type KeyboardEvent, useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { type BrowseType, startWork } from "@/lib/api/query";
import type { BuildChoices } from "@/lib/api/shapes";
import { cn } from "@/lib/cn";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";

const EASE = [0.16, 1, 0.3, 1] as const;

/** StartFromNothing shows each type the API can build from nothing as a sketch of the draft it opens, asking for the app where the type needs one. */
export function StartFromNothing({
  choices,
}: {
  choices: BuildChoices | null;
}) {
  const router = useRouter();
  const [pending, setPending] = useState<BrowseType | null>(null);
  const [asking, setAsking] = useState<BrowseType | null>(null);
  const [message, setMessage] = useState("");

  const buildable = (Object.keys(TYPE_LABELS) as BrowseType[]).flatMap(
    (type) => choices?.types.filter((choice) => choice.type === type) ?? [],
  );

  async function start(type: BrowseType, app?: string) {
    setPending(type);
    setMessage("");
    try {
      const started = await startWork(type, app);
      router.push(workHref(started.id, started.name));
    } catch {
      setPending(null);
      setMessage("Illarin could not start the draft. Try again.");
    }
  }

  return (
    <section aria-labelledby="start-heading" className="mt-16">
      <h2
        className="font-display text-section font-medium text-ink"
        id="start-heading"
      >
        Start an empty draft
      </h2>
      <p className="mt-1.5 text-ui text-mute">
        Pick a type and build it in the editor.
      </p>

      {choices === null ? (
        <p className="mt-5 text-meta text-mute" role="alert">
          The draft types could not load. Reload the page to try again.
        </p>
      ) : null}

      <ul className="mt-6 grid list-none grid-cols-2 gap-3 p-0 sm:grid-cols-3 lg:grid-cols-5">
        {buildable.map(({ type: listed, apps, blocks }) => {
          const type = listed as BrowseType;
          return (
            <li key={type}>
              <DraftChoice
                apps={apps}
                blocks={blocks}
                asking={asking === type}
                disabled={pending !== null}
                onAsk={(open) => setAsking(open ? type : null)}
                onStart={(app) => void start(type, app)}
                pending={pending === type}
                type={type}
              />
            </li>
          );
        })}
      </ul>

      {message ? (
        <p className="mt-4 text-meta text-stop" role="alert">
          {message}
        </p>
      ) : null}
    </section>
  );
}

function DraftChoice({
  apps,
  blocks,
  asking,
  disabled,
  onAsk,
  onStart,
  pending,
  type,
}: {
  apps: BuildChoices["types"][number]["apps"];
  blocks: string[];
  asking: boolean;
  disabled: boolean;
  onAsk: (open: boolean) => void;
  onStart: (app?: string) => void;
  pending: boolean;
  type: BrowseType;
}) {
  const still = useReducedMotion();
  const label = TYPE_LABELS[type];
  const lit = asking || pending;
  const close = (event: KeyboardEvent) => {
    if (event.key === "Escape" && asking) onAsk(false);
  };

  return (
    <div className="group relative">
      <button
        aria-expanded={apps.length > 0 ? asking : undefined}
        className="block w-full rounded-plate text-left outline-offset-3 disabled:cursor-progress"
        disabled={disabled}
        onClick={() => (apps.length > 0 ? onAsk(!asking) : onStart())}
        onKeyDown={close}
        type="button"
      >
        <span
          className={cn(
            "relative block aspect-[4/5] overflow-hidden rounded-plate bg-inset transition-[translate,box-shadow,background-color] duration-300 ease-[var(--ease-wipe)] group-hover:-translate-y-1 group-hover:shadow-[0_20px_45px_-24px_rgb(0_0_0/0.55)] motion-reduce:transform-none",
            lit &&
              "-translate-y-1 bg-accent-wash shadow-[0_20px_45px_-24px_rgb(0_0_0/0.55)]",
          )}
        >
          <DraftSketch blocks={blocks} type={type} />
          {pending ? (
            <span className="absolute inset-x-0 bottom-0 bg-field/85 px-3 py-2.5 text-meta text-ink backdrop-blur-sm">
              Starting…
            </span>
          ) : null}
        </span>
        <span className="mt-3 flex items-center gap-2 font-ui text-ui font-medium text-ink transition-colors duration-200 group-hover:text-accent">
          <TypeMark className="size-4 shrink-0 text-accent" type={type} />
          {label}
        </span>
        {apps.length > 0 ? (
          <span className="mt-0.5 block text-meta text-mute">
            For {apps.map((app) => app.label).join(" or ")}
          </span>
        ) : null}
      </button>

      <AnimatePresence>
        {asking && !pending ? (
          <motion.div
            animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
            className="absolute inset-x-2 bottom-[4.75rem] z-10 flex flex-col gap-1.5 rounded-control bg-plane p-1.5 shadow-[0_16px_60px_-12px_rgb(0_0_0/0.35)]"
            exit={{ opacity: 0, y: 6, filter: "blur(4px)" }}
            initial={still ? false : { opacity: 0, y: 10, filter: "blur(4px)" }}
            transition={{ duration: 0.25, ease: EASE }}
          >
            {apps.map((app) => (
              <button
                className="flex min-h-11 items-center justify-center rounded-control bg-deep px-3 font-ui text-ui font-medium text-ink transition-colors duration-150 hover:bg-action hover:text-on-accent"
                key={app.id}
                onClick={() => onStart(app.id)}
                onKeyDown={close}
                type="button"
              >
                {app.label}
              </button>
            ))}
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

const LINE_WIDTHS = ["w-11/12", "w-3/4", "w-5/6"];

/** DraftSketch draws the empty draft a type opens with, its blocks filling in while the choice is hovered. */
function DraftSketch({ blocks, type }: { blocks: string[]; type: BrowseType }) {
  return (
    <span aria-hidden="true" className="flex h-full flex-col gap-2 p-3">
      <span className="flex items-center gap-2">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-control bg-deep">
          <TypeMark className="size-4 text-accent" type={type} />
        </span>
        <span className="grid flex-1 gap-1">
          <span className="h-1.5 w-2/3 rounded-full bg-ink/70" />
          <span className="h-1 w-1/3 rounded-full bg-rule" />
        </span>
      </span>
      {blocks.map((title, index) => (
        <span
          className="flex min-h-0 flex-1 flex-col gap-1.5 overflow-hidden rounded-control bg-plane p-2"
          key={title}
        >
          <span className="truncate font-ui text-label font-medium text-mute">
            {title}
          </span>
          {LINE_WIDTHS.map((width, line) => (
            <span className="h-1 rounded-full bg-deep" key={width}>
              <span
                className={cn(
                  "block h-full origin-left scale-x-0 rounded-full bg-accent/60 transition-transform duration-500 ease-[var(--ease-wipe)] group-hover:scale-x-100 group-focus-within:scale-x-100 motion-reduce:transition-none",
                  width,
                )}
                style={{ transitionDelay: `${(index * 3 + line) * 90}ms` }}
              />
            </span>
          ))}
        </span>
      ))}
    </span>
  );
}
