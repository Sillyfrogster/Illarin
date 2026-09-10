import { Rss } from "lucide-react";
import { BrandMark } from "@/components/brand/BrandMark";
import { AppearanceMenu } from "@/components/layout/AppearanceMenu";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
import {
  BLOG_DESCRIPTION,
  PUBLICATION_FEEDS,
} from "@/lib/publication-metadata";
import { siteAddress } from "@/lib/site-address";

export function BlogFooter() {
  return (
    <footer className="mt-20 bg-plane">
      <div className="mx-auto grid w-full max-w-[76rem] gap-x-14 gap-y-8 px-[var(--gutter)] py-12 sm:grid-cols-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)]">
        <div className="grid content-start gap-3">
          <a
            className="group flex items-center gap-2 text-mute hover:text-ink"
            href={siteAddress("/")}
          >
            <BrandMark size={20} tone="accent" />
            <span className="font-display text-section font-medium tracking-[-0.02em]">
              Illarin
            </span>
          </a>
          <p className="max-w-[40ch] font-prose text-meta leading-6 text-mute">
            {BLOG_DESCRIPTION}
          </p>
          <p className="text-meta text-mute">© 2026 Illarin</p>
        </div>

        <nav aria-label="Legal documents">
          <h2 className="text-meta text-mute">Legal</h2>
          <ul className="mt-1 grid list-none">
            {LEGAL_DOCUMENTS.map((document) => (
              <li key={document.href}>
                <a
                  className="flex min-h-11 items-center text-ui text-ink hover:text-accent"
                  href={siteAddress(document.href)}
                >
                  {document.title}
                </a>
              </li>
            ))}
          </ul>
        </nav>

        <div className="grid content-start">
          <h2 className="text-meta text-mute">Read on</h2>
          <a
            className="mt-1 flex min-h-11 items-center gap-2 text-ui text-ink hover:text-accent"
            href={PUBLICATION_FEEDS.rss}
          >
            <Rss aria-hidden="true" className="size-4" />
            RSS feed
          </a>
          <a
            className="flex min-h-11 items-center text-ui text-ink hover:text-accent"
            href={siteAddress("/browse")}
          >
            The collection
          </a>
          <div className="mt-4">
            <AppearanceMenu labelled />
          </div>
        </div>
      </div>
    </footer>
  );
}
