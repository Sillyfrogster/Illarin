"use client";

import { useScroll } from "framer-motion";
import { useEffect, useRef } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { cn } from "@/lib/cn";

const WORDS =
  "Illarin gathers characters, lorebooks, presets, themes and extensions from across the hobby. Each file stays the way its creator made it, and every download comes ready for the app you use.".split(
    " ",
  );

/** Statement says what Illarin is, lit word by word as the reader moves down. */
export function Statement() {
  const text = useRef<HTMLParagraphElement>(null);
  const { scrollYProgress } = useScroll({
    target: text,
    offset: ["start 85%", "end 50%"],
  });

  useEffect(
    () =>
      scrollYProgress.on("change", (progress) =>
        text.current?.style.setProperty(
          "--lit",
          String(progress * WORDS.length),
        ),
      ),
    [scrollYProgress],
  );

  return (
    <section
      aria-label="What Illarin is"
      className={cn(shellClasses, "pt-chapter")}
    >
      <p
        className="max-w-[46ch] font-display text-[clamp(1.5rem,2.5vw,2.25rem)] leading-[1.25] font-medium tracking-[-0.025em] text-ink [--lit:0]"
        ref={text}
      >
        {WORDS.map((word, at) => (
          <span
            className="transition-opacity duration-150 motion-reduce:!opacity-100"
            // biome-ignore lint/suspicious/noArrayIndexKey: the words never reorder
            key={at}
            style={{ opacity: `clamp(0.22, calc(var(--lit) - ${at}), 1)` }}
          >
            {word}{" "}
          </span>
        ))}
      </p>
    </section>
  );
}
