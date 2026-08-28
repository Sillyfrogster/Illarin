"use client";

import { useCallback, useEffect, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { readApps, readCategories, readGrants } from "@/lib/api/publication";
import type {
  PublicationApp,
  PublicationCategory,
  PublicationGrant,
} from "@/lib/api/query";
import { AppList } from "./AppList";
import { CategoryList } from "./CategoryList";
import { ContributorList } from "./ContributorList";
import styles from "./PublicationHub.module.css";

export function PublicationHub() {
  const [apps, setApps] = useState<PublicationApp[] | null>(null);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [grants, setGrants] = useState<PublicationGrant[]>([]);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    const [appAnswer, categoryAnswer, grantAnswer] = await Promise.all([
      readApps(),
      readCategories(),
      readGrants(),
    ]);
    const trouble =
      appAnswer.error ?? categoryAnswer.error ?? grantAnswer.error ?? "";
    if (trouble) {
      setFailure(trouble);
      return;
    }
    setFailure("");
    setCategories(categoryAnswer.value?.categories ?? []);
    setGrants(grantAnswer.value?.grants ?? []);
    setApps(appAnswer.value?.apps ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (!apps) {
    return (
      <p className={rows.loading} aria-live="polite">
        {failure || "Reading the publication…"}
      </p>
    );
  }

  return (
    <div className={styles.hub}>
      {failure ? (
        <p className={rows.failure} role="alert">
          {failure}
        </p>
      ) : null}
      <ContributorList
        grants={grants}
        apps={apps}
        categories={categories}
        onChanged={setGrants}
        onReload={load}
        onFailure={setFailure}
      />
      <div className={styles.pair}>
        <AppList apps={apps} onChanged={setApps} onFailure={setFailure} />
        <CategoryList
          categories={categories}
          onChanged={setCategories}
          onFailure={setFailure}
        />
      </div>
    </div>
  );
}
