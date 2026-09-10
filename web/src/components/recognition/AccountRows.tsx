"use client";

import {
  ArrowDown,
  ArrowUp,
  BriefcaseBusiness,
  Search,
  Type,
  UserRoundCheck,
} from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { type FormEvent, useState } from "react";
import {
  Nothing,
  PanelHead,
  Past,
  PastRow,
  RowAction,
  RowMove,
  StartAction,
} from "@/components/register/RowParts";
import { Button } from "@/components/ui/button";
import { Field, TextInput } from "@/components/ui/field";
import {
  giveDistinction,
  orderAccountDistinctions,
  takeBackDistinction,
} from "@/lib/api/distinctions";
import type { Distinction, DistinctionAssignment } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { readableDate } from "@/lib/dates";
import {
  heldBy,
  movedAssignments,
  nothingIn,
  offeredTo,
  whatIsHeld,
} from "@/lib/recognition-register";

export type LookedUpAccount = {
  assignments: DistinctionAssignment[];
  handle: string;
};

export function AccountRows({
  account,
  definitions,
  looking,
  onFailure,
  onGive,
  onHeld,
  onLookUp,
}: {
  account: LookedUpAccount | null;
  definitions: Distinction[];
  looking: boolean;
  onFailure: (message: string) => void;
  onGive: () => void;
  onHeld: (assignments: DistinctionAssignment[]) => void;
  onLookUp: (handle: string) => void;
}) {
  const [typed, setTyped] = useState("");
  const held = account ? heldBy(account.assignments) : null;
  const offered = account ? offeredTo(definitions, account.assignments) : [];

  function look(event: FormEvent) {
    event.preventDefault();
    const wanted = typed.trim().replace(/^@/, "");
    if (!wanted || looking) return;
    onLookUp(wanted);
  }

  async function takeBack(one: DistinctionAssignment) {
    if (!account) return;
    const answer = await takeBackDistinction(account.handle, one.id);
    if (answer.error) {
      onFailure(answer.error);
      return;
    }
    onFailure("");
    onHeld(
      account.assignments.map((other) =>
        other.id === one.id ? { ...other, active: false } : other,
      ),
    );
  }

  async function reorder(one: DistinctionAssignment, step: number) {
    if (!account) return;
    const order = movedAssignments(account.assignments, one, step);
    if (!order) return;
    const answer = await orderAccountDistinctions(account.handle, order);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onHeld(answer.value.assignments);
  }

  return (
    <>
      <PanelHead id="register-heading" title="Someone's profile" />

      <form className="mt-5 flex flex-wrap items-end gap-3" onSubmit={look}>
        <Field
          className="w-full max-w-[18rem]"
          htmlFor="recognition-handle"
          label="Handle"
        >
          <TextInput
            autoComplete="off"
            id="recognition-handle"
            onChange={(event) => setTyped(event.target.value)}
            placeholder="kestrel.writes"
            value={typed}
          />
        </Field>
        <Button
          disabled={!typed.trim()}
          loading={looking}
          type="submit"
          variant="secondary"
        >
          <Search aria-hidden="true" />
          Look up
        </Button>
      </form>

      {account && held ? (
        <>
          <div className="mt-8 flex flex-wrap items-center justify-between gap-4 rounded-plate bg-deep px-5 py-4">
            <div className="min-w-0">
              <h3 className="font-display text-title font-medium tracking-[-0.02em] text-ink wrap-anywhere">
                <Link
                  className="inline-flex min-h-11 items-center text-ink outline-offset-3 hover:text-accent"
                  href={`/@${account.handle}`}
                >
                  @{account.handle}
                </Link>
              </h3>
              <p className="mt-1 font-prose text-meta text-mute">
                {whatIsHeld(account.assignments)}
              </p>
            </div>
            <StartAction
              disabled={offered.length === 0}
              icon={UserRoundCheck}
              onClick={onGive}
            >
              Give one
            </StartAction>
          </div>

          <HeldGroup
            hint="Every one of them shows, in the order set under Illarin positions."
            title="Illarin positions"
          >
            {held.positions.length === 0 ? (
              <p className="mt-3 font-prose text-meta text-mute">
                No job here yet.
              </p>
            ) : (
              <ul className="mt-2 flex list-none flex-col">
                {held.positions.map((one) => (
                  <HeldRow
                    key={one.id}
                    lead={<HeldMark one={one.distinction} />}
                    name={one.distinction.name}
                  >
                    <RowAction onClick={() => takeBack(one)}>
                      Take back
                    </RowAction>
                  </HeldRow>
                ))}
              </ul>
            )}
          </HeldGroup>

          <HeldGroup
            hint="A profile shows the first six titles and the first six badges, in this order."
            title="Titles and badges"
          >
            {held.earned.length === 0 ? (
              <p className="mt-3 font-prose text-meta text-mute">
                Nothing earned yet.
              </p>
            ) : (
              <ul className="mt-2 flex list-none flex-col">
                {held.earned.map((one) => (
                  <HeldRow
                    key={one.id}
                    lead={<HeldMark one={one.distinction} />}
                    name={one.distinction.name}
                    said={one.distinction.explanation}
                  >
                    <RowMove
                      disabled={!movedAssignments(account.assignments, one, -1)}
                      icon={ArrowUp}
                      label={`Move ${one.distinction.name} earlier`}
                      onClick={() => reorder(one, -1)}
                    />
                    <RowMove
                      disabled={!movedAssignments(account.assignments, one, 1)}
                      icon={ArrowDown}
                      label={`Move ${one.distinction.name} later`}
                      onClick={() => reorder(one, 1)}
                    />
                    <RowAction onClick={() => takeBack(one)}>
                      Take back
                    </RowAction>
                  </HeldRow>
                ))}
              </ul>
            )}
          </HeldGroup>

          {held.past.length > 0 ? (
            <Past summary={`${held.past.length} taken back`}>
              {held.past.map((one) => (
                <PastRow key={one.id}>
                  {one.distinction.name}, given {readableDate(one.assignedAt)}
                </PastRow>
              ))}
            </Past>
          ) : null}
        </>
      ) : (
        <Nothing>{nothingIn("accounts")}</Nothing>
      )}
    </>
  );
}

function HeldRow({
  children,
  lead,
  name,
  said,
}: {
  children: React.ReactNode;
  lead: React.ReactNode;
  name: string;
  said?: string;
}) {
  return (
    <li className="flex min-w-0 items-center gap-3 rounded-plate px-3 py-2.5 transition-colors duration-200 not-first:border-t not-first:border-rule/45 hover:border-transparent hover:bg-deep motion-reduce:transition-none">
      {lead}
      <div className="min-w-0 flex-1">
        <p className="font-ui text-ui font-medium text-ink wrap-anywhere">
          {name}
        </p>
        {said ? <p className="font-prose text-meta text-mute">{said}</p> : null}
      </div>
      <div className="flex shrink-0 items-center gap-1">{children}</div>
    </li>
  );
}

function HeldGroup({
  children,
  hint,
  title,
}: {
  children: React.ReactNode;
  hint: string;
  title: string;
}) {
  return (
    <section className="mt-8 min-w-0">
      <h4 className="font-ui text-ui font-medium text-ink">{title}</h4>
      <p className="mt-1 max-w-[52ch] font-prose text-meta text-mute">{hint}</p>
      {children}
    </section>
  );
}

function HeldMark({ one }: { one: Distinction }) {
  return (
    <span
      className={cn(
        "grid size-9 shrink-0 place-items-center overflow-hidden rounded-control",
        one.mark ? "bg-accent-wash text-accent" : "bg-deep text-mute",
      )}
    >
      {one.mark ? (
        <Image
          alt=""
          className="size-9 object-contain"
          height={36}
          src={one.mark.url}
          unoptimized
          width={36}
        />
      ) : one.form === "position" ? (
        <BriefcaseBusiness
          aria-hidden="true"
          className="size-4"
          strokeWidth={1.7}
        />
      ) : (
        <Type aria-hidden="true" className="size-4" strokeWidth={1.7} />
      )}
    </span>
  );
}

export function GiveStep({
  account,
  definitions,
  onFailure,
  onGiven,
}: {
  account: LookedUpAccount;
  definitions: Distinction[];
  onFailure: (message: string) => void;
  onGiven: (assignment: DistinctionAssignment) => void;
}) {
  const [giving, setGiving] = useState("");
  const offered = offeredTo(definitions, account.assignments);

  async function give(one: Distinction) {
    setGiving(one.id);
    const answer = await giveDistinction(account.handle, one.id);
    setGiving("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onGiven(answer.value);
  }

  if (offered.length === 0) {
    return (
      <p className="max-w-[52ch] font-prose text-prose text-mute">
        @{account.handle} holds everything Illarin gives out.
      </p>
    );
  }

  return (
    <ul className="-mx-4 flex list-none flex-col">
      {offered.map((one) => (
        <li
          className="flex min-w-0 items-center gap-3 rounded-plate px-4 py-3 not-first:border-t not-first:border-rule/45"
          key={one.id}
        >
          <HeldMark one={one} />
          <div className="min-w-0 flex-1">
            <p className="font-ui text-ui font-medium text-ink wrap-anywhere">
              {one.name}
            </p>
            <p className="font-prose text-meta text-mute">
              {one.form === "position" ? "A job at Illarin" : one.explanation}
            </p>
          </div>
          <RowAction busy={giving === one.id} onClick={() => give(one)}>
            Give
          </RowAction>
        </li>
      ))}
    </ul>
  );
}
