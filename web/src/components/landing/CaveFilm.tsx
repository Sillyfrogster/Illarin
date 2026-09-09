"use client";

import type { MotionValue } from "framer-motion";
import { useEffect, useRef, useState } from "react";
import cave from "../../../public/landing/cave.json";
import { useLandingMotion } from "./LandingMotion";

export function CaveFilm({ progress }: { progress: MotionValue<number> }) {
  const { live, showStill } = useLandingMotion();
  const video = useRef<HTMLVideoElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const element = video.current;
    if (!live || !element) return;
    let intersecting = false;
    let frame = 0;
    let watchdog = 0;
    let seekStarted = 0;
    let slowSeeks = 0;
    let disposed = false;

    function sync() {
      frame = 0;
      if (
        !element ||
        disposed ||
        !intersecting ||
        document.hidden ||
        element.seeking ||
        element.readyState < 2
      )
        return;
      const duration = element.duration;
      if (!Number.isFinite(duration)) return;
      const target =
        Math.min(1, Math.max(0, progress.get())) *
        Math.max(0, duration - 1 / cave.fps);
      if (Math.abs(element.currentTime - target) < 1 / cave.fps) return;
      seekStarted = performance.now();
      element.currentTime = target;
      window.clearTimeout(watchdog);
      watchdog = window.setTimeout(() => {
        if (!document.hidden && intersecting) showStill();
      }, 4000);
    }

    function schedule() {
      if (!frame && !disposed) frame = requestAnimationFrame(sync);
    }

    function seeked() {
      window.clearTimeout(watchdog);
      if (document.hidden || !intersecting) return;
      slowSeeks = performance.now() - seekStarted > 600 ? slowSeeks + 1 : 0;
      if (slowSeeks >= 5) showStill();
      else schedule();
    }

    function loaded() {
      window.clearTimeout(watchdog);
      setVisible(true);
      schedule();
    }

    const observer = new IntersectionObserver(([entry]) => {
      intersecting = entry.isIntersecting;
      if (intersecting && !element.src) {
        element.src = matchMedia("(max-width: 767px)").matches
          ? "/landing/cave-mobile.mp4"
          : "/landing/cave-desktop.mp4";
        element.load();
        watchdog = window.setTimeout(showStill, 12000);
      }
      schedule();
    });
    observer.observe(element);
    element.addEventListener("loadeddata", loaded);
    element.addEventListener("seeked", seeked);
    element.addEventListener("error", showStill);
    document.addEventListener("visibilitychange", schedule);
    const unsubscribe = progress.on("change", schedule);
    return () => {
      disposed = true;
      observer.disconnect();
      unsubscribe();
      cancelAnimationFrame(frame);
      window.clearTimeout(watchdog);
      document.removeEventListener("visibilitychange", schedule);
      element.removeEventListener("loadeddata", loaded);
      element.removeEventListener("seeked", seeked);
      element.removeEventListener("error", showStill);
      element.removeAttribute("src");
      element.load();
      setVisible(false);
    };
  }, [live, progress, showStill]);

  if (!live) return null;
  return (
    <video
      ref={video}
      muted
      playsInline
      preload="auto"
      aria-hidden="true"
      tabIndex={-1}
      className="absolute inset-0 h-full w-full object-cover object-[51%_center]"
      style={{ opacity: visible ? 1 : 0 }}
    />
  );
}
