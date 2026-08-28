"use client";

import { useCallback, useEffect, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { readDefinitions } from "@/lib/api/distinctions";
import type { Distinction } from "@/lib/api/query";
import { AccountRecognition } from "./AccountRecognition";
import { DefinitionList } from "./DefinitionList";
import styles from "./RecognitionConsole.module.css";

export function RecognitionConsole() {
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
      <p className={rows.loading} aria-live="polite">
        {failure || "Reading what Illarin gives out…"}
      </p>
    );
  }

  return (
    <div className={styles.console}>
      {failure ? (
        <p className={rows.failure} role="alert">
          {failure}
        </p>
      ) : null}
      <AccountRecognition definitions={definitions} onFailure={setFailure} />
      <DefinitionList
        kind="position"
        title="Illarin positions"
        definitions={definitions}
        onChanged={setDefinitions}
        onFailure={setFailure}
      />
      <DefinitionList
        kind="recognition"
        title="Titles and badges"
        definitions={definitions}
        onChanged={setDefinitions}
        onFailure={setFailure}
      />
    </div>
  );
}
