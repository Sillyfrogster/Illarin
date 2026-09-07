"use client";

import { CircleAlert } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import rows from "@/components/console/Console.module.css";
import {
  readApps,
  readCategories,
  readDeliveries,
  readDestinations,
  readGrants,
} from "@/lib/api/publication";
import type {
  PostDelivery,
  PostDeliveryState,
  PublicationApp,
  PublicationCategory,
  PublicationDestination,
  PublicationGrant,
} from "@/lib/api/query";
import { deliveryState } from "@/lib/publication-delivery";
import { AppList } from "./AppList";
import { CategoryList } from "./CategoryList";
import { ContributorList } from "./ContributorList";
import { DeliveryList } from "./DeliveryList";
import { DestinationList } from "./DestinationList";
import styles from "./PublicationHub.module.css";

export function PublicationHub() {
  const [apps, setApps] = useState<PublicationApp[] | null>(null);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [grants, setGrants] = useState<PublicationGrant[]>([]);
  const [destinations, setDestinations] = useState<PublicationDestination[]>(
    [],
  );
  const [deliveries, setDeliveries] = useState<PostDelivery[]>([]);
  const [view, setView] = useState("all");
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    const [
      appAnswer,
      categoryAnswer,
      grantAnswer,
      destinationAnswer,
      deliveryAnswer,
    ] = await Promise.all([
      readApps(),
      readCategories(),
      readGrants(),
      readDestinations(),
      readDeliveries(),
    ]);
    const trouble =
      appAnswer.error ??
      categoryAnswer.error ??
      grantAnswer.error ??
      destinationAnswer.error ??
      deliveryAnswer.error ??
      "";
    if (trouble) {
      setFailure(trouble);
      return;
    }
    setFailure("");
    setCategories(categoryAnswer.value?.categories ?? []);
    setGrants(grantAnswer.value?.grants ?? []);
    setDestinations(destinationAnswer.value?.destinations ?? []);
    setDeliveries(deliveryAnswer.value?.deliveries ?? []);
    setApps(appAnswer.value?.apps ?? []);
  }, []);

  const narrow = useCallback(
    async (next: string, state?: PostDeliveryState) => {
      setView(next);
      const answer = await readDeliveries(state);
      if (answer.error) {
        setFailure(answer.error);
        return;
      }
      setFailure("");
      setDeliveries(answer.value?.deliveries ?? []);
    },
    [],
  );

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

  const stuck = deliveries.filter(
    (one) => deliveryState(one) === "gaveUp",
  ).length;

  return (
    <div className={styles.hub}>
      {failure ? (
        <p className={rows.failure} role="alert">
          {failure}
        </p>
      ) : null}
      {stuck > 0 ? (
        <p className={styles.attention}>
          <CircleAlert size={16} strokeWidth={1.9} aria-hidden="true" />
          {stuck === 1
            ? "One announcement stopped short of its destination."
            : `${stuck} announcements stopped short of their destinations.`}
          <a href="#deliveries">Look at them</a>
        </p>
      ) : null}
      <ContributorList
        grants={grants}
        apps={apps}
        categories={categories}
        destinations={destinations}
        onChanged={setGrants}
        onReload={load}
        onFailure={setFailure}
      />
      <DestinationList
        destinations={destinations}
        onChanged={(changed) => {
          setDestinations(changed);
          void load();
        }}
        onFailure={setFailure}
      />
      <DeliveryList
        deliveries={deliveries}
        onChanged={(changed) =>
          setDeliveries((held) =>
            held.map((one) => (one.id === changed.id ? changed : one)),
          )
        }
        onFailure={setFailure}
        onView={(next, state) => void narrow(next, state)}
        view={view}
      />
      <div className={styles.pair}>
        <AppList
          apps={apps}
          destinations={destinations}
          onChanged={setApps}
          onFailure={setFailure}
        />
        <CategoryList
          categories={categories}
          onChanged={setCategories}
          onFailure={setFailure}
        />
      </div>
    </div>
  );
}
