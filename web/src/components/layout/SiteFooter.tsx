import { SiDiscord, SiKofi } from "@icons-pack/react-simple-icons";
import Link from "next/link";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { shellClasses } from "@/components/layout/Shell";
import { BLOG_HOME } from "@/lib/blog-paths";
import { navLink } from "@/lib/cn";
import { DISCORD_INVITE, KOFI_PAGE } from "@/lib/contact";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
import { ArtworkSwitch } from "./ArtworkSwitch";

export function SiteFooter() {
  const destinations = [
    { label: "Guide", href: "/docs" },
    { label: "Blog", href: BLOG_HOME },
    { label: "About", href: "/about" },
  ];
  return (
    <footer className="mt-chapter bg-field pb-16">
      <div className={shellClasses}>
        <div className="h-px w-full bg-edge" />
        <div className="grid gap-group pt-10 sm:grid-cols-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)] md:gap-x-14">
          <div className="min-w-0">
            <BrandLogo className="w-40" />
            <a
              className="mt-3 flex min-h-control w-fit items-center text-meta text-ink hover:text-accent"
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
                  <Link className={navLink} href={item.href}>
                    {item.label}
                  </Link>
                </li>
              ))}
              <li>
                <a
                  className="flex min-h-control w-fit items-center gap-2 text-ui font-medium text-ink decoration-accent underline-offset-4 hover:underline"
                  href={DISCORD_INVITE}
                  rel="noopener"
                  target="_blank"
                >
                  <SiDiscord aria-hidden="true" className="size-4" title="" />
                  Discord
                </a>
              </li>
              <li>
                <a
                  className="flex min-h-control w-fit items-center gap-2 text-ui font-medium text-ink decoration-accent underline-offset-4 hover:underline"
                  href={KOFI_PAGE}
                  rel="noopener"
                  target="_blank"
                >
                  <SiKofi aria-hidden="true" className="size-4" title="" />
                  Ko-fi
                </a>
              </li>
            </ul>
          </nav>

          <nav aria-label="Legal documents">
            <h2 className="text-meta text-mute">Legal</h2>
            <ul className="mt-1 grid list-none">
              {LEGAL_DOCUMENTS.map((document) => (
                <li key={document.href}>
                  <Link className={navLink} href={document.href}>
                    {document.title}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        </div>
        <div className="mt-10 flex flex-wrap items-center gap-x-8 gap-y-2 border-t border-edge pt-4">
          <ArtworkSwitch />
        </div>
      </div>
    </footer>
  );
}
