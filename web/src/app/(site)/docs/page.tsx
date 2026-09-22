import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { DOCS } from "@/lib/docs";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Developer documentation",
  "Guides for connecting apps and publishing extensions on Illarin.",
);

export default function DocsHome() {
  return (
    <Shell className="pt-12 pb-chapter lg:pt-18">
      <header className="max-w-[47rem]">
        <p className="font-ui text-meta font-medium text-accent">
          Documentation
        </p>
        <h1 className="mt-4 font-display text-display font-medium tracking-[-0.04em] text-ink">
          Build with Illarin
        </h1>
        <p className="mt-6 font-prose text-lede text-mute">
          Integration contracts for app developers and release guidance for
          extension developers.
        </p>
      </header>
      <div className="mt-14 grid gap-3 md:grid-cols-2">
        {DOCS.map((doc) => (
          <Link
            className="group flex min-h-44 flex-col justify-between rounded-plate bg-deep p-6 text-ink outline-offset-3 hover:bg-inset"
            href={`/docs/${doc.slug}`}
            key={doc.slug}
          >
            <span>
              <span className="block font-ui text-section font-medium">
                {doc.title}
              </span>
              <span className="mt-3 block max-w-[32ch] font-prose text-ui text-mute">
                {doc.description}
              </span>
            </span>
            <ArrowRight
              aria-hidden="true"
              className="mt-6 size-5 text-accent transition-transform group-hover:translate-x-1"
            />
          </Link>
        ))}
      </div>
    </Shell>
  );
}
