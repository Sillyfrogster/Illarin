"use client";

import { AnimatePresence, motion } from "framer-motion";
import { ArrowRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import { Button } from "@/components/ui/button";
import { RadioGroup } from "@/components/ui/radio-group";
import { type BrowseType, startWork } from "@/lib/api/query";
import type { BuildChoice, BuildChoices } from "@/lib/api/shapes";
import { useAuth } from "@/lib/auth";
import { timing } from "@/lib/timing";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { DraftPreview } from "./DraftPreview";

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
        Your draft opens with this page.
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
      <RadioGroup
        aria-label="Type"
        onValueChange={(next) =>
          setChosen(choices.find((choice) => choice.type === next) ?? chosen)
        }
        options={choices.map((choice) => ({
          value: choice.type,
          label: TYPE_LABELS[choice.type as BrowseType],
          media: (
            <span className="relative block aspect-[5/6] w-6 shrink-0 overflow-hidden rounded-[5px]">
              <DefaultCover compact type={choice.type as BrowseType} />
            </span>
          ),
        }))}
        value={chosen.type}
      />

      <AnimatePresence initial={false} mode="wait">
        <motion.div
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -6, transition: timing.quick }}
          className="min-w-0 max-lg:order-3"
          initial={{ opacity: 0, y: 10 }}
          key={chosen.type}
          transition={timing.quick}
        >
          {draft ? <DraftPreview blocks={draft.blocks} type={type} /> : null}
        </motion.div>
      </AnimatePresence>

      <div className="grid min-w-0 content-start gap-5 max-lg:order-2 lg:sticky lg:top-[calc(var(--header-height)+2rem)]">
        {chosen.apps.length > 0 ? (
          <fieldset className="min-w-0">
            <legend className="text-meta font-medium text-ink">App</legend>
            <p className="mt-0.5 text-label text-mute">
              Presets and themes are made for one app.
            </p>
            <RadioGroup
              className="mt-2.5"
              onValueChange={(next) =>
                setApps((current) => ({ ...current, [type]: next }))
              }
              options={chosen.apps.map((named) => ({
                value: named.id,
                label: named.label,
              }))}
              value={app}
            />
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
