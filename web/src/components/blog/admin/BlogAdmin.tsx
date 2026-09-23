"use client";

import { AnimatePresence } from "framer-motion";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RegisterRail } from "@/components/register/RegisterRail";
import { Trouble } from "@/components/ui/field";
import { Waiting } from "@/components/ui/waiting";
import { DiscordChannel } from "@/components/updates/DiscordChannel";
import { WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import { readCategories, readWriters } from "@/lib/api/blog";
import type { BlogCategory, WriterResponse } from "@/lib/api/query";
import {
  REGISTERS,
  type Register,
  registerName,
  registerStandings,
} from "@/lib/blog-admin";
import { cn } from "@/lib/cn";
import { CategoryRows, CategoryStep } from "./CategoryRows";
import { WriterRows, WriterStep } from "./WriterRows";

type Step =
  | { what: "writer"; writer: WriterResponse | null }
  | { what: "category"; category: BlogCategory };

export function BlogAdmin() {
  const [writers, setWriters] = useState<WriterResponse[] | null>(null);
  const [categories, setCategories] = useState<BlogCategory[]>([]);
  const [register, setRegister] = useState<Register>("writers");
  const [step, setStep] = useState<Step | null>(null);
  const [failure, setFailure] = useState("");
  const [refusal, setRefusal] = useState("");

  const load = useCallback(async () => {
    const [categoriesIn, writersIn] = await Promise.all([
      readCategories(),
      readWriters(),
    ]);
    const trouble = categoriesIn.error ?? writersIn.error ?? "";
    if (trouble) {
      setFailure(trouble);
      return;
    }
    setFailure("");
    setCategories(categoriesIn.value?.categories ?? []);
    setWriters(writersIn.value?.writers ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const standings = useMemo(
    () => registerStandings({ categories, writers: writers ?? [] }),
    [categories, writers],
  );

  function open(next: Step | null) {
    setRefusal("");
    setStep(next);
  }

  function close() {
    setRefusal("");
    setStep(null);
  }

  if (!writers) {
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

        {register === "writers" ? (
          <WriterRows
            onOpen={(writer) => open({ what: "writer", writer })}
            writers={writers}
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

        {register === "discord" ? (
          <>
            <p className="mb-5 max-w-[36rem] font-prose text-ui text-mute">
              A post's first publication goes to this channel when its writer
              leaves Post to Discord on.
            </p>
            <DiscordChannel scope="blog" />
          </>
        ) : null}
      </section>

      <AnimatePresence>
        {step ? (
          <WorkspaceRail
            description={stepHint(step)}
            key={stepKey(step)}
            onClose={close}
            title={stepTitle(step)}
            tone="accent"
          >
            {refusal ? (
              <div className="mb-5">
                <Trouble>{refusal}</Trouble>
              </div>
            ) : null}

            {step.what === "writer" ? (
              <WriterStep
                existing={step.writer}
                onClose={close}
                onFailure={setRefusal}
                onSwitchedOff={load}
                onSwitchedOn={(saved) =>
                  setWriters((held) => [saved, ...(held ?? [])])
                }
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
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>
    </>
  );
}

function stepKey(step: Step): string {
  if (step.what === "writer") return `writer-${step.writer?.accountId ?? ""}`;
  return `category-${step.category.id}`;
}

function stepTitle(step: Step): string {
  if (step.what === "writer") {
    return step.writer
      ? `@${step.writer.handle} writes for the blog`
      : "Switch on a writer";
  }
  return `Rename ${step.category.label}`;
}

function stepHint(step: Step): string {
  if (step.what === "writer") {
    return step.writer
      ? "They reach Your posts and the editor, and publish under their own name."
      : "The switch opens the blog editor for one account. It grants no other authority.";
  }
  return "Readers see the name. Nothing that already references this category moves.";
}
