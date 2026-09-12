import { ArrowRight, ArrowUp } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";
import {
  clauseAnchor,
  LEGAL_DOCUMENTS,
  LEGAL_EFFECTIVE_DATE,
  type LegalHref,
  nextDocument,
} from "@/lib/legal-documents";
import { LegalContents } from "./LegalContents";

export type LegalClause = {
  body: ReactNode;
  heading: string;
};

const READING = [
  "font-prose text-article text-ink",
  "[&_p]:mt-4 [&_p]:text-pretty",
  "[&_h3]:mt-8 [&_h3]:font-ui [&_h3]:text-ui [&_h3]:font-medium",
  "[&_ul]:mt-4 [&_ul]:list-disc [&_ul]:pl-6 [&_ul]:marker:text-mute",
  "[&_ol]:mt-4 [&_ol]:list-decimal [&_ol]:pl-6 [&_ol]:marker:text-mute",
  "[&_li]:mt-2 [&_li]:pl-1",
  "[&_strong]:font-medium",
  "[&_a]:underline [&_a]:decoration-accent/55 [&_a]:underline-offset-[3px] hover:[&_a]:decoration-accent",
].join(" ");

export function LegalPage({
  clauses,
  href,
  lede,
  title,
}: {
  clauses: LegalClause[];
  href: LegalHref;
  lede: ReactNode;
  title: string;
}) {
  const onward = nextDocument(href);

  return (
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <div className="grid items-start gap-x-14 gap-y-8 lg:grid-cols-[15rem_minmax(0,1fr)]">
        <aside className="grid gap-6 lg:sticky lg:top-[calc(var(--header-height)+2rem)]">
          <nav aria-label="Legal documents">
            <h2 className="font-ui text-meta font-medium text-mute">
              Illarin&rsquo;s terms
            </h2>
            <ul className="mt-2 grid list-none">
              {LEGAL_DOCUMENTS.map((document) => (
                <li key={document.href}>
                  <Link
                    aria-current={document.href === href ? "page" : undefined}
                    className="flex min-h-11 items-center font-ui text-ui text-mute outline-offset-3 hover:text-ink aria-[current=page]:font-medium aria-[current=page]:text-accent"
                    href={document.href}
                  >
                    {document.title}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
          <LegalContents clauses={clauses} />
        </aside>

        <article className="min-w-0 max-w-[70ch]" id="document-top">
          <header>
            <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
              {title}
            </h1>
            <p className="mt-3 font-prose text-meta text-mute">
              Effective {LEGAL_EFFECTIVE_DATE}
            </p>
          </header>

          <p className="mt-8 font-prose text-lede text-ink">{lede}</p>

          {clauses.map((clause) => (
            <section
              className="mt-12 scroll-mt-[calc(var(--header-height)+2rem)]"
              id={clauseAnchor(clause.heading)}
              key={clause.heading}
            >
              <h2 className="font-display text-title font-medium tracking-[-0.02em] text-ink">
                {clause.heading}
              </h2>
              <div className={READING}>{clause.body}</div>
            </section>
          ))}

          {onward ? (
            <Link
              className="mt-section flex flex-wrap items-center justify-between gap-4 rounded-plate bg-deep p-5 font-ui text-ui text-ink outline-offset-3 hover:bg-rule/45"
              href={onward.href}
            >
              <span>
                Next: {onward.title}
                <span className="sr-only">, the next document</span>
              </span>
              <ArrowRight aria-hidden="true" className="size-4 text-accent" />
            </Link>
          ) : null}

          <a
            className="mt-group inline-flex min-h-11 items-center gap-2 font-ui text-meta text-mute outline-offset-3 hover:text-ink"
            href="#document-top"
          >
            Back to the top
            <ArrowUp aria-hidden="true" className="size-4" />
          </a>
        </article>
      </div>
    </Shell>
  );
}
