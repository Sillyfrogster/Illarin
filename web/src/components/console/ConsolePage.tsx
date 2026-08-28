import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";
import styles from "./ConsolePage.module.css";

export function ConsolePage({
  eyebrow,
  heading,
  hint,
  children,
}: {
  eyebrow: string;
  heading: string;
  hint: string;
  children: ReactNode;
}) {
  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <header className={styles.masthead}>
          <p className={styles.eyebrow}>{eyebrow}</p>
          <h1>{heading}</h1>
          <p className={styles.hint}>{hint}</p>
        </header>
        {children}
      </Shell>
    </section>
  );
}
