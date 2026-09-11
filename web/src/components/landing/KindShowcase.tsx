import { ArrowUpRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { LineLink } from "@/components/ui/line-link";
import { Reveal } from "./LandingMotion";

const TOOLS = [
  {
    kind: "preset",
    label: "Presets",
    title: "Shape the model’s replies.",
    description: "Prompt fragments and model settings for your roleplay app.",
  },
  {
    kind: "theme",
    label: "Themes",
    title: "Customise your app.",
    description: "Colours and styles for supported roleplay apps.",
  },
  {
    kind: "pack",
    label: "Lumia packs",
    title: "Add Lumia to Lumiverse.",
    description: "Collections of Lumia personalities, definitions and avatars.",
  },
] as const;

export function KindShowcase() {
  return (
    <section
      id="explore"
      aria-labelledby="landing-kinds-title"
      className="relative scroll-mt-28 py-20 sm:py-28"
    >
      <Shell>
        <Reveal className="mb-12 grid items-end gap-6 sm:mb-16 lg:grid-cols-[1.2fr_.8fr] lg:gap-24">
          <h2
            id="landing-kinds-title"
            className="max-w-[760px] font-display text-[clamp(2.8rem,5vw,5.3rem)] leading-[1.03] font-medium tracking-[-.045em]"
          >
            Characters, worlds
            <br />
            and roleplay tools.
          </h2>
          <p className="max-w-[370px] text-lede text-mute">
            Find assets for your roleplay app, with previews and compatible
            downloads.
          </p>
        </Reveal>
        <div className="grid items-start gap-10 md:grid-cols-[1.12fr_1fr] md:gap-8 lg:gap-12">
          <Reveal>
            <Link
              href="/browse?kind=character"
              className="group block rounded-control text-ink outline-offset-8"
            >
              <div className="relative aspect-[1.13] overflow-hidden rounded-plate bg-[#110f17]">
                <Image
                  src="/landing/watcher-portrait.webp"
                  alt=""
                  fill
                  sizes="(max-width: 767px) 100vw, 50vw"
                  className="object-contain object-bottom transition-transform duration-700 group-hover:scale-[1.035] group-focus-visible:scale-[1.035] motion-reduce:transform-none motion-reduce:transition-none"
                />
                <span className="absolute top-5 left-5 rounded-full bg-[#09080d] px-4 py-2 font-ui text-meta text-[#ffffff]">
                  Characters
                </span>
                <span className="absolute right-5 bottom-5 flex size-12 items-center justify-center rounded-full bg-[#ffffff] text-[#090909] transition-transform group-hover:-rotate-45 motion-reduce:transform-none">
                  <ArrowUpRight aria-hidden="true" />
                </span>
              </div>
              <h3 className="mt-6 font-display text-[clamp(1.8rem,2.7vw,2.8rem)] leading-tight font-medium tracking-tight group-hover:text-accent">
                Find a character.
              </h3>
              <p className="mt-3 max-w-[450px] text-prose text-mute">
                Preview a character’s description, personality and greetings
                before downloading.
              </p>
            </Link>
          </Reveal>
          <Reveal delay={0.08} className="md:pt-24">
            <Link
              href="/browse?kind=lorebook"
              className="group block rounded-control text-ink outline-offset-8"
            >
              <div className="relative aspect-[1.12] overflow-hidden rounded-plate bg-[#110f17]">
                <Image
                  src="/landing/moon.webp"
                  alt=""
                  fill
                  sizes="(max-width: 767px) 100vw, 45vw"
                  className="object-cover object-[65%_center] transition-transform duration-700 group-hover:scale-[1.035] group-focus-visible:scale-[1.035] motion-reduce:transform-none motion-reduce:transition-none"
                />
                <span className="absolute top-5 left-5 rounded-full bg-[#09080d] px-4 py-2 font-ui text-meta text-[#ffffff]">
                  Lorebooks
                </span>
                <span className="absolute right-5 bottom-5 flex size-12 items-center justify-center rounded-full bg-[#ffffff] text-[#090909] transition-transform group-hover:-rotate-45 motion-reduce:transform-none">
                  <ArrowUpRight aria-hidden="true" />
                </span>
              </div>
              <h3 className="mt-6 font-display text-[clamp(1.8rem,2.7vw,2.8rem)] leading-tight font-medium tracking-tight group-hover:text-accent">
                Build your story’s world.
              </h3>
              <p className="mt-3 max-w-[450px] text-prose text-mute">
                Lorebook entries add context about places, people and events
                when their keys match a conversation.
              </p>
            </Link>
          </Reveal>
        </div>
        <div className="mt-16 grid gap-8 border-t border-rule pt-10 md:mt-24 md:grid-cols-3 md:gap-10">
          {TOOLS.map((item, index) => (
            <Reveal key={item.kind} delay={index * 0.06}>
              <LineLink
                href={`/browse?kind=${item.kind}`}
                className="min-h-11 font-ui text-ui text-accent"
              >
                {item.label}
                <ArrowUpRight className="ml-3 size-4" aria-hidden="true" />
              </LineLink>
              <h3 className="mt-3 font-display text-title font-medium tracking-tight">
                {item.title}
              </h3>
              <p className="mt-3 max-w-[340px] text-prose text-mute">
                {item.description}
              </p>
            </Reveal>
          ))}
        </div>
        <Reveal className="mt-16 flex flex-col justify-between gap-6 rounded-plate bg-deep px-7 py-8 sm:flex-row sm:items-center sm:px-10">
          <p className="max-w-[660px] text-prose text-mute">
            <span className="font-medium text-ink">
              Download or send to your app.
            </span>{" "}
            Read a creation, choose an available download format, or send it to
            a compatible linked application.
          </p>
          <LineLink href="/browse" className="min-h-11 shrink-0 text-ink">
            Browse the catalog
            <ArrowUpRight className="ml-3 size-4" aria-hidden="true" />
          </LineLink>
        </Reveal>
      </Shell>
    </section>
  );
}
