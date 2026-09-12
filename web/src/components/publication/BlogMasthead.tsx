"use client";

import {
  motion,
  useMotionTemplate,
  useScroll,
  useTransform,
} from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { LineLink } from "@/components/ui/line-link";
import type { PublicationCategory } from "@/lib/api/query";
import { archivePath, BLOG_HOME } from "@/lib/blog-paths";
import { useOrigins } from "@/lib/origins";

export function BlogMasthead({
  categories,
}: {
  categories: PublicationCategory[];
}) {
  const pathname = usePathname();
  const { site } = useOrigins();
  const onArchive = pathname === BLOG_HOME || pathname.startsWith("/page/");
  const { scrollY } = useScroll();
  const depth = useTransform(scrollY, [0, 40], [0.1, 0.28], { clamp: true });
  const lift = useMotionTemplate`drop-shadow(0 6px 12px rgb(0 0 0 / ${depth}))`;

  return (
    <motion.header
      className="sticky top-0 z-80 bg-plane"
      style={{ filter: lift }}
    >
      <div className="mx-auto flex w-full max-w-[76rem] flex-wrap items-center justify-between gap-x-6 px-[var(--gutter)] py-2 sm:min-h-[var(--header-height)] sm:flex-nowrap sm:py-0">
        <div className="flex min-w-0 items-center gap-3">
          <a
            aria-label="Illarin home"
            className="flex min-h-11 shrink-0 items-center gap-2 text-ink"
            href={site}
          >
            <BrandLogo />
          </a>
          <Link
            className="flex min-h-11 items-center font-display text-[1.375rem] leading-none font-normal tracking-[-0.02em] text-mute hover:text-ink"
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
    </motion.header>
  );
}
