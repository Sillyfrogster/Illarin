import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { createWorld, smooth } from "./world";

const TIMES = [0, 26, 54, 73, 100];
const PREFERENCE = "illarin.flight.motion";
const ASSETS = {
  room: "room-start",
  kingdom: "kingdom",
  city: "city",
  homecoming: "room-finished",
  butterfly: "butterfly",
  gateway: "gate",
  gallery: "gallery",
  distance: "kingdom-distance",
  mark: "mark",
};

export function startJourney(
  root: HTMLElement,
  onMode: (mode: string) => void,
) {
  gsap.registerPlugin(ScrollTrigger);
  const chapters = Array.from(root.querySelectorAll<HTMLElement>(".chapter"));
  const links = Array.from(
    root.querySelectorAll<HTMLAnchorElement>(".chapters a"),
  );
  const stage = root.querySelector<HTMLElement>(".stage");
  const canvas = root.querySelector<HTMLCanvasElement>("canvas");
  const veil = root.querySelector<HTMLElement>(".arrival");
  const instruction = root.querySelector<HTMLElement>("#instruction");
  const exhibition = root.querySelector<HTMLElement>(".gallery-works");
  const media = matchMedia("(prefers-reduced-motion: reduce)");
  const events = new AbortController();
  let renderer: ReturnType<typeof createWorld> | undefined;
  let trigger: ScrollTrigger | undefined;
  let timeline: gsap.core.Timeline | undefined;
  let loading: Promise<void> | undefined;
  let live = false,
    disposed = false,
    wanted = true,
    active = 0;
  let ambient = 0;
  let lastDraw = -Infinity;
  let redrawUntil = 0;
  const state = { progress: 0 };
  try {
    wanted =
      localStorage.getItem(PREFERENCE) !== "off" &&
      localStorage.getItem("illarin.landing-motion.v1") !== "still";
  } catch {}

  function update() {
    if (!live) return;
    redrawUntil = gsap.ticker.time + 1.5;
    const p = state.progress;
    active = p < 0.2 ? 0 : p < 0.485 ? 1 : p < 0.64 ? 2 : p < 0.855 ? 3 : 4;
    if (root.dataset.chapter === String(active)) return;
    root.dataset.chapter = String(active);
    chapters.forEach((chapter, i) => {
      if (chapter.contains(document.activeElement) && i !== active) {
        root
          .querySelector<HTMLButtonElement>(".motion-control")
          ?.focus({ preventScroll: true });
      }
      chapter.inert = i !== active;
      chapter.setAttribute("aria-hidden", String(i !== active));
    });
    links.forEach((link, i) => {
      if (i === active) link.setAttribute("aria-current", "step");
      else link.removeAttribute("aria-current");
    });
    if (instruction)
      instruction.textContent =
        active === 4
          ? "Replay journey"
          : active === 3
            ? "Continue the story"
            : "Follow the butterflies";
  }

  function draw(time: number) {
    if (
      !live ||
      document.hidden ||
      time > redrawUntil ||
      time - lastDraw < 1 / 60
    )
      return;
    if (lastDraw !== -Infinity) ambient += Math.min(time - lastDraw, 0.1);
    lastDraw = time;
    renderer?.render(state.progress, ambient, smooth(0, 2.2, ambient));
    if (veil) veil.style.opacity = String(1 - smooth(0, 1.6, ambient));
  }

  function go(index: number, behavior: ScrollBehavior = "smooth") {
    if (live && trigger) {
      window.scrollTo({
        top:
          trigger.start + ((trigger.end - trigger.start) * TIMES[index]) / 100,
        behavior,
      });
    } else
      chapters[index].scrollIntoView({ behavior: "instant", block: "start" });
  }

  function setMode(enable: boolean, preserve = true) {
    if (disposed) return;
    if (enable && !renderer && !media.matches) {
      load();
      return;
    }
    const previous = active;
    live = enable && !!renderer && !media.matches;
    trigger?.kill();
    trigger = undefined;
    timeline?.revert();
    timeline = undefined;
    gsap.ticker.remove(draw);
    root.classList.toggle("motion", live);
    onMode(media.matches ? "reduced" : live ? "live" : "still");
    if (live) {
      ambient = 0;
      lastDraw = -Infinity;
      redrawUntil = gsap.ticker.time + 4;
      root.dataset.chapter = "";
      state.progress = 0;
      timeline = gsap.timeline({ paused: true, onUpdate: update });
      timeline.to(state, { progress: 1, duration: 100, ease: "none" }, 0);
      [
        [0, 10],
        [22, 32],
        [49, 58],
        [67, 80],
        [96, 101],
      ].forEach(([start, end], i) => {
        if (i > 0)
          timeline?.fromTo(
            chapters[i],
            { autoAlpha: 0, y: 24 },
            {
              autoAlpha: 1,
              y: 0,
              duration: 4,
              ease: "power2.out",
              immediateRender: false,
            },
            start,
          );
        if (i < 4)
          timeline?.to(
            chapters[i],
            { autoAlpha: 0, y: -18, duration: 4, ease: "power2.in" },
            end,
          );
      });
      timeline.fromTo(
        exhibition,
        { y: 42, scale: 0.88 },
        {
          y: 0,
          scale: 1,
          duration: 6,
          ease: "power3.out",
          immediateRender: false,
        },
        66,
      );
      timeline.to(
        exhibition,
        { y: 35, scale: 1.12, duration: 5, ease: "power2.in" },
        80,
      );
      timeline.fromTo(
        root.querySelector(".chapter-line i"),
        { scaleX: 0 },
        { scaleX: 1, duration: 100, ease: "none" },
        0,
      );
      trigger = ScrollTrigger.create({
        trigger: root.querySelector("#journey"),
        start: "top top",
        end: "bottom bottom",
        animation: timeline,
        scrub: 1.15,
      });
      renderer?.resize();
      if (preserve) go(previous, "instant");
      gsap.ticker.add(draw);
      update();
    } else {
      chapters.forEach((chapter) => {
        chapter.inert = false;
        chapter.removeAttribute("aria-hidden");
      });
      if (preserve) go(previous, "instant");
    }
  }

  function load() {
    root.classList.add("motion");
    onMode("loading");
    if (loading) return;
    loading = Promise.all(
      Object.entries(ASSETS).map(async ([name, file]) => {
        const image = new Image();
        image.src = `/landing/flight/${file}.${name === "mark" ? "svg" : "webp"}`;
        await image.decode();
        return [name, image] as const;
      }),
    )
      .then((entries) => {
        if (disposed || !canvas) return;
        renderer = createWorld(canvas, Object.fromEntries(entries));
        setMode(wanted, false);
      })
      .catch(() => {
        if (disposed) return;
        setMode(false, false);
        onMode("unavailable");
      });
  }

  function toggle() {
    wanted = !wanted;
    try {
      localStorage.setItem(PREFERENCE, wanted ? "on" : "off");
      localStorage.removeItem("illarin.landing-motion.v1");
    } catch {}
    setMode(wanted);
  }

  root.addEventListener(
    "click",
    (event) => {
      const element = event.target instanceof Element ? event.target : null;
      const link = element?.closest<HTMLAnchorElement>(".chapters a");
      if (link) {
        event.preventDefault();
        go(links.indexOf(link));
      }
      if (element?.closest(".replay")) go(0);
      if (element?.closest(".follow")) go(active === 4 ? 0 : active + 1);
    },
    { signal: events.signal },
  );
  media.addEventListener("change", () => setMode(wanted), {
    signal: events.signal,
  });
  document.addEventListener(
    "visibilitychange",
    () => {
      if (!document.hidden) redrawUntil = gsap.ticker.time + 1.5;
    },
    { signal: events.signal },
  );
  document
    .querySelector('a[href="#main-content"]')
    ?.addEventListener("click", () => setMode(false), {
      signal: events.signal,
    });
  const observer = new ResizeObserver(() => {
    if (live) {
      renderer?.resize();
      redrawUntil = gsap.ticker.time + 1.5;
    }
  });
  if (stage) observer.observe(stage);
  setMode(wanted, false);

  return {
    toggle,
    dispose() {
      disposed = true;
      live = false;
      events.abort();
      observer.disconnect();
      gsap.ticker.remove(draw);
      trigger?.kill();
      timeline?.revert();
      root.classList.remove("motion");
    },
  };
}
