"use client";

import { motion, useInView } from "framer-motion";
import { Pause, Play } from "lucide-react";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";

const PREFERENCE = "illarin.landing-motion.v1";

type Connection = EventTarget & {
  saveData?: boolean;
  effectiveType?: string;
};

type MotionState = {
  live: boolean;
  reduced: boolean;
  ready: boolean;
  fallback: boolean;
  toggle: () => void;
  showStill: () => void;
};

const LandingMotionContext = createContext<MotionState | null>(null);

export function LandingMotion({ children }: { children: ReactNode }) {
  const scrollAnchor = useRef<{ element: Element; top: number } | null>(null);
  const rememberPosition = useCallback(() => {
    const element = document
      .elementFromPoint(window.innerWidth / 2, 160)
      ?.closest("section");
    if (element)
      scrollAnchor.current = {
        element,
        top: element.getBoundingClientRect().top,
      };
  }, []);
  const [preference, setPreference] = useState<string | null>(null);
  const [reduced, setReduced] = useState(true);
  const [limited, setLimited] = useState(true);
  const [ready, setReady] = useState(false);
  const [fallback, setFallback] = useState(false);

  useEffect(() => {
    const media = matchMedia("(prefers-reduced-motion: reduce)");
    const device = navigator as Navigator & {
      deviceMemory?: number;
      connection?: Connection;
    };
    function update() {
      rememberPosition();
      setReduced(media.matches);
      setLimited(
        Boolean(
          device.connection?.saveData ||
            /^(slow-)?2g$/.test(device.connection?.effectiveType ?? "") ||
            (device.deviceMemory && device.deviceMemory <= 4) ||
            (device.hardwareConcurrency && device.hardwareConcurrency <= 4),
        ),
      );
    }
    try {
      setPreference(localStorage.getItem(PREFERENCE));
    } catch {
      setPreference(null);
    }
    update();
    setReady(true);
    media.addEventListener("change", update);
    device.connection?.addEventListener("change", update);
    return () => {
      media.removeEventListener("change", update);
      device.connection?.removeEventListener("change", update);
    };
  }, [rememberPosition]);

  const live =
    ready &&
    !reduced &&
    !fallback &&
    (preference === "live" || (preference !== "still" && !limited));
  useLayoutEffect(() => {
    const anchor = scrollAnchor.current;
    scrollAnchor.current = null;
    if (!anchor) return;
    if (anchor.element.hasAttribute("data-cave-journey")) {
      if (!live && anchor.top < 0)
        window.scrollTo({
          top: window.scrollY + anchor.element.getBoundingClientRect().top - 72,
          behavior: "instant",
        });
    } else {
      window.scrollBy({
        top: anchor.element.getBoundingClientRect().top - anchor.top,
        behavior: "instant",
      });
    }
  }, [live]);
  const showStill = useCallback(() => {
    rememberPosition();
    setFallback(true);
  }, [rememberPosition]);
  const toggle = useCallback(() => {
    rememberPosition();
    const next = live ? "still" : "live";
    setPreference(next);
    setFallback(false);
    try {
      localStorage.setItem(PREFERENCE, next);
    } catch {}
  }, [live, rememberPosition]);
  const value = useMemo(
    () => ({ live, reduced, ready, fallback, toggle, showStill }),
    [live, reduced, ready, fallback, toggle, showStill],
  );

  return (
    <LandingMotionContext.Provider value={value}>
      <div
        className="group/landing"
        data-landing-motion={live ? "live" : "still"}
      >
        {children}
      </div>
    </LandingMotionContext.Provider>
  );
}

export function useLandingMotion() {
  const value = useContext(LandingMotionContext);
  if (!value) throw new Error("Landing motion needs its provider");
  return value;
}

export function MotionControl() {
  const { live, reduced, ready, fallback, toggle } = useLandingMotion();
  return (
    <div className="fixed right-4 bottom-4 z-40 flex items-center gap-2 rounded-full bg-plane p-1 text-meta text-ink shadow-cover sm:right-6 sm:bottom-6">
      <span className="sr-only" aria-live="polite">
        {fallback ? "Animations paused for performance" : "Page animations"}
      </span>
      <Button
        variant="ghost"
        size="compact"
        disabled={!ready || reduced}
        onClick={toggle}
        aria-label={live ? "Pause page animations" : "Play page animations"}
        aria-pressed={live}
        className="rounded-full text-meta disabled:opacity-100"
      >
        {live ? <Pause aria-hidden="true" /> : <Play aria-hidden="true" />}
        {ready && reduced ? "Reduced motion" : live ? "Playing" : "Paused"}
      </Button>
    </div>
  );
}

export function Reveal({
  children,
  className,
  delay = 0,
}: {
  children: ReactNode;
  className?: string;
  delay?: number;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, amount: 0.12 });
  const { live } = useLandingMotion();
  return (
    <motion.div
      ref={ref}
      className={className}
      initial={{ opacity: 1, y: 0 }}
      animate={live && !inView ? { opacity: 1, y: 12 } : { opacity: 1, y: 0 }}
      transition={
        live
          ? { type: "spring", stiffness: 170, damping: 25, delay }
          : { duration: 0 }
      }
    >
      {children}
    </motion.div>
  );
}
