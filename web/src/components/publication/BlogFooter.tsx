import { Rss } from "lucide-react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { AppearanceMenu } from "@/components/layout/AppearanceMenu";
import { PUBLICATION_FEEDS } from "@/lib/blog-paths";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
import { BLOG_DESCRIPTION } from "@/lib/publication-metadata";
import { siteAddress } from "@/lib/site-address";

export function BlogFooter() {
  return (
    <footer className="mt-20 bg-plane">
      <div className="mx-auto grid w-full max-w-[76rem] gap-x-14 gap-y-8 px-[var(--gutter)] py-12 sm:grid-cols-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)]">
        <div className="grid content-start gap-3">
          <a
            aria-label="Illarin home"
            className="flex min-h-11 w-fit items-center"
            href={siteAddress("/")}
          >
            <BrandLogo className="w-40" tone="accent" />
          </a>
          <p className="max-w-[40ch] font-prose text-meta leading-6 text-mute">
            {BLOG_DESCRIPTION}
          </p>
          <a
            className="flex min-h-11 w-fit items-center text-meta text-ink hover:text-accent"
            download
            href="/brand/illarin-brandkit.zip"
          >
            Download the brand kit
          </a>
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
          <h2 className="text-meta text-mute">Blog</h2>
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
            Browse
          </a>
          <div className="mt-4">
            <AppearanceMenu labelled />
          </div>
        </div>
      </div>
    </footer>
  );
}
