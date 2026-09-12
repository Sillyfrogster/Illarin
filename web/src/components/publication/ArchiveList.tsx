"use client";

import { motion, useReducedMotion } from "framer-motion";
import { useState } from "react";
import type { PostSummary } from "@/lib/api/query";
import type { ArchiveNarrowing } from "@/lib/archive-entry";
import { ArchiveRow } from "./ArchiveEntry";

export function ArchiveList({
  narrowed,
  posts,
}: {
  narrowed: ArchiveNarrowing;
  posts: PostSummary[];
}) {
  const [here, setHere] = useState<string | null>(null);
  const reduced = useReducedMotion();
  return (
    <div className="relative max-w-[62rem]">
      {posts.map((post) => (
        <article
          className="relative"
          key={post.id}
          onBlur={() => setHere(null)}
          onFocus={() => setHere(post.id)}
          onMouseEnter={() => setHere(post.id)}
          onMouseLeave={() => setHere(null)}
        >
          {here === post.id ? (
            <motion.div
              aria-hidden="true"
              className="absolute inset-y-1 -inset-x-4 rounded-plate bg-deep sm:-inset-x-6"
              layoutId="archive-here"
              transition={
                reduced
                  ? { duration: 0 }
                  : { type: "spring", stiffness: 260, damping: 30 }
              }
            />
          ) : null}
          <ArchiveRow narrowed={narrowed} post={post} />
        </article>
      ))}
    </div>
  );
}
