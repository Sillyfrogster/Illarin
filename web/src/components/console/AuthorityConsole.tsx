"use client";

import type { ReactNode } from "react";
import { useAuth } from "@/lib/auth";
import { ConsoleGate } from "./ConsoleGate";
import { ConsolePage } from "./ConsolePage";
import styles from "./ConsolePage.module.css";

export function AuthorityConsole({
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
    <ConsolePage eyebrow={eyebrow} heading={heading} hint={hint}>
      {account === undefined ? (
        <p className={styles.loading} aria-live="polite">
          Checking your account…
        </p>
      ) : null}
      {account !== undefined && (!account || !publicationAuthority) ? (
        <ConsoleGate
          heading="One recorded account manages this"
          line="Being an admin or a moderator does not carry it."
          href={account ? "/" : "/sign-in"}
          action={account ? "Back to Illarin" : "Sign in"}
        />
      ) : null}
      {account && publicationAuthority ? children : null}
    </ConsolePage>
  );
}
