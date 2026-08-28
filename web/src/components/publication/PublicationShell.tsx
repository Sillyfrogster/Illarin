"use client";

import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { useAuth } from "@/lib/auth";
import styles from "./PublicationShell.module.css";

const SECTIONS: { href: string; label: string }[] = [
  { href: "/publication/apps", label: "Apps" },
  { href: "/publication/categories", label: "Categories" },
  { href: "/publication/contributors", label: "Contributors" },
  { href: "/publication/titles", label: "Titles and badges" },
];

export function PublicationShell({
  heading,
  hint,
  children,
}: {
  heading: string;
  hint: string;
  children: ReactNode;
}) {
  const { account, publicationAuthority } = useAuth();
  const here = usePathname();

  if (account === undefined) {
    return (
      <p className={styles.loading} aria-live="polite">
        Checking your account…
      </p>
    );
  }

  if (!account || !publicationAuthority) {
    return (
      <section className={styles.gate}>
        <ShieldCheck size={27} strokeWidth={1.35} aria-hidden="true" />
        <h2>Only Illarin's publication authority manages this</h2>
        <p>
          Being an admin or a moderator is not enough. The authority is one
          recorded account.
        </p>
        <Link href={account ? "/" : "/sign-in"}>
          {account ? "Back to Illarin" : "Sign in"}
        </Link>
      </section>
    );
  }

  return (
    <div className={styles.hub}>
      <nav className={styles.sections} aria-label="Publication">
        <ul>
          {SECTIONS.map((section) => (
            <li key={section.href}>
              <Link
                href={section.href}
                aria-current={here === section.href ? "page" : undefined}
              >
                {section.label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <section className={styles.section}>
        <header className={styles.heading}>
          <h2>{heading}</h2>
          <p>{hint}</p>
        </header>
        {children}
      </section>
    </div>
  );
}
