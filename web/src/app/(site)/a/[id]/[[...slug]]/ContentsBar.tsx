"use client";

import { Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { WorkBlock } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { useWorkspace } from "./workspace/state";

type ContentsBlock = Pick<WorkBlock, "id" | "title">;

const TOOL =
  "inline-flex min-h-11 shrink-0 items-center gap-2 rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink";

export function ContentsBar({
  blocks,
  shellClassName,
  writing,
}: {
  blocks: ContentsBlock[];
  shellClassName: string;
  writing: boolean;
}) {
  const workspace = useWorkspace();
  const bar = useRef<HTMLDivElement>(null);
  const [activeBlockId, setActiveBlockId] = useState(blocks[0]?.id ?? null);

  useEffect(() => {
    const hash = window.location.hash;
    const hashBlockId = hash.startsWith("#block-")
      ? hash.slice("#block-".length)
      : null;
    setActiveBlockId((active) => {
      if (hashBlockId && blocks.some((block) => block.id === hashBlockId)) {
        return hashBlockId;
      }
      return active && blocks.some((block) => block.id === active)
        ? active
        : (blocks[0]?.id ?? null);
    });
  }, [blocks]);

  useEffect(() => {
    if (blocks.length === 0) return;

    let frame = 0;
    const updateActiveBlock = () => {
      frame = 0;
      const readingLine =
        (bar.current?.getBoundingClientRect().bottom ?? 0) + 16;
      const current = blocks.find((block) => {
        const target = document.getElementById(`block-${block.id}`);
        return target
          ? target.getBoundingClientRect().bottom > readingLine
          : false;
      });
      setActiveBlockId((active) => current?.id ?? blocks.at(-1)?.id ?? active);
    };
    const scheduleUpdate = () => {
      if (frame) return;
      frame = window.requestAnimationFrame(updateActiveBlock);
    };

    scheduleUpdate();
    window.addEventListener("scroll", scheduleUpdate, { passive: true });
    window.addEventListener("resize", scheduleUpdate);
    return () => {
      window.removeEventListener("scroll", scheduleUpdate);
      window.removeEventListener("resize", scheduleUpdate);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [blocks]);

  if (blocks.length === 0 && !writing) return null;

  return (
    <div
      className="sticky top-[var(--header-height)] z-20 bg-field shadow-contents"
      ref={bar}
    >
      <div
        className={cn(
          shellClassName,
          "flex items-center justify-between gap-4 py-1",
        )}
      >
        {blocks.length > 0 ? (
          <nav
            aria-label="Contents"
            className="flex min-w-0 flex-1 items-center gap-6"
          >
            <span className="hidden shrink-0 text-meta text-mute lg:inline">
              On this page
            </span>
            <ol className="flex min-w-0 list-none items-center gap-6 overflow-x-auto">
              {blocks.map((block) => (
                <li className="shrink-0" key={block.id}>
                  <a
                    aria-current={
                      activeBlockId === block.id ? "location" : undefined
                    }
                    className="flex min-h-11 items-center whitespace-nowrap text-ui text-mute outline-offset-3 hover:text-ink aria-[current=location]:font-medium aria-[current=location]:text-accent"
                    href={`#block-${block.id}`}
                    onClick={() => setActiveBlockId(block.id)}
                  >
                    {block.title}
                  </a>
                </li>
              ))}
            </ol>
          </nav>
        ) : (
          <p className="shrink-0 py-3 text-meta text-mute">Start this page</p>
        )}

        {writing && workspace.addableBlocks.length > 0 ? (
          <button
            aria-expanded={workspace.pane?.kind === "catalog"}
            className={cn(TOOL, "shrink-0")}
            onClick={() => workspace.openPane({ kind: "catalog" })}
            type="button"
          >
            <Plus aria-hidden="true" size={17} />
            <span>Add block</span>
          </button>
        ) : null}
      </div>
    </div>
  );
}
