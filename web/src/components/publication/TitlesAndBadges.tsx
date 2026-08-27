"use client";

import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { readDefinitions } from "@/lib/api/distinctions";
import type { Distinction, DistinctionForm } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { AccountDistinctions } from "./AccountDistinctions";
import { DefinitionColumn } from "./DefinitionColumn";
import styles from "./TitlesAndBadges.module.css";

const COLUMNS: { form: DistinctionForm; heading: string; hint: string }[] = [
  {
    form: "position",
    heading: "Illarin positions",
    hint: "Official jobs. Everyone who holds one shows all of them, in this order.",
  },
  {
    form: "title",
    heading: "Titles",
    hint: "Names a person carries. A profile shows the first six it was given.",
  },
  {
    form: "badge",
    heading: "Badges",
    hint: "Recognition with a mark and a short explanation.",
  },
];

export function TitlesAndBadges() {
  const { account, publicationAuthority } = useAuth();
  const [definitions, setDefinitions] = useState<Distinction[] | null>(null);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    const answer = await readDefinitions();
    if (answer.error) {
      setFailure(answer.error);
      return;
    }
    setFailure("");
    setDefinitions(answer.value?.definitions ?? []);
  }, []);

  useEffect(() => {
    if (!publicationAuthority) return;
    void load();
  }, [load, publicationAuthority]);

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
        <h2>Only Illarin's publication authority manages these</h2>
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

  if (!definitions) {
    return (
      <p className={styles.loading} aria-live="polite">
        {failure || "Reading the titles and badges…"}
      </p>
    );
  }

  return (
    <div className={styles.regions}>
      {failure ? (
        <p className={styles.failure} role="alert">
          {failure}
        </p>
      ) : null}
      <AccountDistinctions definitions={definitions} />
      <section className={styles.definitions}>
        <h2 className={styles.definitionsHeading}>What can be given</h2>
        {COLUMNS.map((column) => (
          <DefinitionColumn
            key={column.form}
            form={column.form}
            heading={column.heading}
            hint={column.hint}
            definitions={definitions}
            onChanged={setDefinitions}
            onFailure={setFailure}
          />
        ))}
      </section>
    </div>
  );
}
