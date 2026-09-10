"use client";

import { AnimatePresence } from "framer-motion";
import { useCallback, useEffect, useState } from "react";
import { RegisterRail } from "@/components/register/RegisterRail";
import { Trouble } from "@/components/ui/field";
import { Waiting } from "@/components/ui/waiting";
import { WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import {
  readAccountDistinctions,
  readDefinitions,
} from "@/lib/api/distinctions";
import type { Distinction, DistinctionAssignment } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import {
  type RecognitionRegister as Register,
  recognitionCells,
} from "@/lib/recognition-register";
import { AccountRows, GiveStep, type LookedUpAccount } from "./AccountRows";
import { DefinitionRows, DefinitionStep } from "./DefinitionRows";

/** What the rail beside the register is open on, and which thing it is about. */
type Step =
  | { existing: Distinction | null; what: "definition" }
  | { what: "give" };

/** Everything Illarin gives out, one register at a time, with edits in the rail. */
export function RecognitionRegister() {
  const [definitions, setDefinitions] = useState<Distinction[] | null>(null);
  const [register, setRegister] = useState<Register>("titles");
  const [account, setAccount] = useState<LookedUpAccount | null>(null);
  const [looking, setLooking] = useState(false);
  const [step, setStep] = useState<Step | null>(null);
  const [failure, setFailure] = useState("");
  const [refusal, setRefusal] = useState("");

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

  function open(next: Step | null) {
    setRefusal("");
    setStep(next);
  }

  function close() {
    setRefusal("");
    setStep(null);
  }

  async function lookUp(handle: string) {
    setLooking(true);
    const answer = await readAccountDistinctions(handle);
    setLooking(false);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      setAccount(null);
      return;
    }
    setFailure("");
    close();
    setAccount({
      assignments: answer.value.assignments,
      handle: answer.value.handle,
    });
  }

  function replace(saved: Distinction, added: boolean) {
    setDefinitions((held) =>
      added
        ? [...(held ?? []), saved]
        : (held ?? []).map((one) => (one.id === saved.id ? saved : one)),
    );
    setStep((open) =>
      open?.what === "definition" && open.existing?.id === saved.id
        ? { existing: saved, what: "definition" }
        : open,
    );
  }

  function held(assignments: DistinctionAssignment[]) {
    setAccount((found) => (found ? { ...found, assignments } : found));
  }

  if (!definitions) {
    return <Waiting>{failure || "Reading what Illarin gives out…"}</Waiting>;
  }

  return (
    <>
      <RegisterRail
        cells={recognitionCells(definitions)}
        chosen={register}
        label="What Illarin gives out"
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

        {register === "accounts" ? (
          <AccountRows
            account={account}
            definitions={definitions}
            looking={looking}
            onFailure={setFailure}
            onGive={() => open({ what: "give" })}
            onHeld={held}
            onLookUp={lookUp}
          />
        ) : (
          <DefinitionRows
            definitions={definitions}
            onChanged={setDefinitions}
            onFailure={setFailure}
            onOpen={(existing) => open({ existing, what: "definition" })}
            register={register}
          />
        )}
      </section>

      <AnimatePresence>
        {step ? (
          <WorkspaceRail
            description={stepHint(step, register)}
            key={stepKey(step)}
            onClose={close}
            title={stepTitle(step, register, account)}
          >
            {refusal ? (
              <div className="mb-5">
                <Trouble>{refusal}</Trouble>
              </div>
            ) : null}

            {step.what === "definition" && register !== "accounts" ? (
              <DefinitionStep
                existing={step.existing}
                onClose={close}
                onFailure={setRefusal}
                onSaved={replace}
                register={register}
              />
            ) : null}

            {step.what === "give" && account ? (
              <GiveStep
                account={account}
                definitions={definitions}
                onFailure={setRefusal}
                onGiven={(assignment) =>
                  held([...account.assignments, assignment])
                }
              />
            ) : null}
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>
    </>
  );
}

/** Keeps a step's typed input while the register behind it changes. */
function stepKey(step: Step): string {
  if (step.what === "give") return "give";
  return `definition-${step.existing?.id ?? ""}`;
}

function stepTitle(
  step: Step,
  register: Register,
  account: LookedUpAccount | null,
): string {
  if (step.what === "give") {
    return account ? `Give one to @${account.handle}` : "Give one";
  }
  if (step.existing) return step.existing.name;
  return register === "titles" ? "Add a title or badge" : "Add a position";
}

function stepHint(step: Step, register: Register): string | undefined {
  if (step.what === "give") {
    return "It shows on their profile at once. None of them lets anyone do anything.";
  }
  if (register === "titles") {
    return "A mark makes it a badge. Without one it is a title.";
  }
  return undefined;
}
