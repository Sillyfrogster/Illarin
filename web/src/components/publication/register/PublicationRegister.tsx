"use client";

import { AnimatePresence } from "framer-motion";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RegisterRail } from "@/components/register/RegisterRail";
import { Trouble } from "@/components/ui/field";
import { Waiting } from "@/components/ui/waiting";
import { RailBack, WorkspaceRail } from "@/components/workspace/WorkspaceRail";
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
import { cn } from "@/lib/cn";
import { deliveryState } from "@/lib/publication-delivery";
import {
  REGISTERS,
  type Register,
  registerName,
  registerStandings,
} from "@/lib/publication-register";
import { AppRows, AppStep } from "./AppRows";
import { CategoryRows, CategoryStep } from "./CategoryRows";
import { ContributorRows, ContributorStep } from "./ContributorRows";
import { DeliveryRows } from "./DeliveryRows";
import { DestinationRows, DestinationStep } from "./DestinationRows";
import { SecretStep } from "./SecretStep";

type Step =
  | { what: "contributor"; grant: PublicationGrant | null }
  | { what: "app"; app: PublicationApp | null }
  | { what: "category"; category: PublicationCategory }
  | { what: "destination"; destination: PublicationDestination | null }
  | { what: "secret"; destination: PublicationDestination };

export function PublicationRegister() {
  const [apps, setApps] = useState<PublicationApp[] | null>(null);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [grants, setGrants] = useState<PublicationGrant[]>([]);
  const [destinations, setDestinations] = useState<PublicationDestination[]>(
    [],
  );
  const [deliveries, setDeliveries] = useState<PostDelivery[]>([]);
  const [stopped, setStopped] = useState(0);
  const [register, setRegister] = useState<Register>("contributors");
  const [view, setView] = useState("all");
  const [step, setStep] = useState<Step | null>(null);
  const [failure, setFailure] = useState("");
  const [refusal, setRefusal] = useState("");

  const load = useCallback(async () => {
    const [appsIn, categoriesIn, grantsIn, destinationsIn, sent, short] =
      await Promise.all([
        readApps(),
        readCategories(),
        readGrants(),
        readDestinations(),
        readDeliveries(),
        readDeliveries("failed"),
      ]);
    const trouble =
      appsIn.error ??
      categoriesIn.error ??
      grantsIn.error ??
      destinationsIn.error ??
      sent.error ??
      "";
    if (trouble) {
      setFailure(trouble);
      return;
    }
    setFailure("");
    setCategories(categoriesIn.value?.categories ?? []);
    setGrants(grantsIn.value?.grants ?? []);
    setDestinations(destinationsIn.value?.destinations ?? []);
    setDeliveries(sent.value?.deliveries ?? []);
    setStopped(countStopped(short.value?.deliveries ?? []));
    setView("all");
    setApps(appsIn.value?.apps ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

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

  const standings = useMemo(
    () =>
      registerStandings({
        apps: apps ?? [],
        categories,
        destinations,
        grants,
        stopped,
      }),
    [apps, categories, destinations, grants, stopped],
  );

  function open(next: Step | null) {
    setRefusal("");
    setStep(next);
  }

  function close() {
    setRefusal("");
    setStep(null);
  }

  function replaceDestination(saved: PublicationDestination) {
    setDestinations((held) =>
      held.some((one) => one.id === saved.id)
        ? held.map((one) => (one.id === saved.id ? saved : one))
        : [...held, saved],
    );
    setStep((open) =>
      open?.what === "destination" && open.destination?.id === saved.id
        ? { destination: saved, what: "destination" }
        : open,
    );
  }

  function replaceApp(saved: PublicationApp, added: boolean) {
    setApps((held) =>
      added
        ? [...(held ?? []), saved]
        : (held ?? []).map((one) => (one.id === saved.id ? saved : one)),
    );
    setStep((open) =>
      open?.what === "app" && open.app?.id === saved.id
        ? { app: saved, what: "app" }
        : open,
    );
  }

  if (!apps) {
    return <Waiting>{failure || "Loading blog administration…"}</Waiting>;
  }

  return (
    <>
      <RegisterRail
        cells={REGISTERS.map((one) => ({
          attention: standings[one].attention,
          count: standings[one].count,
          id: one,
          name: registerName(one),
        }))}
        chosen={register}
        label="Blog administration"
        onChoose={(next) => {
          setRegister(next);
          close();
        }}
      />

      <section
        aria-labelledby="register-heading"
        className={cn(
          "mt-8 min-w-0 transition-[padding] duration-500 ease-wipe motion-reduce:transition-none",
          step && "lg:pr-[30rem]",
        )}
      >
        {failure ? (
          <div className="mb-6">
            <Trouble>{failure}</Trouble>
          </div>
        ) : null}

        {register === "contributors" ? (
          <ContributorRows
            apps={apps}
            grants={grants}
            onOpen={(grant) => open({ grant, what: "contributor" })}
          />
        ) : null}

        {register === "apps" ? (
          <AppRows
            apps={apps}
            onFailure={setFailure}
            onOpen={(app) => open({ app, what: "app" })}
            onOrdered={setApps}
            onSaved={replaceApp}
          />
        ) : null}

        {register === "categories" ? (
          <CategoryRows
            categories={categories}
            onFailure={setFailure}
            onOpen={(category) => open({ category, what: "category" })}
            onOrdered={setCategories}
            onSaved={(saved) =>
              setCategories((held) =>
                held.map((one) => (one.id === saved.id ? saved : one)),
              )
            }
          />
        ) : null}

        {register === "destinations" ? (
          <DestinationRows
            destinations={destinations}
            onFailure={setFailure}
            onOpen={(destination) => open({ destination, what: "destination" })}
            onSaved={replaceDestination}
          />
        ) : null}

        {register === "deliveries" ? (
          <DeliveryRows
            deliveries={deliveries}
            onChanged={(changed) => {
              setDeliveries((held) =>
                held.map((one) => (one.id === changed.id ? changed : one)),
              );
              setStopped((held) => Math.max(0, held - 1));
            }}
            onFailure={setFailure}
            onView={(next, state) => void narrow(next, state)}
            view={view}
          />
        ) : null}
      </section>

      <AnimatePresence>
        {step ? (
          <WorkspaceRail
            description={stepHint(step)}
            key={stepKey(step)}
            onClose={close}
            title={stepTitle(step)}
            tone={step.what === "secret" ? "stop" : "accent"}
          >
            {refusal ? (
              <div className="mb-5">
                <Trouble>{refusal}</Trouble>
              </div>
            ) : null}

            {step.what === "secret" ? (
              <div className="mb-5">
                <RailBack
                  onClick={() =>
                    open({
                      destination: step.destination,
                      what: "destination",
                    })
                  }
                >
                  {step.destination.name}
                </RailBack>
              </div>
            ) : null}

            {step.what === "contributor" ? (
              <ContributorStep
                apps={apps}
                categories={categories}
                destinations={destinations}
                existing={step.grant}
                onClose={close}
                onFailure={setRefusal}
                onRevoked={load}
                onSaved={(saved, added) => {
                  setGrants((held) =>
                    added
                      ? [saved, ...held]
                      : held.map((one) => (one.id === saved.id ? saved : one)),
                  );
                  setStep((open) =>
                    open?.what === "contributor" && open.grant
                      ? { grant: saved, what: "contributor" }
                      : open,
                  );
                }}
              />
            ) : null}

            {step.what === "app" ? (
              <AppStep
                destinations={destinations}
                existing={step.app}
                onClose={close}
                onFailure={setRefusal}
                onSaved={replaceApp}
              />
            ) : null}

            {step.what === "category" ? (
              <CategoryStep
                category={step.category}
                onClose={close}
                onFailure={setRefusal}
                onSaved={(saved) =>
                  setCategories((held) =>
                    held.map((one) => (one.id === saved.id ? saved : one)),
                  )
                }
              />
            ) : null}

            {step.what === "destination" ? (
              <DestinationStep
                existing={step.destination}
                onClose={close}
                onFailure={setRefusal}
                onRemoved={() => {
                  const gone = step.destination?.id;
                  setDestinations((held) =>
                    held.filter((one) => one.id !== gone),
                  );
                  void load();
                }}
                onRotate={() =>
                  step.destination
                    ? open({
                        destination: step.destination,
                        what: "secret",
                      })
                    : undefined
                }
                onSaved={replaceDestination}
              />
            ) : null}

            {step.what === "secret" ? (
              <SecretStep
                destination={step.destination}
                onClose={close}
                onFailure={setRefusal}
                onRotated={replaceDestination}
              />
            ) : null}
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>
    </>
  );
}

function countStopped(deliveries: PostDelivery[]): number {
  return deliveries.filter(
    (one) => deliveryState(one) === "gaveUp" && !one.removed,
  ).length;
}

function stepKey(step: Step): string {
  if (step.what === "contributor") return `contributor-${step.grant?.id ?? ""}`;
  if (step.what === "app") return `app-${step.app?.id ?? ""}`;
  if (step.what === "category") return `category-${step.category.id}`;
  if (step.what === "secret") return `secret-${step.destination.id}`;
  return `destination-${step.destination?.id ?? ""}`;
}

function stepTitle(step: Step): string {
  if (step.what === "contributor") {
    return step.grant
      ? `What @${step.grant.holder.handle} may publish`
      : "Approve a contributor";
  }
  if (step.what === "app") {
    return step.app ? step.app.name : "Add an app";
  }
  if (step.what === "category") return `Rename ${step.category.label}`;
  if (step.what === "secret") {
    return `A new signing secret for ${step.destination.name}`;
  }
  return step.destination ? step.destination.name : "Add a destination";
}

function stepHint(step: Step): string | undefined {
  if (step.what === "contributor") {
    return step.grant
      ? `They publish for ${step.grant.app.name} under their own name. Changing the app needs a new approval.`
      : "One approval, one person, one app. They publish under their own name and gain no other authority.";
  }
  if (step.what === "category") {
    return "Readers see the name. Nothing that already references this category moves.";
  }
  return undefined;
}
