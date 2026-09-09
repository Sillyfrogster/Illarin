"use client";

import { motion, useScroll, useTransform } from "framer-motion";
import { ArrowDown, ArrowUpRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useRef } from "react";
import { Button } from "@/components/ui/button";
import { CaveFilm } from "./CaveFilm";
import { useLandingMotion } from "./LandingMotion";

export function EncounterHero() {
  const root = useRef<HTMLElement>(null);
  const { live } = useLandingMotion();
  const { scrollYProgress } = useScroll({
    target: root,
    offset: ["start 72px", "end end"],
  });
  const introOpacity = useTransform(
    scrollYProgress,
    [0, 0.18, 0.35],
    [1, 1, 0],
  );
  const introY = useTransform(scrollYProgress, [0, 0.35], [0, -60]);
  const endOpacity = useTransform(scrollYProgress, [0.65, 0.85], [0, 1]);
  const endY = useTransform(scrollYProgress, [0.65, 0.9], [30, 0]);

  return (
    <section
      ref={root}
      aria-labelledby="landing-title"
      data-cave-journey
      data-live={live}
      className="relative bg-[#09080d] text-[#ffffff] data-[live=true]:h-[260svh] motion-reduce:!h-auto"
    >
      <div className="sticky top-[72px] isolate flex h-[max(700px,calc(100svh-72px))] flex-col overflow-hidden sm:h-[max(680px,calc(100svh-72px))]">
        <div className="absolute inset-0">
          <Image
            src="/landing/cave-entry.webp"
            alt=""
            fill
            preload
            unoptimized
            sizes="100vw"
            className="object-cover object-[51%_center]"
          />
        </div>
        <CaveFilm progress={scrollYProgress} />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,#09080d55_0%,transparent_35%,#09080d44_55%,#09080df2_100%)]"
        />
        <div className="relative z-10 flex justify-between gap-8 px-[var(--gutter)] pt-12 sm:pt-14">
          <p className="max-w-[210px] font-ui text-ui leading-relaxed text-[#ffffff]/80">
            A home for the
            <br />
            AI roleplay imagination.
          </p>
          <a
            href="#explore"
            className="inline-flex min-h-11 items-center gap-3 self-start font-ui text-meta text-[#ffffff] hover:text-[#d3baff]"
          >
            Skip to the good stuff{" "}
            <ArrowDown className="size-4" aria-hidden="true" />
          </a>
        </div>
        <div className="relative z-10 mt-auto px-[var(--gutter)] pb-8 sm:pb-10">
          <div className="relative mb-8 sm:mb-10">
            <motion.div
              style={live ? { opacity: introOpacity, y: introY } : undefined}
            >
              <p className="mb-4 font-ui text-ui text-[#d3baff]">
                For the beautifully curious.
              </p>
              <h1
                id="landing-title"
                className="max-w-[1200px] font-display text-[clamp(3rem,15vw,3.7rem)] leading-[.96] font-medium tracking-[-.055em] sm:text-[clamp(3.7rem,8.7vw,9.5rem)]"
              >
                Follow your
                <br />
                imagination.
              </h1>
            </motion.div>
            <motion.p
              aria-hidden="true"
              style={{ opacity: live ? endOpacity : 0, y: live ? endY : 0 }}
              className="pointer-events-none absolute bottom-0 left-0 font-display text-[clamp(2.7rem,13vw,3.4rem)] leading-[.98] font-medium tracking-[-.05em] sm:text-[clamp(3.4rem,7.5vw,8rem)]"
            >
              There’s a whole
              <br />
              world in here.
            </motion.p>
          </div>
          <div className="flex flex-col justify-between gap-6 border-t border-[#ffffff]/30 pt-6 lg:flex-row lg:items-end">
            <p className="max-w-[400px] text-prose leading-relaxed text-[#ffffff]/85">
              Meet a character. Get lost in their world. Discover what other
              minds have made, and share a little of yours.
            </p>
            <div className="flex flex-wrap items-center gap-3 lg:pb-1">
              <Button
                asChild
                variant="primary"
                size="large"
                className="rounded-full px-6"
              >
                <Link href="/browse">
                  Explore Illarin <ArrowUpRight aria-hidden="true" />
                </Link>
              </Button>
              <Button
                asChild
                size="large"
                variant="ghost"
                className="rounded-full text-[#ffffff] hover:bg-[#ffffff]/10 hover:text-[#ffffff]"
              >
                <Link href="/upload">Share your work</Link>
              </Button>
            </div>
          </div>
        </div>
        <motion.div
          aria-hidden="true"
          className="absolute inset-x-0 bottom-0 z-20 h-[3px] origin-left bg-[#b89aff]"
          style={{ scaleX: live ? scrollYProgress : 0 }}
        />
      </div>
    </section>
  );
}
