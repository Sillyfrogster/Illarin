import Link from "next/link";
import { BrandMark } from "@/components/brand/BrandMark";
import { ThemeControl } from "@/components/layout/ThemeControl";
import { LEGAL_DOCUMENTS } from "@/lib/legal-documents";
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
          <ThemeControl />
        </div>
      </div>
    </footer>
  );
}
