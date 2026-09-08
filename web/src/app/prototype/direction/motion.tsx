"use client";

import {
  AnimatePresence,
  animate,
  motion,
  useInView,
  useReducedMotion,
} from "framer-motion";
import Image, { type StaticImageData } from "next/image";
import {
  type CSSProperties,
  Fragment,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";
import { cn } from "./ui";

const EASE = [0.22, 1, 0.36, 1] as const;

/** A title arrives a word at a time, drawn in CSS so it is whole with or without scripting */
export function RiseText({
  text,
  className,
  delay = 0,
}: {
  text: string;
  className?: string;
  delay?: number;
}) {
  const words = text.split(" ");
  return (
    <span className={cn("v-rise", className)}>
      {words.map((word, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: the word list is fixed for a given title
        <Fragment key={index}>
          <span className="vd:relative vd:inline-block vd:overflow-hidden vd:align-top">
            <span
              style={{ animationDelay: `${delay + index * 0.035}s` }}
              className="vd:inline-block vd:will-change-transform"
            >
              {word}
            </span>
          </span>
          {index < words.length - 1 ? " " : ""}
        </Fragment>
      ))}
    </span>
  );
}

/** A light runs the edge of a plate once it is on screen, the way the artwork's beam runs its glass */
export function EdgeBeam({
  className,
  seconds = 9,
}: {
  className?: string;
  seconds?: number;
}) {
  return (
    <span
      aria-hidden="true"
      style={{ "--beam-seconds": `${seconds}s` } as CSSProperties}
      className={cn("v-beam vd:pointer-events-none vd:absolute", className)}
    />
  );
}

/** The medium a text index still shows its media through */
export function SpineMedia({
  image,
  alt,
  fallback,
}: {
  image?: StaticImageData;
  alt: string;
  fallback: ReactNode;
}) {
  const reduced = useReducedMotion();
  return (
    <div className="vd:relative vd:aspect-4/5 vd:w-full vd:overflow-hidden vd:rounded-plate vd:bg-plane vd:shadow-[inset_0_0_0_1px_var(--v-hair)]">
      <AnimatePresence mode="wait" initial={false}>
        {image ? (
          <motion.div
            key={image.src}
            initial={{ opacity: 0, scale: 1.04 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: reduced ? 0.01 : 0.45, ease: EASE }}
            className="vd:absolute vd:inset-0"
          >
            <Image
              src={image}
              alt={alt}
              fill
              sizes="320px"
              className="vd:object-cover"
            />
          </motion.div>
        ) : (
          <motion.div
            key="fallback"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: reduced ? 0.01 : 0.35 }}
            className="vd:absolute vd:inset-0 vd:flex vd:items-end vd:p-5"
          >
            {fallback}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

/** A count that runs up when it is first read, and reads as a plain number otherwise */
export function Counter({
  value,
  className,
}: {
  value: number;
  className?: string;
}) {
  const reduced = useReducedMotion();
  const ref = useRef<HTMLSpanElement>(null);
  const seen = useInView(ref, { once: true, amount: 0.4 });
  const [shown, setShown] = useState(value);

  useEffect(() => {
    if (reduced || !seen || value === 0) {
      setShown(value);
      return;
    }
    const run = animate(0, value, {
      duration: Math.min(0.25 + value * 0.06, 1.1),
      ease: EASE,
      onUpdate: (next) => setShown(Math.round(next)),
    });
    return () => run.stop();
  }, [reduced, seen, value]);

  return (
    <span ref={ref} className={cn("v-tabular", className)}>
      {shown}
    </span>
  );
}

/** The room a page borrows from its media, drifting slowly and holding still under reduced motion */
export function Room({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("v-room vd:relative", className)}>
      <div
        aria-hidden="true"
        className="v-drift vd:pointer-events-none vd:absolute vd:inset-0 vd:overflow-hidden"
      />
      <div className="vd:relative">{children}</div>
    </div>
  );
}

/** A dock that stays put and grows the control under the pointer, without moving its neighbours */
export function Dock({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  const reduced = useReducedMotion();
  return (
    <motion.div
      initial={{ opacity: 0, y: 26 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: reduced ? 0.01 : 0.5, ease: EASE, delay: 0.2 }}
      className={cn(
        "v-glass vd:fixed vd:inset-x-0 vd:bottom-0 vd:z-40 vd:mx-auto vd:flex vd:w-fit vd:max-w-[calc(100vw-1.5rem)] vd:items-center vd:gap-1 vd:overflow-x-auto vd:rounded-t-2xl vd:px-2 vd:py-2 vd:sm:bottom-5 vd:sm:rounded-full",
        className,
      )}
    >
      {children}
    </motion.div>
  );
}

/** Content that arrives as the reader reaches it, and shows anyway if no observer ever fires */
export function Arrive({
  children,
  className,
  distance = 20,
}: {
  children: ReactNode;
  className?: string;
  distance?: number;
}) {
  const reduced = useReducedMotion();
  const ref = useRef<HTMLDivElement>(null);
  const seen = useInView(ref, { once: true, amount: 0.12 });
  const [late, setLate] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setLate(true), 1200);
    return () => clearTimeout(timer);
  }, []);

  const shown = reduced || seen || late;
  return (
    <motion.div
      ref={ref}
      initial={false}
      animate={shown ? { opacity: 1, y: 0 } : { opacity: 0, y: distance }}
      transition={{ duration: reduced ? 0.01 : 0.65, ease: EASE }}
      className={className}
    >
      {children}
    </motion.div>
  );
}
