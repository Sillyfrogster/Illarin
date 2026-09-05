"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandMark } from "@/components/brand/BrandMark";
import type { PublicationCategory } from "@/lib/api/query";
import styles from "./BlogMasthead.module.css";

/** The blog's own chrome: Illarin's identity, the publication's home, and its categories. */
export function BlogMasthead({
  categories,
}: {
  categories: PublicationCategory[];
}) {
  const pathname = usePathname();
  const onArchive = pathname === "/blog" || pathname.startsWith("/blog/page/");
  return (
    <header className={styles.masthead}>
      <div className={styles.bar}>
        <div className={styles.identity}>
          <Link className={styles.brand} href="/" aria-label="Illarin home">
            <BrandMark size={24} />
            <span className={styles.wordmark}>Illarin</span>
          </Link>
          <span aria-hidden="true" className={styles.divider} />
          <Link className={styles.publication} href="/blog">
            Blog
          </Link>
        </div>

        {categories.length > 0 ? (
          <nav className={styles.nav} aria-label="Publication categories">
            <Link
              aria-current={onArchive ? "page" : undefined}
              className={styles.category}
              href="/blog"
            >
              All
            </Link>
            {categories.map((category) => {
              const address = `/blog/category/${category.slug}`;
              const current =
                pathname === address || pathname.startsWith(`${address}/`);
              return (
                <Link
                  aria-current={current ? "page" : undefined}
                  className={styles.category}
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
    </header>
  );
}
