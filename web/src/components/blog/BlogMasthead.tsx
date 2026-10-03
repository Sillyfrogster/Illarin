"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { BlogCategory } from "@/lib/api/query";
import { archivePath, BLOG_HOME } from "@/lib/blog-paths";
import { navLink } from "@/lib/cn";

export function BlogMasthead({ categories }: { categories: BlogCategory[] }) {
  const pathname = usePathname();
  const onArchive =
    pathname === BLOG_HOME || pathname.startsWith(`${BLOG_HOME}/page/`);

  return (
    <div className="sticky top-[var(--site-header-offset)] z-70 border-b border-rule bg-field">
      <div className="mx-auto flex w-full max-w-[var(--shell)] flex-wrap items-center justify-between gap-x-6 px-[var(--gutter)] py-2 sm:min-h-14 sm:flex-nowrap sm:py-0">
        <div className="flex min-w-0 items-center">
          <Link
            className="flex min-h-control items-center font-display text-[1.375rem] leading-none font-medium tracking-[-0.02em] text-ink"
            href={BLOG_HOME}
          >
            Blog
          </Link>
        </div>

        {categories.length > 0 ? (
          <nav
            aria-label="Blog categories"
            className="-mx-1 flex min-w-0 items-center gap-x-4 overflow-x-auto px-1 [scrollbar-width:none] max-sm:w-full [&::-webkit-scrollbar]:hidden"
          >
            <Link
              aria-current={onArchive ? "page" : undefined}
              className={navLink}
              href={BLOG_HOME}
            >
              All posts
            </Link>
            {categories.map((category) => {
              const address = archivePath("category", category.slug);
              return (
                <Link
                  aria-current={
                    pathname === address || pathname.startsWith(`${address}/`)
                      ? "page"
                      : undefined
                  }
                  className={navLink}
                  href={address}
                  key={category.id}
                >
                  {category.label}
                </Link>
              );
            })}
          </nav>
        ) : null}
      </div>
    </div>
  );
}
