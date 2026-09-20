import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import type { BrowsePage } from "@/lib/api/query";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { BrowseRetry } from "./BrowseRetry";

export function BrowseChapter({ page }: { page: BrowsePage | null }) {
  const items = page?.items ?? [];
  const hidden =
    page?.emptyState === "suppressed" || (page?.suppressed ?? 0) > 0;
  if (!items.length)
    return (
      <div className="gallery-state">
        <h3>
          {!page
            ? "Recent work could not load."
            : hidden
              ? "Recent work is hidden."
              : "The first story could be yours."}
        </h3>
        <p role={!page ? "status" : undefined}>
          {!page
            ? "Try again, or explore the community in Browse."
            : hidden
              ? "Published work is outside your content preference. You can change what you see in Browse."
              : "Create a character or lorebook, or bring a work you’ve already started."}
        </p>
        {!page ? (
          <BrowseRetry />
        ) : (
          <Button asChild variant="primary">
            <Link href={hidden ? "/browse" : "/upload"}>
              {hidden ? "Go to Browse" : "Start creating"} →
            </Link>
          </Button>
        )}
      </div>
    );
  return (
    <>
      <ul className="exhibition" aria-label="Recently published works">
        {items.map((work) => (
          <li className="exhibit" key={work.id}>
            <Link href={workHref(work.id, work.name)}>
              <div className="exhibit-art">
                {work.cover ? (
                  <Image
                    src={work.cover.url}
                    alt=""
                    width={work.cover.width}
                    height={work.cover.height}
                    sizes="(max-width:600px) 45vw, 285px"
                    unoptimized
                    className={
                      work.isNsfw && page?.nsfwPreference !== "shown"
                        ? "blur-xl"
                        : undefined
                    }
                  />
                ) : (
                  <span className="px-5 text-center text-meta text-mute">
                    {TYPE_LABELS[work.type]}
                    <br />
                    No cover image
                  </span>
                )}
              </div>
              <div className="exhibit-caption">
                <span>
                  {TYPE_LABELS[work.type]}
                  {work.isNsfw
                    ? page?.nsfwPreference === "shown"
                      ? " · Adult"
                      : " · Adult, blurred"
                    : ""}
                </span>
                <strong>{work.name || "Untitled"}</strong>
                <span className="exhibit-arrow" aria-hidden="true">
                  ↗
                </span>
              </div>
            </Link>
            <Link className="creator" href={`/@${work.creator}`}>
              @{work.creator}
            </Link>
          </li>
        ))}
      </ul>
      {hidden ? (
        <p className="content-preference">
          Some works are outside your content preference.
        </p>
      ) : null}
    </>
  );
}

export function BrowseLoading() {
  return (
    <div className="gallery-state" aria-busy="true">
      <output>Finding recent creations…</output>
    </div>
  );
}
