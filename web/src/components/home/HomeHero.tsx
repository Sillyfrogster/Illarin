import type { CSSProperties } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";
import { Aside, Lead } from "./Cta";
import "./home.css";

/** HomeHero is the headline and the page's one main action beside the watcher, whose picture runs the hero's full height and fades into it. */
export function HomeHero() {
  return (
    <section className="relative isolate overflow-hidden bg-media text-over">
      <div
        aria-hidden="true"
        className="home-settle absolute inset-y-0 right-0 -z-10 aspect-video bg-[url(/home/watcher-hero.webp)] bg-cover bg-top mask-[linear-gradient(to_right,transparent,#000_35%)] max-md:inset-x-0 max-md:aspect-auto max-md:bg-[position:71%_0%] max-md:mask-[linear-gradient(to_bottom,#000_45%,transparent_85%)]"
        data-artwork
      />
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-linear-to-r from-media/80 via-media/45 via-35% to-transparent to-60% max-md:hidden"
      />
      <div
        className={cn(
          shellClasses,
          "flex min-h-[min(calc(100svh-var(--header-height)-5rem),46rem)] flex-col justify-center gap-8 py-section max-md:min-h-[calc(100svh-var(--header-height))] max-md:justify-end",
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
