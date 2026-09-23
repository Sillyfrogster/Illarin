"use client";

import { ArrowUpRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { type BrowseType, startWork } from "@/lib/api/query";
import type { BuildChoices } from "@/lib/api/shapes";
import { cn } from "@/lib/cn";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";

const TYPE =
  "group flex min-h-14 items-center gap-3 rounded-control bg-inset px-4 font-ui text-ui font-medium text-ink outline-offset-3 transition-colors duration-200 hover:bg-accent-wash hover:text-accent disabled:opacity-45 motion-reduce:transition-none";

/** StartFromNothing offers each type the API can build from nothing, in the site's type order, asking for an app where the type needs one. */
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
    setAsking(null);
    setMessage("");
    try {
      const started = await startWork(type, app);
      router.push(workHref(started.id, started.name));
    } catch {
      setPending(null);
      setMessage("The draft could not be started. Try again.");
    }
  }

  return (
    <section
      aria-labelledby="start-heading"
      className="border-t border-rule pt-8"
    >
      <h2
        className="font-display text-section font-medium text-ink"
        id="start-heading"
      >
        Create an empty draft
      </h2>
      <p className="mt-2 text-ui text-mute">
        Choose a type, then add content in the editor. You can import a file
        later.
      </p>

      {choices === null ? (
        <p className="mt-5 text-meta text-mute" role="alert">
          The types you can start could not be loaded. Reload the page to try
          again.
        </p>
      ) : null}
      <div className="mt-5 grid grid-cols-2 gap-2 sm:grid-cols-3">
        {buildable.map(({ type: listed, apps }) => {
          const type = listed as BrowseType;
          return apps.length > 0 ? (
            <Popover
              key={type}
              onOpenChange={(open) => setAsking(open ? type : null)}
              open={asking === type}
            >
              <PopoverTrigger
                className={TYPE}
                disabled={pending !== null}
                type="button"
              >
                <TypeLabel type={type} pending={pending === type} />
              </PopoverTrigger>
              <PopoverContent
                align="start"
                className="w-[min(24rem,calc(100vw-2rem))]"
              >
                <p className="text-meta text-mute">
                  Which app is this {TYPE_LABELS[type].toLowerCase()} for? This
                  sets the fields available in the editor.
                </p>
                <div className="mt-4 flex flex-wrap gap-2">
                  {apps.map((app) => (
                    <button
                      className={cn(
                        TYPE,
                        "bg-accent-wash hover:bg-accent-wash/70",
                      )}
                      key={app.id}
                      onClick={() => void start(type, app.id)}
                      type="button"
                    >
                      {app.label}
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          ) : (
            <button
              className={TYPE}
              disabled={pending !== null}
              key={type}
              onClick={() => void start(type)}
              type="button"
            >
              <TypeLabel type={type} pending={pending === type} />
            </button>
          );
        })}
      </div>

      {message ? (
        <p
          className="mt-4 rounded-control bg-stop-wash p-3 text-meta text-ink"
          role="alert"
        >
          {message}
        </p>
      ) : null}
    </section>
  );
}

function TypeLabel({ type, pending }: { type: BrowseType; pending: boolean }) {
  return (
    <>
      <TypeMark className="size-4 shrink-0 text-accent" type={type} />
      {TYPE_LABELS[type]}
      {pending ? <span className="text-meta text-mute">starting…</span> : null}
      <ArrowUpRight
        aria-hidden="true"
        className="ml-auto size-4 shrink-0 text-mute transition-transform duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 motion-reduce:transform-none"
      />
    </>
  );
}
