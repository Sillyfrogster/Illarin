"use client";

import { ArrowUpRight, Package } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { ConsoleGate } from "@/components/console/ConsoleGate";
import { ConsolePage } from "@/components/console/ConsolePage";
import { Section } from "@/components/console/Section";
import { readWorkspace } from "@/lib/api/publication";
import type { PublicationGrant, PublicationWorkspace } from "@/lib/api/query";
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

  return (
    <ConsolePage
      eyebrow="Publication"
      heading="What you may publish"
      hint="The projects Illarin approved you to write for, and the categories each approval covers."
    >
      <Inside account={account} open={open} failure={failure} />
    </ConsolePage>
  );
}

function Inside({
  account,
  open,
  failure,
}: {
  account: ReturnType<typeof useAuth>["account"];
  open: PublicationWorkspace | null;
  failure: string;
}) {
  if (account === undefined) {
    return (
      <p className={rows.loading} aria-live="polite">
        Checking your account…
      </p>
    );
  }

  if (!account) {
    return (
      <ConsoleGate
        heading="Sign in to see what you may publish"
        line="Illarin opens this page to the accounts it has approved."
        href="/sign-in"
        action="Sign in"
      />
    );
  }

  if (!open) {
    return (
      <p className={rows.loading} aria-live="polite">
        {failure || "Reading your approvals…"}
      </p>
    );
  }

  if (open.grants.length === 0) {
    return (
      <ConsoleGate
        heading="Nobody has approved you to publish"
        line="Approval comes from Illarin's publication authority. An admin or a moderator role is not the same thing."
        href="/"
        action="Back to Illarin"
      />
    );
  }

  return (
    <div className={styles.grants}>
      {open.grants.map((grant) => (
        <Approval key={grant.id} grant={grant} />
      ))}
      <p className={styles.later}>
        Writing happens here once the editor is built. Until then this page is
        the record of what you were approved for. Your{" "}
        <Link href={`/${open.handle}`}>Verified App Contributor badge</Link>{" "}
        stays on your profile for as long as an approval stands.
      </p>
    </div>
  );
}

function Approval({ grant }: { grant: PublicationGrant }) {
  return (
    <Section
      lead={
        <span className={styles.mark}>
          {grant.app.mark ? (
            <Image
              src={grant.app.mark.url}
              alt=""
              width={40}
              height={40}
              unoptimized
            />
          ) : (
            <Package size={19} strokeWidth={1.6} aria-hidden="true" />
          )}
        </span>
      }
      title={grant.app.name}
      action={
        <a
          className={styles.home}
          href={grant.app.home}
          rel="noreferrer noopener"
          target="_blank"
        >
          {grant.app.home.replace(/^https:\/\//, "")}
          <ArrowUpRight size={14} strokeWidth={1.7} aria-hidden="true" />
        </a>
      }
    >
      <ol className={rows.list}>
        {grant.categories.map((category) => (
          <li className={rows.row} data-plain="true" key={category.id}>
            <span className={rows.name}>
              {category.label}{" "}
              <span className={rows.slug}>{category.slug}</span>
            </span>
            <span className={rows.detail} />
            <span className={rows.actions}>
              {category.id === grant.defaultCategory.id ? (
                <span className={styles.byDefault}>Default</span>
              ) : null}
            </span>
          </li>
        ))}
      </ol>
      <div className={styles.footnote}>
        <p className={styles.byline}>
          A post carries your name and {grant.app.name}, and Illarin stays the
          publisher. {grant.defaultCategory.label} is chosen unless you pick
          another.
        </p>
      </div>
    </Section>
  );
}
