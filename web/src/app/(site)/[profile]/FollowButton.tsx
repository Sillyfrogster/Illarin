"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { Check, LoaderCircle } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { followCreator, stopFollowingCreator } from "@/lib/api/profile";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";

const SPRING = { type: "spring", stiffness: 380, damping: 32 } as const;

/** Follow collapses into a circle while it saves, shows a check, and settles as Following. */
export function FollowButton({
  handle,
  following,
  onChange,
}: {
  handle: string;
  following: boolean;
  onChange: (following: boolean, followers: number) => void;
}) {
  const { account } = useAuth();
  const still = useReducedMotion();
  const [phase, setPhase] = useState<"idle" | "saving" | "done">("idle");
  const [trouble, setTrouble] = useState("");

  if (account === null) {
    return (
      <Button asChild className="w-full" variant="primary">
        <Link href={`/sign-in?returnTo=${encodeURIComponent(`/@${handle}`)}`}>
          Follow
        </Link>
      </Button>
    );
  }

  async function change() {
    if (phase !== "idle") return;
    setPhase("saving");
    setTrouble("");
    const answer = following
      ? await stopFollowingCreator(handle)
      : await followCreator(handle);
    if (!answer.value) {
      setPhase("idle");
      setTrouble(answer.error ?? "That did not work. Try again.");
      return;
    }
    onChange(answer.value.following, answer.value.followers);
    if (!answer.value.following || still) {
      setPhase("idle");
      return;
    }
    setPhase("done");
    window.setTimeout(() => setPhase("idle"), 700);
  }

  const collapsed = phase !== "idle";
  return (
    <div className="min-w-0">
      <motion.button
        animate={{ width: collapsed ? 44 : "100%" }}
        aria-busy={phase === "saving" || undefined}
        aria-pressed={following}
        className={cn(
          "relative flex h-11 items-center justify-center overflow-hidden rounded-control font-ui text-ui font-medium tracking-tight outline-offset-3 transition-colors duration-200",
          following || collapsed
            ? "bg-accent-wash text-accent"
            : "bg-action text-on-accent shadow-[0_4px_14px_-5px_var(--v-action),inset_0_1px_0_rgb(255_255_255/0.18)] hover:bg-action/90",
        )}
        disabled={account === undefined}
        initial={false}
        onClick={change}
        transition={still ? { duration: 0 } : SPRING}
        type="button"
      >
        <AnimatePresence initial={false} mode="popLayout">
          {phase === "saving" ? (
            <motion.span
              animate={{ opacity: 1, scale: 1 }}
              className="flex"
              exit={{ opacity: 0, scale: 0.6 }}
              initial={{ opacity: 0, scale: 0.6 }}
              key="saving"
            >
              <LoaderCircle
                aria-hidden="true"
                className="size-5 motion-safe:animate-spin"
              />
            </motion.span>
          ) : phase === "done" ? (
            <motion.span
              animate={{ opacity: 1, scale: 1 }}
              className="flex"
              exit={{ opacity: 0, scale: 0.6 }}
              initial={{ opacity: 0, scale: 0.4 }}
              key="done"
            >
              <Check aria-hidden="true" className="size-5" strokeWidth={2.4} />
            </motion.span>
          ) : (
            <motion.span
              animate={{ opacity: 1, y: 0 }}
              className="flex items-center gap-2 whitespace-nowrap"
              exit={{ opacity: 0, y: -10 }}
              initial={{ opacity: 0, y: 10 }}
              key={following ? "following" : "follow"}
            >
              {following ? (
                <Check
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={2.2}
                />
              ) : null}
              {following ? "Following" : "Follow"}
            </motion.span>
          )}
        </AnimatePresence>
      </motion.button>
      {trouble ? (
        <p className="mt-2 font-ui text-meta text-stop" role="alert">
          {trouble}
        </p>
      ) : null}
    </div>
  );
}
