"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import { useEffect, useState } from "react";
import { TimedUndo } from "@/components/ui/timed-undo";
import { cn } from "@/lib/cn";

export type Toast =
  | { kind: "placed"; pieceIds: string[]; name: string }
  | { kind: "said"; text: string; tone?: "stop" };

const UNDO_SECONDS = 10;
const SAID_MS = 6000;

/** ShelfToast says what the shelf just did, and holds the undo for a placement until its seconds run out. */
export function ShelfToast({
  onClose,
  onUndo,
  toast,
}: {
  onClose: () => void;
  onUndo: (pieceIds: string[], name: string) => Promise<void>;
  toast: Toast | null;
}) {
  const still = useReducedMotion();
  const [undoing, setUndoing] = useState(false);

  useEffect(() => {
    if (toast?.kind !== "said") return;
    const timer = window.setTimeout(onClose, SAID_MS);
    return () => window.clearTimeout(timer);
  }, [toast, onClose]);

  return (
    <AnimatePresence>
      {toast ? (
        <motion.output
          animate={{ opacity: 1, y: 0, scale: 1, filter: "blur(0px)" }}
          className={cn(
            "fixed inset-x-4 bottom-28 z-50 mx-auto flex max-w-lg items-center gap-3 rounded-plate bg-plane py-2 pr-2 pl-5 text-meta shadow-popover md:bottom-32",
            toast.kind === "said" && toast.tone === "stop"
              ? "text-stop"
              : "text-ink",
          )}
          exit={{ opacity: 0, y: still ? 0 : 10, filter: "blur(2px)" }}
          initial={
            still
              ? { opacity: 0 }
              : { opacity: 0, y: 18, scale: 0.97, filter: "blur(3px)" }
          }
          key={toast.kind === "placed" ? toast.pieceIds.join() : toast.text}
          transition={
            still
              ? { duration: 0 }
              : { type: "spring", stiffness: 420, damping: 34 }
          }
        >
          <span className="min-w-0 flex-1 py-2 wrap-anywhere">
            {toast.kind === "placed"
              ? toast.pieceIds.length === 1
                ? `Placed “${toast.name}”`
                : `Placed ${toast.name}`
              : toast.text}
          </span>
          {toast.kind === "placed" ? (
            <TimedUndo
              busy={undoing}
              onExpire={onClose}
              onUndo={async () => {
                setUndoing(true);
                await onUndo(toast.pieceIds, toast.name);
                setUndoing(false);
              }}
              seconds={UNDO_SECONDS}
            />
          ) : (
            <button
              aria-label="Close"
              className="inline-flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-ink"
              onClick={onClose}
              type="button"
            >
              <X aria-hidden="true" size={16} />
            </button>
          )}
        </motion.output>
      ) : null}
    </AnimatePresence>
  );
}
