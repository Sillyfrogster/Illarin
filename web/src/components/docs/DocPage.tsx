import { ArrowLeft, ArrowRight } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import {
  type DocCollection,
  type DocPage as DocPageEntry,
  docHref,
  nextDocPage,
  previousDocPage,
} from "@/lib/docs/collections";
import type { Doc } from "@/lib/docs/read-doc";
import { DocBlocks } from "./DocBlocks";
import { DocContents } from "./DocContents";
import { DocSidebar } from "./DocSidebar";

const CONTENTS_FLOOR = 2;

const STICKY =
  "lg:sticky lg:top-[calc(var(--header-height)+2rem)] lg:max-h-[calc(100vh-var(--header-height)-2rem)] lg:overflow-y-auto";

export function DocPage({
  collection,
  doc,
  page,
}: {
  collection: DocCollection;
  doc: Doc;
  page: DocPageEntry;
}) {
  const back = previousDocPage(collection, page);
  const onward = nextDocPage(collection, page);
  const contents = doc.contents.length >= CONTENTS_FLOOR;

  return (
    <Shell className="pt-6 pb-chapter lg:pt-10">
      <div className="grid items-start gap-x-10 gap-y-4 lg:grid-cols-[14rem_minmax(0,1fr)] xl:grid-cols-[14rem_minmax(0,1fr)_12rem] xl:gap-x-14">
        <div className={STICKY}>
          <DocSidebar current={`${collection.name} · ${page.title}`} />
        </div>

        <article className="min-w-0 max-w-[46rem]" id="document-top">
          {contents ? (
            <div className="mb-6 xl:hidden">
              <DocContents entries={doc.contents} />
            </div>
          ) : null}
          <header className="border-b border-rule pb-6">
            <h1 className="font-display text-[2rem] leading-[1.15] font-semibold tracking-[-0.02em] text-ink text-balance">
              {doc.title}
            </h1>
            <p className="mt-2 font-prose text-prose text-mute text-pretty">
              {doc.lede}
            </p>
          </header>

          <DocBlocks blocks={doc.blocks} />

          <nav
            aria-label="Pages"
            className="mt-section grid gap-3 border-t border-rule pt-6 sm:grid-cols-2"
          >
            {back ? (
              <Link
                className="flex min-h-11 items-center gap-2 rounded-control font-ui text-ui text-mute outline-offset-3 hover:text-ink"
                href={docHref(collection, back)}
              >
                <ArrowLeft aria-hidden="true" className="size-4" />
                <span>
                  <span className="sr-only">Previous: </span>
                  {back.title}
                </span>
              </Link>
            ) : (
              <span />
            )}
            {onward ? (
              <Link
                className="flex min-h-11 items-center justify-end gap-2 rounded-control font-ui text-ui font-medium text-ink outline-offset-3 hover:text-accent"
                href={docHref(collection, onward)}
              >
                <span>
                  <span className="sr-only">Next: </span>
                  {onward.title}
                </span>
                <ArrowRight aria-hidden="true" className="size-4" />
              </Link>
            ) : null}
          </nav>
        </article>

        {contents ? (
          <div className={`hidden xl:block ${STICKY}`}>
            <DocContents entries={doc.contents} />
          </div>
        ) : null}
      </div>
    </Shell>
  );
}
