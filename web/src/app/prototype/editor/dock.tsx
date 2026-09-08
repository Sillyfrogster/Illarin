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
import { cn, SpectralButton } from "./ui";

export type SaveState = "saving" | "unsaved" | "private" | "published";

const STATUS_LABEL: Record<SaveState, string> = {
  saving: "Saving…",
  unsaved: "Unsaved changes",
  private: "Saved privately",
  published: "Everything is published",
};

/** Amber while work is private, spectral once readers have it */
function StatusLight({ state }: { state: SaveState }) {
  const reduced = useReducedMotion();
  return (
    <span className="ws:relative ws:flex ws:size-2.5 ws:shrink-0">
      {state === "unsaved" && !reduced && (
        <motion.span
          animate={{ scale: [1, 2.1], opacity: [0.5, 0] }}
          transition={{ duration: 1.9, repeat: Number.POSITIVE_INFINITY }}
          className="ws:absolute ws:inset-0 ws:rounded-full ws:bg-amber"
        />
      )}
      <span
        className={cn(
          "ws:relative ws:size-2.5 ws:rounded-full",
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
}: {
  icon: LucideIcon;
  label: string;
  badge?: number;
  active?: boolean;
  pointer: ReturnType<typeof useMotionValue<number>>;
  onClick: () => void;
}) {
  const reduced = useReducedMotion();
  const ref = useRef<HTMLButtonElement>(null);
  const distance = useTransform(pointer, (x) => {
    if (!ref.current || x === Number.POSITIVE_INFINITY) return 999;
    const box = ref.current.getBoundingClientRect();
    return Math.abs(x - (box.left + box.width / 2));
  });
  const raw = useTransform(distance, [0, 44, 110], [1.28, 1.1, 1]);
  const scale = useSpring(raw, { stiffness: 340, damping: 22, mass: 0.4 });
  return (
    <motion.button
      type="button"
      ref={ref}
      onClick={onClick}
      title={label}
      aria-label={label}
      aria-pressed={active}
      style={reduced ? undefined : { scale }}
      className={cn(
        "ws:relative ws:inline-flex ws:size-11 ws:shrink-0 ws:origin-bottom ws:items-center ws:justify-center ws:rounded-2xl ws:text-ink ws:transition-colors ws:hover:bg-ink/10 ws:motion-reduce:transition-none",
        active && "ws:bg-ink/12",
      )}
    >
      <Icon className="ws:size-[1.15rem]" />
      {badge !== undefined && badge > 0 && (
        <span className="ws:absolute ws:-top-0.5 ws:-right-0.5 ws:inline-flex ws:min-w-5 ws:items-center ws:justify-center ws:rounded-full ws:bg-amber ws:px-1 ws:text-[0.625rem] ws:font-bold ws:text-paper">
          <AnimatedNumber value={badge} />
        </span>
      )}
    </motion.button>
  );
}

export type DockItem = {
  icon: LucideIcon;
  label: string;
  badge?: number;
  active?: boolean;
  onClick: () => void;
};

export function Dock({
  state,
  detail,
  items,
  onSave,
  onReview,
  reviewLabel,
  busy,
  readOnly,
}: {
  state: SaveState;
  detail: string;
  items: DockItem[];
  onSave: () => void;
  onReview: () => void;
  reviewLabel: string;
  busy: boolean;
  readOnly: boolean;
}) {
  const pointer = useMotionValue(Number.POSITIVE_INFINITY);
  return (
    <div className="ws:pointer-events-none ws:fixed ws:inset-x-0 ws:bottom-0 ws:z-30 ws:flex ws:justify-center ws:px-3 ws:pb-3 ws:md:pb-6">
      <motion.div
        initial={{ y: 90, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: 90, opacity: 0 }}
        transition={{ duration: 0.45, ease: [0.22, 1, 0.36, 1], delay: 0.12 }}
        onMouseMove={(event) => pointer.set(event.clientX)}
        onMouseLeave={() => pointer.set(Number.POSITIVE_INFINITY)}
        className="w-glass ws:pointer-events-auto ws:flex ws:w-full ws:max-w-[54rem] ws:items-center ws:gap-2 ws:rounded-[26px] ws:p-2 ws:md:gap-3 ws:md:p-2.5"
      >
        <div
          className="ws:flex ws:min-w-0 ws:items-center ws:gap-2.5 ws:pl-2.5"
          aria-live="polite"
        >
          <StatusLight state={state} />
          <span className="ws:hidden ws:min-w-0 ws:leading-4 ws:lg:block">
            <span className="ws:block ws:truncate ws:text-[0.8125rem] ws:font-semibold">
              {STATUS_LABEL[state]}
            </span>
            <span className="ws:block ws:truncate ws:text-[0.6875rem] ws:text-mute">
              {detail}
            </span>
          </span>
          <span className="ws:sr-only ws:lg:hidden">{STATUS_LABEL[state]}</span>
        </div>

        <span
          aria-hidden="true"
          className="ws:mx-0.5 ws:h-7 ws:w-px ws:shrink-0 ws:bg-line"
        />

        <div className="ws:flex ws:min-w-0 ws:flex-1 ws:items-center ws:justify-center ws:gap-0.5 ws:md:gap-1">
          {items.map((item) => (
            <DockButton key={item.label} pointer={pointer} {...item} />
          ))}
        </div>

        <div className="ws:flex ws:shrink-0 ws:items-center ws:gap-1.5">
          <button
            type="button"
            onClick={onSave}
            disabled={busy || readOnly}
            className="ws:hidden ws:min-h-11 ws:items-center ws:rounded-full ws:px-4 ws:text-sm ws:font-semibold ws:text-ink ws:transition ws:hover:bg-ink/10 ws:disabled:opacity-40 ws:sm:inline-flex ws:motion-reduce:transition-none"
          >
            Save
          </button>
          <SpectralButton
            onClick={onReview}
            disabled={busy || readOnly}
            className="ws:px-4 ws:md:px-6"
          >
            {reviewLabel}
          </SpectralButton>
        </div>
      </motion.div>
    </div>
  );
}
