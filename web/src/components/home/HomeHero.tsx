import type { CSSProperties } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";
import { Aside, Lead } from "./Cta";
import "./home.css";

/** HomeHero is the watcher's picture filling the first screen, with the headline and the page's one main action set in its dark side. */
export function HomeHero() {
  return (
    <section className="relative isolate flex min-h-[calc(100svh-var(--header-height))] overflow-hidden bg-media text-over">
      <div
        aria-hidden="true"
        className="home-settle absolute inset-0 -z-10 bg-[url(/home/watcher-hero.webp)] bg-cover bg-[position:70%_20%] max-md:bg-[url(/home/watcher-portrait.webp)] max-md:bg-[position:60%_15%]"
        data-artwork
      />
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-linear-to-r from-media/85 via-media/35 to-transparent max-md:bg-linear-to-t max-md:from-media max-md:via-media/60"
      />
      <div
        className={cn(
          shellClasses,
          "flex flex-col justify-end gap-8 pt-24 pb-16 sm:pb-20",
        )}
      >
        <h1 className="font-display text-[clamp(2.5rem,5.2vw,4.5rem)] leading-[1.02] font-medium tracking-[-0.04em]">
          <span className="home-line">
            <span style={{ "--line": 0 } as CSSProperties}>One place for</span>
          </span>
          <span className="home-line">
            <span style={{ "--line": 1 } as CSSProperties}>
              every roleplay app
            </span>
          </span>
        </h1>
        <div className="home-after grid gap-8">
          <p className="max-w-[42ch] text-lede text-over-mute">
            Characters, lorebooks, presets, themes and extensions, ready in the
            format SillyTavern, RisuAI or Lumiverse reads.
          </p>
          <div className="flex flex-wrap items-center gap-x-8 gap-y-2">
            <Lead href="/browse">Browse works</Lead>
            <Aside href="/upload">Publish yours</Aside>
          </div>
        </div>
      </div>
    </section>
  );
}
