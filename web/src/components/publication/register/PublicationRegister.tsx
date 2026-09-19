"use client";

import { AnimatePresence } from "framer-motion";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RegisterRail } from "@/components/register/RegisterRail";
import { Trouble } from "@/components/ui/field";
import { Waiting } from "@/components/ui/waiting";
import { RailBack, WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import {
  readCategories,
  readDeliveries,
  readGrants,
  readIntegrations,
  readWorkspace,
} from "@/lib/api/publication";
import type {
  BlogAnnouncementAttempt,
  BlogAnnouncementAttemptState,
  BlogIntegration,
  PublicationApp,
  PublicationCategory,
  PublicationGrant,
} from "@/lib/api/query";
import { attemptState } from "@/lib/attempt-standing";
import { cn } from "@/lib/cn";
import {
  REGISTERS,
  type Register,
  registerName,
  registerStandings,
} from "@/lib/publication-register";
import { AttemptRows } from "./AttemptRows";
import { CategoryRows, CategoryStep } from "./CategoryRows";
import { ContributorRows, ContributorStep } from "./ContributorRows";
import { IntegrationRows, IntegrationStep } from "./IntegrationRows";
import { SecretStep } from "./SecretStep";

type Step =
  | { what: "contributor"; grant: PublicationGrant | null }
  | { what: "category"; category: PublicationCategory }
  | { what: "integration"; integration: BlogIntegration | null }
  | { what: "secret"; integration: BlogIntegration };

export function PublicationRegister() {
  const [apps, setApps] = useState<PublicationApp[] | null>(null);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [grants, setGrants] = useState<PublicationGrant[]>([]);
  const [integrations, setIntegrations] = useState<BlogIntegration[]>([]);
  const [attempts, setAttempts] = useState<BlogAnnouncementAttempt[]>([]);
  const [stopped, setStopped] = useState(0);
  const [register, setRegister] = useState<Register>("contributors");
  const [view, setView] = useState("all");
  const [step, setStep] = useState<Step | null>(null);
  const [failure, setFailure] = useState("");
  const [refusal, setRefusal] = useState("");

  const load = useCallback(async () => {
    const [workspace, categoriesIn, grantsIn, integrationsIn, sent, short] =
      await Promise.all([
        readWorkspace(),
        readCategories(),
        readGrants(),
        readIntegrations(),
        readDeliveries(),
        readDeliveries("failed"),
      ]);
    const trouble =
      workspace.error ??
      categoriesIn.error ??
      grantsIn.error ??
      integrationsIn.error ??
      sent.error ??
      "";
    if (trouble) {
      setFailure(trouble);
      return;
    }
    setFailure("");
    setCategories(categoriesIn.value?.categories ?? []);
    setGrants(grantsIn.value?.grants ?? []);
    setIntegrations(integrationsIn.value?.integrations ?? []);
    setAttempts(sent.value?.attempts ?? []);
    setStopped(countStopped(short.value?.attempts ?? []));
    setView("all");
    setApps(
      knownApps(workspace.value?.apps ?? [], grantsIn.value?.grants ?? []),
    );
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const narrow = useCallback(
    async (next: string, state?: BlogAnnouncementAttemptState) => {
      setView(next);
      const answer = await readDeliveries(state);
      if (answer.error) {
        setFailure(answer.error);
        return;
      }
      setFailure("");
      setAttempts(answer.value?.attempts ?? []);
    },
    [],
  );

  const standings = useMemo(
    () =>
      registerStandings({
        categories,
        integrations,
        grants,
        stopped,
      }),
    [categories, integrations, grants, stopped],
  );

  function open(next: Step | null) {
    setRefusal("");
    setStep(next);
  }

  function close() {
    setRefusal("");
    setStep(null);
  }

  function replaceIntegration(saved: BlogIntegration) {
    setIntegrations((held) =>
      held.some((one) => one.id === saved.id)
        ? held.map((one) => (one.id === saved.id ? saved : one))
        : [...held, saved],
    );
    setStep((open) =>
      open?.what === "integration" && open.integration?.id === saved.id
        ? { integration: saved, what: "integration" }
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

        {register === "integrations" ? (
          <IntegrationRows
            integrations={integrations}
            onFailure={setFailure}
            onOpen={(integration) => open({ integration, what: "integration" })}
            onSaved={replaceIntegration}
          />
        ) : null}

        {register === "attempts" ? (
          <AttemptRows
            attempts={attempts}
            onChanged={(changed) => {
              setAttempts((held) =>
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
                      integration: step.integration,
                      what: "integration",
                    })
                  }
                >
                  {step.integration.name}
                </RailBack>
              </div>
            ) : null}

            {step.what === "contributor" ? (
              <ContributorStep
                apps={apps}
                categories={categories}
                integrations={integrations}
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

            {step.what === "integration" ? (
              <IntegrationStep
                existing={step.integration}
                onClose={close}
                onFailure={setRefusal}
                onRemoved={() => {
                  const gone = step.integration?.id;
                  setIntegrations((held) =>
                    held.filter((one) => one.id !== gone),
                  );
                  void load();
                }}
                onRotate={() =>
                  step.integration
                    ? open({
                        integration: step.integration,
                        what: "secret",
                      })
                    : undefined
                }
                onSaved={replaceIntegration}
              />
            ) : null}

            {step.what === "secret" ? (
              <SecretStep
                integration={step.integration}
                onClose={close}
                onFailure={setRefusal}
                onRotated={replaceIntegration}
              />
            ) : null}
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>
    </>
  );
}

function knownApps(
  named: PublicationApp[],
  grants: PublicationGrant[],
): PublicationApp[] {
  const known = new Map(named.map((app) => [app.id, app]));
  for (const grant of grants) {
    if (!known.has(grant.app.id)) known.set(grant.app.id, grant.app);
  }
  return [...known.values()];
}

function countStopped(attempts: BlogAnnouncementAttempt[]): number {
  return attempts.filter(
    (one) => attemptState(one) === "gaveUp" && !one.removed,
  ).length;
}

function stepKey(step: Step): string {
  if (step.what === "contributor") return `contributor-${step.grant?.id ?? ""}`;
  if (step.what === "category") return `category-${step.category.id}`;
  if (step.what === "secret") return `secret-${step.integration.id}`;
  return `integration-${step.integration?.id ?? ""}`;
}

function stepTitle(step: Step): string {
  if (step.what === "contributor") {
    return step.grant
      ? `What @${step.grant.holder.handle} may publish`
      : "Approve a contributor";
  }
  if (step.what === "category") return `Rename ${step.category.label}`;
  if (step.what === "secret") {
    return `A new signing secret for ${step.integration.name}`;
  }
  return step.integration ? step.integration.name : "Add an integration";
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
