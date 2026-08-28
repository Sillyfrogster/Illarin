"use client";

import { Package, ShieldCheck } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { readWorkspace } from "@/lib/api/publication";
import type { PublicationWorkspace } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import styles from "./ContributorWorkspace.module.css";

export function ContributorWorkspace() {
  const { account } = useAuth();
  const [open, setOpen] = useState<PublicationWorkspace | null>(null);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    const answer = await readWorkspace();
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setOpen(answer.value);
  }, []);

  useEffect(() => {
    if (!account) return;
    void load();
  }, [account, load]);

  if (account === undefined) {
    return (
      <p className={styles.loading} aria-live="polite">
        Checking your account…
      </p>
    );
  }

  if (!account) {
    return (
      <section className={styles.gate}>
        <ShieldCheck size={27} strokeWidth={1.35} aria-hidden="true" />
        <h2>Sign in to reach your publication workspace</h2>
        <p>Illarin opens this to accounts approved to publish for an app.</p>
        <Link href="/sign-in">Sign in</Link>
      </section>
    );
  }

  if (!open) {
    return (
      <p className={styles.loading} aria-live="polite">
        {failure || "Reading what you may publish…"}
      </p>
    );
  }

  if (open.grants.length === 0) {
    return (
      <section className={styles.gate}>
        <ShieldCheck size={27} strokeWidth={1.35} aria-hidden="true" />
        <h2>You are not approved to publish</h2>
        <p>
          Publishing for a project needs an approval from Illarin's publication
          authority. Being an admin or a moderator does not carry one.
        </p>
        <Link href="/">Back to Illarin</Link>
      </section>
    );
  }

  return (
    <div className={styles.grants}>
      {open.grants.map((grant) => (
        <section className={styles.grant} key={grant.id}>
          <header className={styles.app}>
            <span className={styles.mark}>
              {grant.app.mark ? (
                <Image
                  src={grant.app.mark.url}
                  alt=""
                  width={44}
                  height={44}
                  unoptimized
                />
              ) : (
                <Package size={20} strokeWidth={1.6} aria-hidden="true" />
              )}
            </span>
            <span className={styles.identity}>
              <span className={styles.name}>{grant.app.name}</span>
              <a
                className={styles.home}
                href={grant.app.home}
                rel="noreferrer noopener"
                target="_blank"
              >
                {grant.app.home.replace(/^https:\/\//, "")}
              </a>
            </span>
          </header>

          <h2 className={styles.heading}>What you may publish</h2>
          <ul className={styles.allowed}>
            {grant.categories.map((category) => (
              <li key={category.id}>
                <span className={styles.category}>{category.label}</span>
                {category.id === grant.defaultCategory.id ? (
                  <span className={styles.byDefault}>default</span>
                ) : null}
              </li>
            ))}
          </ul>

          <p className={styles.byline}>
            Posts carry your name and this app. Illarin remains the publisher.
          </p>
        </section>
      ))}

      <p className={styles.later}>
        Your{" "}
        <Link href={`/${open.handle}`}>Verified App Contributor badge</Link> is
        on your public profile for as long as an approval stands. Writing a post
        arrives with the editor.
      </p>
    </div>
  );
}
