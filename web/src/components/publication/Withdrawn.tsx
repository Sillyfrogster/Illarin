import Link from "next/link";
import { Button } from "@/components/ui/button";
import { WITHDRAWAL_MESSAGE } from "@/lib/publication-withdrawal";

export function Withdrawn({ explanation }: { explanation: string }) {
  return (
    <section className="mx-auto grid min-h-[calc(100svh-var(--header-height))] w-full max-w-[76rem] content-center justify-items-start px-[var(--gutter)] py-section">
      <p aria-hidden="true" className="mb-5 text-meta font-medium text-accent">
        Withdrawn
      </p>
      <h1 className="max-w-[21ch] font-display text-hero font-medium tracking-[-0.03em] break-words text-balance">
        {WITHDRAWAL_MESSAGE}
      </h1>
      {explanation ? (
        <p className="mt-7 max-w-[46ch] font-prose text-lede text-mute text-pretty">
          {explanation}
        </p>
      ) : null}
      <Button asChild className="mt-10" size="large" variant="primary">
        <Link href="/blog">Read the rest of the blog</Link>
      </Button>
    </section>
  );
}
