"use client";

import { ChevronDown } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { Button } from "@/components/ui/button";
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

const CHOICE =
  "flex min-h-11 items-center gap-2 rounded-control px-3 font-ui text-ui text-ink outline-offset-2 transition-colors duration-150 hover:bg-accent-wash hover:text-accent disabled:opacity-45 motion-reduce:transition-none";

/** StartFromNothing offers each type the API can build from nothing, from one menu, naming the app where a type needs one. */
export function StartFromNothing({
  choices,
}: {
  choices: BuildChoices | null;
}) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  const buildable = (Object.keys(TYPE_LABELS) as BrowseType[]).flatMap(
    (type) => choices?.types.filter((choice) => choice.type === type) ?? [],
  );

  async function start(type: BrowseType, app?: string) {
    setPending(true);
    setMessage("");
    try {
      const started = await startWork(type, app);
      router.push(workHref(started.id, started.name));
    } catch {
      setPending(false);
      setMessage("Illarin could not start the draft. Try again.");
    }
  }

  if (choices === null) {
    return (
      <p className="mt-5 text-meta text-mute" role="alert">
        The draft types could not load. Reload the page to try again.
      </p>
    );
  }

  return (
    <div className="mt-5 flex flex-wrap items-center gap-3">
      <Popover>
        <PopoverTrigger asChild>
          <Button loading={pending} variant="ghost">
            {pending ? "Starting…" : "Start an empty draft"}
            <ChevronDown aria-hidden="true" className="size-4" />
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align="start"
          className="w-[min(22rem,calc(100vw-2rem))] p-2"
        >
          <ul className="grid list-none gap-0.5 p-0">
            {buildable.map(({ type: listed, apps }) => {
              const type = listed as BrowseType;
              return (
                <li key={type}>
                  {apps.length === 0 ? (
                    <button
                      className={cn(CHOICE, "w-full")}
                      disabled={pending}
                      onClick={() => void start(type)}
                      type="button"
                    >
                      <TypeMark
                        className="size-4 shrink-0 text-accent"
                        type={type}
                      />
                      {TYPE_LABELS[type]}
                    </button>
                  ) : (
                    <div className="flex flex-wrap items-center gap-x-1 pl-3">
                      <span className="flex min-h-11 items-center gap-2 font-ui text-ui text-ink">
                        <TypeMark
                          className="size-4 shrink-0 text-accent"
                          type={type}
                        />
                        {TYPE_LABELS[type]} for
                      </span>
                      {apps.map((app) => (
                        <button
                          className={cn(CHOICE, "font-medium text-accent")}
                          disabled={pending}
                          key={app.id}
                          onClick={() => void start(type, app.id)}
                          type="button"
                        >
                          {app.label}
                        </button>
                      ))}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        </PopoverContent>
      </Popover>

      {message ? (
        <p className="text-meta text-stop" role="alert">
          {message}
        </p>
      ) : null}
    </div>
  );
}
