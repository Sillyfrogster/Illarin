"use client";

import { SiGithub } from "@icons-pack/react-simple-icons";
import * as RadioGroupPrimitive from "@radix-ui/react-radio-group";
import { ArrowRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Segmented } from "@/components/ui/segmented";
import { type BrowseType, startWork } from "@/lib/api/query";
import type { BuildChoice, BuildChoices } from "@/lib/api/shapes";
import { cn, focusRing } from "@/lib/cn";
import { TYPE_LABELS, WORK_TYPES } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { FromGitHub } from "./FromGitHub";

/** StartFromNothing lets a creator pick a type, and the app where the type needs one, and names what the draft opens with; an extension starts from GitHub instead. */
export function StartFromNothing({
  choices,
}: {
  choices: BuildChoices | null;
}) {
  const buildable = WORK_TYPES.flatMap(
    (type) => choices?.types.filter((choice) => choice.type === type) ?? [],
  );

  return (
    <section
      aria-labelledby="start-heading"
      className="flex min-w-0 flex-col rounded-card bg-inset p-6"
    >
      <h2
        className="font-display text-section font-medium text-ink"
        id="start-heading"
      >
        Start a new draft
      </h2>
      <p className="mt-1.5 text-ui text-mute">
        Pick a type. Your draft opens in the editor.
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
  const [type, setType] = useState(choices[0].type as BrowseType);
  const [apps, setApps] = useState<Record<string, string>>({});
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  const chosen = choices.find((choice) => choice.type === type);
  const app = chosen?.apps.length ? (apps[type] ?? chosen.apps[0].id) : "";
  const appLabel = chosen?.apps.find((named) => named.id === app)?.label;
  const draft =
    chosen?.drafts.find((candidate) => candidate.app === app) ??
    chosen?.drafts[0];
  const parts = [
    ...new Set(
      draft?.blocks.flatMap((block) =>
        block.elements.map((element) => element.label),
      ),
    ),
  ];

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
    <>
      <RadioGroupPrimitive.Root
        aria-label="Type"
        className="mt-5 grid grid-cols-3 gap-2"
        onValueChange={(next) => {
          setType(next as BrowseType);
          setMessage("");
        }}
        value={type}
      >
        {[
          ...choices.map((choice) => choice.type as BrowseType),
          "extension" as const,
        ].map((tile) => (
          <RadioGroupPrimitive.Item
            className={cn(
              "group/tile flex cursor-pointer flex-col items-start gap-3 rounded-plate bg-field p-3 text-left font-ui text-ui text-ink ring-1 ring-transparent transition-[background-color,box-shadow] duration-80 hover:ring-rule data-[state=checked]:bg-accent-wash data-[state=checked]:ring-accent",
              focusRing,
            )}
            key={tile}
            value={tile}
          >
            <span className="flex size-9 items-center justify-center rounded-control bg-fill text-mute transition-colors duration-80 group-data-[state=checked]/tile:bg-action group-data-[state=checked]/tile:text-on-accent">
              <TypeMark className="size-[1.125rem]" type={tile} />
            </span>
            <span className="max-w-full truncate">{TYPE_LABELS[tile]}</span>
          </RadioGroupPrimitive.Item>
        ))}
      </RadioGroupPrimitive.Root>

      <div className="mt-auto flex flex-col gap-5 pt-7">
        {chosen ? (
          <>
            <div>
              <h3 className="text-meta font-medium text-ink">Opens with</h3>
              <ul className="mt-2 flex list-none flex-wrap gap-1.5 p-0">
                {parts.map((part) => (
                  <li key={part}>
                    <Badge className="bg-field">{part}</Badge>
                  </li>
                ))}
              </ul>
            </div>
            {chosen.apps.length > 0 ? (
              <div>
                <h3 className="text-meta font-medium text-ink" id="start-app">
                  App
                </h3>
                <Segmented
                  aria-labelledby="start-app"
                  className="mt-2 w-full"
                  onValueChange={(next) =>
                    setApps((current) => ({ ...current, [type]: next }))
                  }
                  options={chosen.apps.map((named) => ({
                    value: named.id,
                    label: named.label,
                  }))}
                  value={app}
                />
              </div>
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
          </>
        ) : (
          <>
            <p className="text-ui text-mute">
              Its latest GitHub release becomes your draft, and later releases
              arrive as new versions. Have the zip? Upload it instead.
            </p>
            <FromGitHub>
              <Button className="w-full" variant="primary">
                <SiGithub aria-hidden="true" />
                Start from GitHub
              </Button>
            </FromGitHub>
          </>
        )}
      </div>
    </>
  );
}
