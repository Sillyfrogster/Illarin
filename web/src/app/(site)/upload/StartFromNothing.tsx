"use client";

import { ArrowUpRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { KindMark } from "@/components/catalog/KindMark";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  type BrowseKind,
  type StartAssetApp,
  startAsset,
} from "@/lib/api/query";
import { assetHref } from "@/lib/asset-url";
import { cn } from "@/lib/cn";
import {
  APP_CHOICES,
  BUILDABLE_KINDS,
  KIND_LABELS,
  KINDS_ASKING_FOR_AN_APP,
} from "@/lib/kinds";

const KIND =
  "group flex min-h-14 items-center gap-3 rounded-control bg-inset px-4 font-ui text-ui font-medium text-ink outline-offset-3 transition-colors duration-200 hover:bg-accent-wash hover:text-accent disabled:opacity-45 motion-reduce:transition-none";

export function StartFromNothing() {
  const router = useRouter();
  const [pending, setPending] = useState<BrowseKind | null>(null);
  const [asking, setAsking] = useState<BrowseKind | null>(null);
  const [message, setMessage] = useState("");

  async function start(kind: BrowseKind, app?: StartAssetApp) {
    setPending(kind);
    setAsking(null);
    setMessage("");
    try {
      const started = await startAsset(kind, app);
      router.push(assetHref(started.id, started.name));
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
        Choose an asset kind, then add content in the editor. You can import a
        file later.
      </p>

      <div className="mt-5 grid grid-cols-2 gap-2 sm:grid-cols-3">
        {BUILDABLE_KINDS.map((kind) =>
          KINDS_ASKING_FOR_AN_APP.includes(kind) ? (
            <Popover
              key={kind}
              onOpenChange={(open) => setAsking(open ? kind : null)}
              open={asking === kind}
            >
              <PopoverTrigger
                className={KIND}
                disabled={pending !== null}
                type="button"
              >
                <KindLabel kind={kind} pending={pending === kind} />
              </PopoverTrigger>
              <PopoverContent
                align="start"
                className="w-[min(24rem,calc(100vw-2rem))]"
              >
                <p className="text-meta text-mute">
                  Which app is this {KIND_LABELS[kind].toLowerCase()} for? This
                  sets the fields available in the editor.
                </p>
                <div className="mt-4 flex flex-wrap gap-2">
                  {APP_CHOICES.map((app) => (
                    <button
                      className={cn(
                        KIND,
                        "bg-accent-wash hover:bg-accent-wash/70",
                      )}
                      key={app.value}
                      onClick={() => void start(kind, app.value)}
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
              className={KIND}
              disabled={pending !== null}
              key={kind}
              onClick={() => void start(kind)}
              type="button"
            >
              <KindLabel kind={kind} pending={pending === kind} />
            </button>
          ),
        )}
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

function KindLabel({ kind, pending }: { kind: BrowseKind; pending: boolean }) {
  return (
    <>
      <KindMark className="size-4 shrink-0 text-accent" kind={kind} />
      {KIND_LABELS[kind]}
      {pending ? <span className="text-meta text-mute">starting…</span> : null}
      <ArrowUpRight
        aria-hidden="true"
        className="ml-auto size-4 shrink-0 text-mute transition-transform duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 motion-reduce:transform-none"
      />
    </>
  );
}
