"use client";

import { LayoutPanelTop, PenLine } from "lucide-react";
import {
  type CSSProperties,
  type ReactNode,
  useCallback,
  useEffect,
} from "react";
import { flushSync } from "react-dom";
import { cn, focusRing } from "@/lib/cn";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import { type Altitude, useWorkspace } from "./state";

/** The arrange map's scale: half size on a wide screen, a little larger on a tablet, and none on a phone, where blocks are always full width. */
function mapZoom(width: number): number {
  if (width >= 1180) return 0.5;
  if (width >= 768) return 0.6;
  return 1;
}

function blockNearMiddle(): string | null {
  const middle = window.innerHeight / 2;
  let best: { id: string; distance: number } | null = null;
  for (const node of document.querySelectorAll<HTMLElement>(
    "[data-arrange-id]",
  )) {
    const box = node.getBoundingClientRect();
    const distance =
      box.top <= middle && box.bottom >= middle
        ? 0
        : Math.min(Math.abs(box.top - middle), Math.abs(box.bottom - middle));
    if (!best || distance < best.distance)
      best = { distance, id: node.dataset.arrangeId ?? "" };
  }
  return best && best.distance < window.innerHeight / 2 ? best.id : null;
}

/** useAltitudeShift moves between writing and the arrange map, morphing every block between its two sizes and keeping the block in view that the creator was looking at. */
export function useAltitudeShift() {
  const workspace = useWorkspace();
  const { altitude, setAltitude } = workspace;
  return useCallback(
    (next: Altitude, focus?: string) => {
      if (altitude === next) return;
      const anchor = focus ?? blockNearMiddle();
      const flip = () => {
        flushSync(() => setAltitude(next));
        const node = anchor ? document.getElementById(`block-${anchor}`) : null;
        if (!node) {
          window.scrollTo({ top: 0 });
          return;
        }
        if (next === "write") {
          node.scrollIntoView({ block: "start" });
          return;
        }
        const box = node.getBoundingClientRect();
        window.scrollBy({
          top:
            box.top +
            Math.min(box.height, window.innerHeight) / 2 -
            window.innerHeight / 2,
        });
      };
      const reduced = window.matchMedia(
        "(prefers-reduced-motion: reduce)",
      ).matches;
      if (!document.startViewTransition || reduced) {
        flip();
        return;
      }
      document.startViewTransition(flip);
    },
    [altitude, setAltitude],
  );
}

/** EditCanvas lays the page out as the reader sees it and, on the arrange map, shrinks the whole of it so every block fits in a glance. */
export function EditCanvas({ children }: { children: ReactNode }) {
  const workspace = useWorkspace();
  const shift = useAltitudeShift();
  const [measure, width] = useMeasuredWidth<HTMLDivElement>();
  const mapped =
    workspace.editing &&
    workspace.look === "altitudes" &&
    workspace.altitude === "arrange";
  const zoom = mapped && width ? mapZoom(width) : 1;

  useEffect(() => {
    if (!mapped) return;
    const back = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented) return;
      if (document.querySelector('[role="dialog"], [role="menu"]')) return;
      event.preventDefault();
      event.stopImmediatePropagation();
      shift("write");
    };
    window.addEventListener("keydown", back, true);
    return () => window.removeEventListener("keydown", back, true);
  }, [mapped, shift]);

  return (
    <div
      className={cn(
        mapped &&
          "bg-inset bg-[radial-gradient(var(--v-rule)_1px,transparent_1px)] [background-size:24px_24px] py-10",
      )}
      data-altitude={mapped ? "arrange" : undefined}
      ref={measure}
    >
      {mapped ? (
        <p className="mx-auto mb-8 max-w-[var(--shell)] px-[var(--gutter)] text-meta text-mute">
          Drag a block to move it, or drag its right edge to resize it. Click a
          block to write in it.
        </p>
      ) : null}
      <div
        className={cn(mapped && zoom < 1 && "mx-auto")}
        style={
          mapped && zoom < 1
            ? ({
                "--map-zoom": zoom,
                width: `${width}px`,
                zoom,
              } as CSSProperties)
            : undefined
        }
      >
        {children}
      </div>
    </div>
  );
}

/** AltitudeSwitch is the dock's choice between writing on the page and seeing the page as a map to arrange. */
export function AltitudeSwitch({ compact = false }: { compact?: boolean }) {
  const workspace = useWorkspace();
  const shift = useAltitudeShift();
  const options: { value: Altitude; label: string; icon: typeof PenLine }[] = [
    { icon: PenLine, label: "Write", value: "write" },
    { icon: LayoutPanelTop, label: "Arrange", value: "arrange" },
  ];
  return (
    <fieldset
      aria-label="How you edit the page"
      className="relative m-0 inline-flex shrink-0 items-center gap-0.5 rounded-control border-0 bg-fill p-1"
    >
      {options.map(({ icon: Icon, label, value }) => {
        const chosen = workspace.altitude === value;
        return (
          <button
            aria-pressed={chosen}
            className={cn(
              "inline-flex h-8 items-center gap-1.5 rounded-chip px-3 text-ui font-medium transition-colors duration-100",
              chosen ? "bg-action text-on-accent" : "text-mute hover:text-ink",
              compact && "px-2.5",
              focusRing,
              "focus-visible:ring-offset-1 focus-visible:ring-offset-fill",
            )}
            key={value}
            onClick={() => shift(value)}
            type="button"
          >
            <Icon aria-hidden="true" className="size-4" />
            <span className={cn(compact && "max-sm:sr-only")}>{label}</span>
          </button>
        );
      })}
    </fieldset>
  );
}
