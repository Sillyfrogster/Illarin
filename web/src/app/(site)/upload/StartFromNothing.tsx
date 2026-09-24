"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ArrowRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import { Button } from "@/components/ui/button";
import { type BrowseType, startWork } from "@/lib/api/query";
import type { BuildChoice, BuildChoices } from "@/lib/api/shapes";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { DraftPreview } from "./DraftPreview";

const EASE = [0.16, 1, 0.3, 1] as const;

/** StartFromNothing lets a creator pick a type, and the app where the type needs one, beside the page that draft opens with. */
export function StartFromNothing({
  choices,
}: {
  choices: BuildChoices | null;
}) {
  const { account } = useAuth();
  const buildable = (Object.keys(TYPE_LABELS) as BrowseType[]).flatMap(
    (type) => choices?.types.filter((choice) => choice.type === type) ?? [],
  );

  if (!account?.emailVerified) return null;

  return (
    <section aria-labelledby="start-heading" className="mt-20">
      <h2
        className="font-display text-section font-medium text-ink"
        id="start-heading"
      >
        Start an empty draft
      </h2>
      <p className="mt-1.5 text-ui text-mute">
        This is the page your draft opens with. Fill it in the editor.
      </p>
      {buildable.length > 0 ? (
        <Builder choices={buildable} />
      ) : (
        <p className="mt-5 text-meta text-mute" role="alert">
          The draft types could not load. Reload the page to try again.
        </p>
      )}
    </section>
  );
}

function Builder({ choices }: { choices: BuildChoice[] }) {
  const router = useRouter();
  const still = useReducedMotion();
  const [chosen, setChosen] = useState(choices[0]);
  const [apps, setApps] = useState<Record<string, string>>({});
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  const type = chosen.type as BrowseType;
  const app = chosen.apps.length > 0 ? (apps[type] ?? chosen.apps[0].id) : "";
  const appLabel = chosen.apps.find((named) => named.id === app)?.label;
  const draft =
    chosen.drafts.find((candidate) => candidate.app === app) ??
    chosen.drafts[0];

  async function start() {
    setPending(true);
    setMessage("");
    try {
      const started = await startWork(type, app || undefined);
      router.push(workHref(started.id, started.name));
    } catch {
      setPending(false);
      setMessage("Illarin could not start the draft. Try again.");
    }
  }

  return (
    <div className="mt-7 grid items-start gap-6 lg:grid-cols-[13rem_minmax(0,1fr)_14rem] lg:gap-8">
      <fieldset className="flex min-w-0 gap-2 overflow-x-auto pb-1 [scrollbar-width:none] lg:flex-col lg:overflow-visible">
        <legend className="sr-only">Type</legend>
        {choices.map((choice) => {
          const listed = choice.type as BrowseType;
          const selected = choice.type === chosen.type;
          return (
            <label
              className={cn(
                "relative flex min-h-14 shrink-0 cursor-pointer items-center gap-3 rounded-control py-2 pr-4 pl-2 font-ui text-ui font-medium transition-colors duration-200 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent has-[:focus-visible]:outline-offset-3",
                selected ? "text-on-accent" : "text-ink hover:bg-deep",
              )}
              key={choice.type}
            >
              {selected ? (
                <motion.span
                  className="absolute inset-0 rounded-control bg-action shadow-[0_4px_14px_-5px_var(--v-action)]"
                  layoutId={still ? undefined : "draft-type"}
                  transition={{ type: "spring", stiffness: 420, damping: 34 }}
                />
              ) : null}
              <input
                checked={selected}
                className="sr-only"
                name="draft-type"
                onChange={() => setChosen(choice)}
                type="radio"
              />
              <span className="relative block aspect-[5/6] w-8 overflow-hidden rounded-[6px]">
                <DefaultCover compact type={listed} />
              </span>
              <span className="relative">{TYPE_LABELS[listed]}</span>
            </label>
          );
        })}
      </fieldset>

      <AnimatePresence initial={false} mode="wait">
        <motion.div
          animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
          exit={{ opacity: 0, y: -6, filter: "blur(4px)" }}
          className="min-w-0 max-lg:order-3"
          initial={still ? false : { opacity: 0, y: 10, filter: "blur(4px)" }}
          key={chosen.type}
          transition={{ duration: 0.28, ease: EASE }}
        >
          {draft ? <DraftPreview blocks={draft.blocks} type={type} /> : null}
        </motion.div>
      </AnimatePresence>

      <div className="grid min-w-0 content-start gap-5 max-lg:order-2 lg:sticky lg:top-[calc(var(--header-height)+2rem)]">
        {chosen.apps.length > 0 ? (
          <fieldset>
            <legend className="text-meta font-medium text-ink">App</legend>
            <p className="mt-0.5 text-label text-mute">
              Each app has its own settings.
            </p>
            <div className="mt-2.5 grid gap-1.5">
              {chosen.apps.map((named) => {
                const selected = named.id === app;
                return (
                  <label
                    className={cn(
                      "relative flex min-h-11 cursor-pointer items-center rounded-control px-3.5 font-ui text-ui transition-colors duration-200 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent has-[:focus-visible]:outline-offset-2",
                      selected
                        ? "font-medium text-accent"
                        : "text-ink hover:bg-deep",
                    )}
                    key={named.id}
                  >
                    {selected ? (
                      <motion.span
                        className="absolute inset-0 rounded-control bg-accent-wash"
                        layoutId={still ? undefined : `draft-app-${type}`}
                        transition={{
                          type: "spring",
                          stiffness: 420,
                          damping: 34,
                        }}
                      />
                    ) : null}
                    <input
                      checked={selected}
                      className="sr-only"
                      name={`app-${type}`}
                      onChange={() =>
                        setApps((current) => ({
                          ...current,
                          [type]: named.id,
                        }))
                      }
                      type="radio"
                    />
                    <span className="relative">{named.label}</span>
                  </label>
                );
              })}
            </div>
          </fieldset>
        ) : null}

        <Button
          className="w-full"
          loading={pending}
          onClick={() => void start()}
          variant="primary"
        >
          {pending
            ? "Starting…"
            : `Start ${appLabel ? `a ${appLabel}` : "a"} ${TYPE_LABELS[type].toLowerCase()}`}
          {pending ? null : (
            <ArrowRight aria-hidden="true" className="size-4" />
          )}
        </Button>
        {message ? (
          <p className="text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
      </div>
    </div>
  );
}
