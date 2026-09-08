"use client";

import {
  motion,
  useMotionValue,
  useReducedMotion,
  useSpring,
  useTransform,
} from "framer-motion";
import type { LucideIcon } from "lucide-react";
import { useRef } from "react";
import { AnimatedNumber } from "./animated-number";
import { cn } from "./ui";

export type SaveState = "saving" | "unsaved" | "private" | "published";

const STATUS_LABEL: Record<SaveState, string> = {
  saving: "Saving",
  unsaved: "Unsaved",
  private: "Saved privately",
  published: "Published",
};

export type DockItem = {
  icon: LucideIcon;
  label: string;
  badge?: number;
  active?: boolean;
  onClick: () => void;
};

/** Amber while work is private, spectral once readers have it */
function StatusLight({ state }: { state: SaveState }) {
  const reduced = useReducedMotion();
  return (
    <span className="ws:relative ws:flex ws:size-2 ws:shrink-0">
      {state === "unsaved" && !reduced && (
        <motion.span
          animate={{ scale: [1, 2.4], opacity: [0.55, 0] }}
          transition={{ duration: 2, repeat: Number.POSITIVE_INFINITY }}
          className="ws:absolute ws:inset-0 ws:rounded-full ws:bg-amber"
        />
      )}
      <span
        className={cn(
          "ws:relative ws:size-2 ws:rounded-full",
          state === "published"
            ? ""
            : state === "private"
              ? "ws:shadow-[inset_0_0_0_2px_var(--w-amber)]"
              : "ws:bg-amber",
        )}
        style={
          state === "published"
            ? { background: "var(--w-spectrum)" }
            : undefined
        }
      />
    </span>
  );
}

function DockButton({
  icon: Icon,
  label,
  badge,
  active,
  pointer,
  onClick,
}: DockItem & { pointer: ReturnType<typeof useMotionValue<number>> }) {
  const reduced = useReducedMotion();
  const ref = useRef<HTMLButtonElement>(null);
  const distance = useTransform(pointer, (x) => {
    if (!ref.current || x === Number.POSITIVE_INFINITY) return 999;
    const box = ref.current.getBoundingClientRect();
    return Math.abs(x - (box.left + box.width / 2));
  });
  const raw = useTransform(distance, [0, 40, 108], [1.18, 1.07, 1]);
  const scale = useSpring(raw, { stiffness: 360, damping: 24, mass: 0.4 });
  return (
    <motion.button
      type="button"
      ref={ref}
      onClick={onClick}
      aria-label={label}
      aria-pressed={active}
      style={reduced ? undefined : { scale }}
      className={cn(
        "ws:group ws:relative ws:inline-flex ws:size-11 ws:shrink-0 ws:origin-bottom ws:items-center ws:justify-center ws:rounded-xl ws:text-[color:var(--w-on-bar)] ws:opacity-65 ws:transition ws:hover:bg-[color:var(--w-on-bar)]/15 ws:hover:opacity-100 ws:focus-visible:opacity-100 ws:motion-reduce:transition-none",
        active && "ws:bg-[color:var(--w-on-bar)]/15 ws:opacity-100",
      )}
    >
      <Icon className="ws:size-[1.15rem]" />
      {badge !== undefined && badge > 0 && (
        <span className="ws:absolute ws:top-1 ws:right-1 ws:inline-flex ws:min-w-4 ws:justify-center ws:rounded-full ws:bg-amber ws:px-1 ws:text-[0.625rem] ws:font-bold ws:text-paper">
          <AnimatedNumber value={badge} />
        </span>
      )}
      <span
        role="tooltip"
        className="ws:pointer-events-none ws:absolute ws:bottom-[calc(100%+0.7rem)] ws:left-1/2 ws:z-10 ws:-translate-x-1/2 ws:rounded-lg ws:bg-[color:var(--w-on-bar)] ws:px-2.5 ws:py-1.5 ws:text-xs ws:font-semibold ws:whitespace-nowrap ws:text-[color:var(--w-bar)] ws:opacity-0 ws:transition ws:group-hover:opacity-100 ws:group-focus-visible:opacity-100 ws:motion-reduce:transition-none"
      >
        {label}
      </span>
    </motion.button>
  );
}

export function Dock({
  state,
  detail,
  items,
  onSave,
  onReview,
  reviewLabel,
  busy,
  readOnly,
  shifted,
}: {
  state: SaveState;
  detail: string;
  items: DockItem[];
  onSave: () => void;
  onReview: () => void;
  reviewLabel: string;
  busy: boolean;
  readOnly: boolean;
  shifted: boolean;
}) {
  const pointer = useMotionValue(Number.POSITIVE_INFINITY);
  return (
    <div
      className={cn(
        "ws:pointer-events-none ws:fixed ws:inset-x-0 ws:bottom-0 ws:z-30 ws:flex ws:justify-center ws:px-3 ws:pb-4 ws:transition-[padding] ws:duration-500 ws:ease-[cubic-bezier(0.22,1,0.36,1)] ws:md:pb-7 ws:motion-reduce:transition-none",
        shifted && "ws:max-lg:hidden ws:lg:pr-[28rem]",
      )}
    >
      <motion.div
        initial={{ y: 96, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: 96, opacity: 0 }}
        transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1], delay: 0.14 }}
        onMouseMove={(event) => pointer.set(event.clientX)}
        onMouseLeave={() => pointer.set(Number.POSITIVE_INFINITY)}
        className="w-bar ws:pointer-events-auto ws:flex ws:w-full ws:max-w-[50rem] ws:items-center ws:gap-1 ws:rounded-[20px] ws:p-1.5 ws:md:gap-2 ws:md:rounded-[22px] ws:md:p-2"
      >
        <div
          className="ws:flex ws:min-w-0 ws:items-center ws:gap-2.5 ws:pr-1 ws:pl-3"
          aria-live="polite"
        >
          <StatusLight state={state} />
          <span className="ws:hidden ws:min-w-0 ws:leading-4 ws:lg:block">
            <span className="ws:block ws:truncate ws:text-[0.8125rem] ws:font-semibold">
              {STATUS_LABEL[state]}
            </span>
            <span className="ws:block ws:truncate ws:text-[0.6875rem] ws:text-[color:var(--w-bar-mute)]">
              {detail}
            </span>
          </span>
          <span className="ws:sr-only ws:lg:hidden">
            {STATUS_LABEL[state]}. {detail}
          </span>
        </div>

        <div className="ws:flex ws:min-w-0 ws:flex-1 ws:items-center ws:justify-center ws:gap-0.5">
          {items.map((item) => (
            <DockButton key={item.label} pointer={pointer} {...item} />
          ))}
        </div>

        <div className="ws:flex ws:shrink-0 ws:items-center ws:gap-1">
          <button
            type="button"
            onClick={onSave}
            disabled={busy || readOnly}
            className="ws:hidden ws:min-h-11 ws:items-center ws:rounded-xl ws:px-3.5 ws:text-sm ws:font-semibold ws:text-[color:var(--w-on-bar)] ws:opacity-75 ws:transition ws:hover:bg-[color:var(--w-on-bar)]/15 ws:hover:opacity-100 ws:disabled:opacity-35 ws:sm:inline-flex ws:motion-reduce:transition-none"
          >
            Save
          </button>
          <button
            type="button"
            onClick={onReview}
            disabled={busy || readOnly}
            className="ws:relative ws:inline-flex ws:min-h-11 ws:shrink-0 ws:items-center ws:gap-2 ws:overflow-hidden ws:rounded-xl ws:bg-[color:var(--w-on-bar)] ws:px-5 ws:text-sm ws:font-semibold ws:text-[color:var(--w-bar)] ws:transition ws:hover:opacity-90 ws:disabled:opacity-35 ws:motion-reduce:transition-none"
          >
            <span
              aria-hidden="true"
              className="ws:absolute ws:inset-x-0 ws:bottom-0 ws:h-[2px]"
              style={{ background: "var(--w-spectrum)" }}
            />
            {reviewLabel}
          </button>
        </div>
      </motion.div>
    </div>
  );
}
