import Link from "next/link";
import { shellClasses } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { Butterflies } from "./Butterflies";

/** HomeHero is the watcher sending butterflies across the night, with the line and the two ways in over the dark side of the picture. */
export function HomeHero() {
  return (
    <section className={cn(shellClasses, "pt-3")}>
      <div className="relative isolate flex min-h-[clamp(28rem,calc(100svh-var(--header-height)-17rem),42rem)] overflow-hidden rounded-card bg-inset text-over max-sm:min-h-[34rem] max-sm:items-end [html[data-artwork=off]_&]:text-ink">
        <div
          aria-hidden="true"
          className="absolute inset-0 -z-10 bg-media bg-[url(/home/watcher-butterflies.webp)] bg-cover bg-[position:70%_18%] max-sm:bg-[position:78%_0%]"
          data-artwork
        >
          <div className="absolute inset-0 bg-linear-to-r from-media/85 via-media/35 via-45% to-transparent max-sm:bg-linear-to-t max-sm:from-media max-sm:via-media/60 max-sm:via-40%" />
          <Butterflies />
        </div>
        <div className="flex max-w-[36rem] flex-col justify-center p-8 sm:p-14">
          <h1 className="font-display text-display font-medium tracking-tight text-balance">
            Everything for your roleplay app
          </h1>
          <p className="mt-4 max-w-[30rem] text-lede text-over-mute [html[data-artwork=off]_&]:text-mute">
            Characters, lorebooks, presets, themes and extensions, in the format
            your app reads.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Button asChild variant="primary">
              <Link href="/browse">Browse</Link>
            </Button>
            <Button asChild>
              <Link href="/upload">Publish</Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
