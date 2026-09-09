"use client";

import { Eye, ListTree, Plus, Undo2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { AssetBlock } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { CreatorMenu, type CreatorMenuProps } from "./CreatorMenu";

type ContentsBlock = Pick<AssetBlock, "id" | "title">;

const TOOL =
  "inline-flex min-h-11 shrink-0 items-center gap-2 rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink";

/** The blocks a page holds, and the tools its creator works on it with */
export function ContentsBar({
  blocks,
  isOwner,
  arranging,
  adding,
  readerView,
  canAdd,
  creatorMenu,
  shellClassName,
  onToggleArrange,
  onToggleAdd,
  onReaderView,
  onReturnToEditing,
}: {
  blocks: ContentsBlock[];
  isOwner: boolean;
  arranging: boolean;
  adding: boolean;
  readerView: boolean;
  canAdd: boolean;
  creatorMenu: CreatorMenuProps;
  shellClassName: string;
  onToggleArrange: () => void;
  onToggleAdd: () => void;
  onReaderView: () => void;
  onReturnToEditing: () => void;
}) {
  const { account } = useAuth();
  const bar = useRef<HTMLDivElement>(null);
  const [activeBlockId, setActiveBlockId] = useState(blocks[0]?.id ?? null);
  const hasStaffTools = Boolean(
    account?.role === "admin" && !creatorMenu.isDraft && !creatorMenu.withheld,
  );
  const hasPageTools = isOwner || hasStaffTools;

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

  if (blocks.length === 0 && !hasPageTools) return null;

  return (
    <div
      className="sticky top-[var(--header-height)] z-20 bg-field shadow-contents"
      ref={bar}
    >
      <div
        className={cn(
          shellClassName,
          "flex items-center justify-between gap-6 overflow-x-auto py-1",
        )}
      >
        {arranging ? (
          <p className="shrink-0 py-3 text-meta text-mute">
            Arrangement outline
          </p>
        ) : blocks.length > 0 ? (
          <nav
            aria-label="Contents"
            className="flex min-w-0 items-center gap-6"
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
        ) : isOwner ? (
          <p className="shrink-0 py-3 text-meta text-mute">Start this page</p>
        ) : null}

        {hasPageTools ? (
          <div
            aria-label="Page tools"
            className="flex shrink-0 items-center gap-1"
            role="toolbar"
          >
            {isOwner && readerView ? (
              <>
                <span className="inline-flex min-h-11 shrink-0 items-center gap-2 px-3 text-meta text-mute">
                  <Eye aria-hidden="true" size={16} />
                  <span>Reader’s view</span>
                </span>
                <button
                  className={TOOL}
                  onClick={onReturnToEditing}
                  type="button"
                >
                  <Undo2 aria-hidden="true" size={16} />
                  <span>Return</span>
                </button>
              </>
            ) : isOwner ? (
              <>
                <button
                  aria-expanded={arranging}
                  className={TOOL}
                  onClick={onToggleArrange}
                  type="button"
                >
                  {arranging ? (
                    <X aria-hidden="true" size={17} />
                  ) : (
                    <ListTree aria-hidden="true" size={17} />
                  )}
                  <span>{arranging ? "Close outline" : "Arrange"}</span>
                </button>
                {canAdd ? (
                  <button
                    aria-expanded={adding}
                    className={TOOL}
                    onClick={onToggleAdd}
                    type="button"
                  >
                    {adding ? (
                      <X aria-hidden="true" size={17} />
                    ) : (
                      <Plus aria-hidden="true" size={17} />
                    )}
                    <span>{adding ? "Close add" : "Add block"}</span>
                  </button>
                ) : null}
                <button className={TOOL} onClick={onReaderView} type="button">
                  <Eye aria-hidden="true" size={17} />
                  <span>Reader’s view</span>
                </button>
                <CreatorMenu {...creatorMenu} />
              </>
            ) : (
              <CreatorMenu {...creatorMenu} />
            )}
          </div>
        ) : null}
      </div>
    </div>
  );
}
