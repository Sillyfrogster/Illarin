import { Rss } from "lucide-react";
import Link from "next/link";
import { BrandMark } from "@/components/brand/BrandMark";
import { AppearanceMenu } from "@/components/layout/AppearanceMenu";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
import { PUBLICATION_FEEDS } from "@/lib/publication-metadata";
import styles from "./BlogFooter.module.css";

/** The blog's footer carries the way back to Illarin and nothing a reader cannot use here. */
export function BlogFooter() {
  return (
    <footer className={styles.footer}>
      <div className={styles.inner}>
        <div className={styles.meta}>
          <Link className={styles.brand} href="/">
            <BrandMark size={18} tone="faint" />
            <span className={styles.wordmark}>Illarin</span>
          </Link>
          <p className={styles.description}>
            Official writing from Illarin and the projects it publishes for.
          </p>
          <a className={styles.feed} href={PUBLICATION_FEEDS.rss}>
            <Rss aria-hidden="true" size={13} strokeWidth={1.9} />
            RSS feed
          </a>
          <span className={styles.copyright}>© 2026 Illarin</span>
        </div>

        <div className={styles.aside}>
          <nav className={styles.legal} aria-label="Legal documents">
            {LEGAL_DOCUMENTS.map((document) => (
              <Link href={document.href} key={document.href}>
                {document.title}
              </Link>
            ))}
          </nav>
          <AppearanceMenu labelled />
        </div>
      </div>
    </footer>
  );
}
