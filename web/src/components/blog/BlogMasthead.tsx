"use client";

import {
  motion,
  useMotionTemplate,
  useScroll,
  useTransform,
} from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { LineLink } from "@/components/ui/line-link";
import type { BlogCategory } from "@/lib/api/query";
import { archivePath, BLOG_HOME } from "@/lib/blog-paths";

export function BlogMasthead({ categories }: { categories: BlogCategory[] }) {
  const pathname = usePathname();
  const onArchive =
    pathname === BLOG_HOME || pathname.startsWith(`${BLOG_HOME}/page/`);
  const { scrollY } = useScroll();
  const depth = useTransform(scrollY, [0, 40], [0.1, 0.28], { clamp: true });
  const lift = useMotionTemplate`drop-shadow(0 6px 12px rgb(0 0 0 / ${depth}))`;

  return (
    <motion.div
      className="sticky top-[var(--site-header-offset)] z-70 bg-plane"
      style={{ filter: lift }}
    >
      <div className="mx-auto flex w-full max-w-[var(--shell)] flex-wrap items-center justify-between gap-x-6 px-[var(--gutter)] py-2 sm:min-h-14 sm:flex-nowrap sm:py-0">
        <div className="flex min-w-0 items-center">
          <Link
            className="flex min-h-11 items-center font-display text-[1.375rem] leading-none font-medium tracking-[-0.02em] text-ink"
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
            <LineLink current={onArchive} href={BLOG_HOME}>
              All posts
            </LineLink>
            {categories.map((category) => {
              const address = archivePath("category", category.slug);
              return (
                <LineLink
                  current={
                    pathname === address || pathname.startsWith(`${address}/`)
                  }
                  href={address}
                  key={category.id}
                >
                  {category.label}
                </LineLink>
              );
            })}
          </nav>
        ) : null}
      </div>
    </motion.div>
  );
}
