"use client";

import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";
import { useAuth } from "@/lib/auth";
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
  const { account, publicationAuthority } = useAuth();

  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <header className={styles.masthead}>
          <p className={styles.eyebrow}>{eyebrow}</p>
          <h1>{heading}</h1>
          <p className={styles.hint}>{hint}</p>
        </header>
        {account === undefined ? (
          <p className={styles.loading} aria-live="polite">
            Checking your account…
          </p>
        ) : null}
        {account !== undefined && (!account || !publicationAuthority) ? (
          <section className={styles.gate}>
            <ShieldCheck size={26} strokeWidth={1.35} aria-hidden="true" />
            <h2>One recorded account manages this</h2>
            <p>Being an admin or a moderator does not carry it.</p>
            <Link href={account ? "/" : "/sign-in"}>
              {account ? "Back to Illarin" : "Sign in"}
            </Link>
          </section>
        ) : null}
        {account && publicationAuthority ? children : null}
      </Shell>
    </section>
  );
}
