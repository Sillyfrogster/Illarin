import { BrandMark } from "@/components/brand/BrandMark";
import { LineLink } from "@/components/ui/line-link";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
import { NAV } from "./destinations";

const DESTINATIONS = [
  ...NAV,
  { label: "Publish", href: "/upload" },
  { label: "Account", href: "/settings" },
];

/** The way onward from the bottom of any page */
export function SiteFooter() {
  return (
    <footer className="mt-chapter bg-field pb-16">
      <div className="mx-auto w-full max-w-[var(--shell)] px-[var(--gutter)]">
        <div className="h-px w-full bg-edge" />
        <div className="grid gap-group pt-10 sm:grid-cols-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)] md:gap-x-14">
          <div className="min-w-0">
            <p className="flex items-center gap-2 text-ink">
              <BrandMark size={20} tone="accent" />
              <span className="font-display text-section font-medium tracking-[-0.02em]">
                Illarin
              </span>
            </p>
            <p className="mt-3 max-w-[34ch] font-prose text-meta leading-6 text-mute">
              A cross-application catalog for AI roleplay assets. Every
              creator&rsquo;s source file stays intact.
            </p>
            <p className="mt-8 text-meta text-mute">© 2026 Illarin</p>
          </div>

          <nav aria-label="Site">
            <h2 className="text-meta text-mute">Catalog</h2>
            <ul className="mt-1 grid list-none">
              {DESTINATIONS.map((item) => (
                <li key={item.href}>
                  <LineLink href={item.href} className="text-ink">
                    {item.label}
                  </LineLink>
                </li>
              ))}
            </ul>
          </nav>

          <nav aria-label="Legal documents">
            <h2 className="text-meta text-mute">Legal</h2>
            <ul className="mt-1 grid list-none">
              {LEGAL_DOCUMENTS.map((document) => (
                <li key={document.href}>
                  <LineLink href={document.href} className="text-ink">
                    {document.title}
                  </LineLink>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </div>
    </footer>
  );
}
