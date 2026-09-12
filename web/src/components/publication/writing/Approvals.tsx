"use client";

import { ArrowUpRight, Package } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import type { PublicationGrant, PublicationWorkspace } from "@/lib/api/query";

export function Approvals({ workspace }: { workspace: PublicationWorkspace }) {
  return (
    <section aria-labelledby="approvals" className="min-w-0">
      <h2
        className="font-display text-section font-medium tracking-tight text-ink"
        id="approvals"
      >
        What you may publish
      </h2>

      {workspace.grants.length === 0 ? (
        <p className="mt-3 max-w-[52ch] font-prose text-prose text-mute">
          You write as the Illarin Team. A post carries your name and your
          public positions.
        </p>
      ) : (
        <>
          <p className="mt-3 max-w-[52ch] font-prose text-prose text-mute">
            Your{" "}
            <Link
              className="text-ink underline decoration-rule underline-offset-[3px] hover:decoration-ink"
              href={`/@${workspace.handle}`}
            >
              Verified App Contributor badge
            </Link>{" "}
            stays on your profile for as long as an approval stands.
          </p>
          <div className="mt-5 flex flex-col gap-3">
            {workspace.grants.map((grant) => (
              <Approval grant={grant} key={grant.id} />
            ))}
          </div>
        </>
      )}
    </section>
  );
}

function Approval({ grant }: { grant: PublicationGrant }) {
  return (
    <MorphingDisclosure
      className="rounded-plate bg-deep px-5 py-4"
      lead={
        <span className="flex min-w-0 items-center gap-3">
          <span className="grid size-10 shrink-0 place-items-center overflow-hidden rounded-control bg-plane text-mute">
            {grant.app.mark ? (
              <Image
                alt=""
                className="size-10 object-contain"
                height={40}
                src={grant.app.mark.url}
                unoptimized
                width={40}
              />
            ) : (
              <Package
                aria-hidden="true"
                className="size-5"
                strokeWidth={1.6}
              />
            )}
          </span>
          <span className="min-w-0">
            <span className="block font-display text-ui font-medium text-ink wrap-anywhere">
              {grant.app.name}
            </span>
            <span className="block font-prose text-meta text-mute">
              {grant.categories.length}{" "}
              {grant.categories.length === 1 ? "category" : "categories"} ·{" "}
              {grant.defaultCategory.label} by default
            </span>
          </span>
        </span>
      }
      summary="Approval"
    >
      <div className="mt-4 flex flex-col gap-5">
        <ul className="flex flex-wrap gap-1.5">
          {grant.categories.map((category) => (
            <li
              className="inline-flex min-h-8 items-center gap-2 rounded-control bg-plane px-3 font-ui text-meta text-ink"
              key={category.id}
            >
              {category.label}
              {category.id === grant.defaultCategory.id ? (
                <span className="font-prose text-label text-accent">
                  Default
                </span>
              ) : null}
            </li>
          ))}
        </ul>

        <p className="max-w-[56ch] font-prose text-meta text-mute">
          A post carries your name and {grant.app.name}, and Illarin stays the
          publisher.
        </p>

        <a
          className="inline-flex min-h-11 items-center gap-1.5 font-ui text-meta text-mute outline-offset-3 [overflow-wrap:anywhere] hover:text-ink"
          href={grant.app.home}
          rel="noreferrer noopener"
          target="_blank"
        >
          {grant.app.home.replace(/^https:\/\//, "")}
          <ArrowUpRight aria-hidden="true" className="size-3.5" />
        </a>

        <Link
          className="inline-flex min-h-11 items-center gap-1.5 font-ui text-ui font-medium text-accent outline-offset-3 hover:underline"
          href="/admin/blog/api"
        >
          API tokens and examples
          <ArrowUpRight aria-hidden="true" className="size-4" />
        </Link>
      </div>
    </MorphingDisclosure>
  );
}
