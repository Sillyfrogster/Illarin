import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";
import { Aside, Lead } from "./Cta";
import { HEADING } from "./heading";
import "./home.css";

const STEPS = [
  [
    "Upload the file you have",
    "A character card, lorebook, preset, theme or extension, as it is.",
  ],
  [
    "It stays as you made it",
    "Illarin never rewrites your file. What the page shows sits beside it.",
  ],
  [
    "Readers get their version",
    "Each download is made from your file, in the format their app reads.",
  ],
];

/** PublishBand tells creators what publishing does, the heading on one side and the three steps on the other. */
export function PublishBand() {
  return (
    <section
      aria-labelledby="home-publish"
      className={cn(shellClasses, "grid gap-section pt-chapter lg:grid-cols-2")}
    >
      <div className="flex flex-col gap-6">
        <h2 className={cn(HEADING, "max-w-[18ch]")} id="home-publish">
          Publish once, for every app that reads it
        </h2>
        <p className="max-w-[40ch] text-lede text-mute">
          One upload reaches readers on SillyTavern, RisuAI and Lumiverse.
        </p>
      </div>
      <ol className="m-0 list-none border-b border-rule p-0">
        {STEPS.map(([title, line], at) => (
          <li
            className="grid grid-cols-[2.5rem_minmax(0,1fr)] gap-x-4 border-t border-rule py-6"
            key={title}
          >
            <span className="text-meta text-accent tabular-nums">
              0{at + 1}
            </span>
            <div className="flex flex-col gap-1.5">
              <span className="text-lede font-medium text-ink">{title}</span>
              <span className="text-ui text-mute">{line}</span>
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}

/** Closing is the watcher at her window across the full width, with the page's last call to browse or publish in the night sky. */
export function Closing() {
  return (
    <section
      aria-labelledby="home-close"
      className="relative isolate mt-chapter flex min-h-[min(80svh,52rem)] overflow-hidden bg-media text-over"
    >
      <div
        aria-hidden="true"
        className="home-drift absolute inset-0 -z-10 bg-[url(/home/watcher-window.webp)] bg-cover bg-[position:60%_12%] max-md:bg-[position:66%_10%]"
        data-artwork
      />
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-linear-to-b from-media/80 via-media/20 to-media/60"
      />
      <div className={cn(shellClasses, "flex flex-col gap-8 pt-section")}>
        <h2
          className="max-w-[12ch] font-display text-[clamp(2.25rem,4.4vw,3.75rem)] leading-[1.02] font-medium tracking-[-0.04em]"
          id="home-close"
        >
          Find your next favorite
        </h2>
        <div className="flex flex-wrap items-center gap-x-8 gap-y-2">
          <Lead href="/browse">Browse works</Lead>
          <Aside href="/upload">Publish yours</Aside>
        </div>
      </div>
    </section>
  );
}
