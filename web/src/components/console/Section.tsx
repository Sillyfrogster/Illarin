import type { ReactNode } from "react";
import styles from "./Section.module.css";

export function Section({
  id,
  lead,
  title,
  count,
  action,
  children,
  retired,
  retiredLabel,
  wide,
}: {
  id?: string;
  lead?: ReactNode;
  title: string;
  count?: number;
  action?: ReactNode;
  children: ReactNode;
  retired?: ReactNode;
  retiredLabel?: string;
  wide?: boolean;
}) {
  return (
    <section className={styles.section} data-wide={wide || undefined} id={id}>
      <header className={styles.heading}>
        {lead}
        <h2>{title}</h2>
        {count === undefined ? null : (
          <span className={styles.count}>{count}</span>
        )}
        {action ? <div className={styles.action}>{action}</div> : null}
      </header>
      {children}
      {retired && retiredLabel ? (
        <details className={styles.retired}>
          <summary>{retiredLabel}</summary>
          {retired}
        </details>
      ) : null}
    </section>
  );
}
