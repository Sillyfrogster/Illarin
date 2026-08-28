"use client";

import { useCallback, useEffect, useState } from "react";
import { readDefinitions } from "@/lib/api/distinctions";
import type { Distinction, DistinctionForm } from "@/lib/api/query";
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
    void load();
  }, [load]);

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
