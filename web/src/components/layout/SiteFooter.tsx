import { BrandLogo } from "@/components/brand/BrandLogo";
import { shellClasses } from "@/components/layout/Shell";
import { LineLink } from "@/components/ui/line-link";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
import { primaryDestinations } from "./destinations";

export function SiteFooter() {
  const destinations = [
    ...primaryDestinations(),
    { label: "Publish", href: "/upload" },
    { label: "Account settings", href: "/settings" },
  ];
  return (
    <footer className="mt-chapter bg-field pb-16">
      <div className={shellClasses}>
        <div className="h-px w-full bg-edge" />
        <div className="grid gap-group pt-10 sm:grid-cols-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)] md:gap-x-14">
          <div className="min-w-0">
            <BrandLogo className="w-40" tone="accent" />
            <p className="mt-3 max-w-[34ch] font-prose text-meta leading-6 text-mute">
              A hub for AI roleplay work, across apps. Every creator&rsquo;s
              source file stays intact.
            </p>
            <a
              className="mt-3 flex min-h-11 w-fit items-center text-meta text-ink hover:text-accent"
              download
              href="/brand/illarin-brandkit.zip"
            >
              Download the brand kit
            </a>
            <p className="mt-8 text-meta text-mute">© 2026 Illarin</p>
          </div>

          <nav aria-label="Site">
            <h2 className="text-meta text-mute">Site</h2>
            <ul className="mt-1 grid list-none">
              {destinations.map((item) => (
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
