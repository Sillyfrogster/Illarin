import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";
import styles from "./PublicationLayout.module.css";

export default function PublicationLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <header className={styles.masthead}>
          <p className={styles.eyebrow}>Publication</p>
          <h1>Illarin's publication</h1>
          <p className={styles.standfirst}>
            The apps Illarin publishes for, the categories it sorts writing
            into, and the people approved to write.
          </p>
        </header>
        {children}
      </Shell>
    </section>
  );
}
