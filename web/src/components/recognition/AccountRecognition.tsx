"use client";

import { ArrowDown, ArrowUp, Search, X } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { type FormEvent, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Section } from "@/components/console/Section";
import {
  giveDistinction,
  orderAccountDistinctions,
  readAccountDistinctions,
  takeBackDistinction,
} from "@/lib/api/distinctions";
import type { Distinction, DistinctionAssignment } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./AccountRecognition.module.css";

export function AccountRecognition({
  definitions,
  onFailure,
}: {
  definitions: Distinction[];
  onFailure: (message: string) => void;
}) {
  const [typed, setTyped] = useState("");
  const [handle, setHandle] = useState("");
  const [assignments, setAssignments] = useState<DistinctionAssignment[]>([]);
  const [looking, setLooking] = useState(false);
  const [chosen, setChosen] = useState("");

  const active = assignments.filter((one) => one.active);
  const past = assignments.filter((one) => !one.active);
  const positions = active.filter((one) => one.distinction.form === "position");
  const earned = active.filter((one) => one.distinction.form !== "position");
  const held = new Set(active.map((one) => one.distinction.id));
  const offered = definitions.filter(
    (one) => !one.retired && !held.has(one.id),
  );

  async function look(event: FormEvent) {
    event.preventDefault();
    const wanted = typed.trim().replace(/^@/, "");
    if (!wanted || looking) return;
    setLooking(true);
    const answer = await readAccountDistinctions(wanted);
    setLooking(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      setHandle("");
      setAssignments([]);
      return;
    }
    onFailure("");
    setHandle(answer.value.handle);
    setAssignments(answer.value.assignments);
    setChosen("");
  }

  async function give() {
    if (!chosen || !handle) return;
    const answer = await giveDistinction(handle, chosen);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    setChosen("");
    setAssignments([...assignments, answer.value]);
  }

  async function takeBack(assignment: DistinctionAssignment) {
    const answer = await takeBackDistinction(handle, assignment.id);
    if (answer.error) {
      onFailure(answer.error);
      return;
    }
    onFailure("");
    setAssignments(
      assignments.map((one) =>
        one.id === assignment.id ? { ...one, active: false } : one,
      ),
    );
  }

  async function reorder(assignment: DistinctionAssignment, step: number) {
    const at = earned.indexOf(assignment);
    const target = at + step;
    if (target < 0 || target >= earned.length) return;
    const order = active.map((one) => one.id);
    const answer = await orderAccountDistinctions(
      handle,
      moved(
        order,
        order.indexOf(earned[at].id),
        order.indexOf(earned[target].id),
      ),
    );
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    setAssignments(answer.value.assignments);
  }

  return (
    <Section
      title="Give it to someone"
      retiredLabel={past.length > 0 ? `${past.length} taken back` : undefined}
      retired={
        past.length > 0 ? (
          <ul className={rows.pastList}>
            {past.map((one) => (
              <li className={rows.pastRow} key={one.id}>
                <span>{one.distinction.name}</span>
                {new Date(one.assignedAt).toLocaleDateString()}
              </li>
            ))}
          </ul>
        ) : null
      }
    >
      <form className={styles.lookup} onSubmit={look}>
        <label className="sr-only" htmlFor="recognition-handle">
          Handle
        </label>
        <input
          id="recognition-handle"
          value={typed}
          placeholder="Handle"
          autoComplete="off"
          onChange={(event) => setTyped(event.target.value)}
        />
        <button type="submit" disabled={looking || !typed.trim()}>
          <Search size={15} strokeWidth={2} aria-hidden="true" />
          {looking ? "Looking" : "Look up"}
        </button>
      </form>

      {handle ? (
        <div className={styles.account}>
          <div className={styles.who}>
            <Link href={`/${handle}`}>@{handle}</Link>
            <span>
              {positions.length} {positions.length === 1 ? "job" : "jobs"} ·{" "}
              {earned.length} earned
            </span>
            <div className={styles.give}>
              <label className="sr-only" htmlFor="give-recognition">
                What to give
              </label>
              <select
                id="give-recognition"
                value={chosen}
                onChange={(event) => setChosen(event.target.value)}
              >
                <option value="">Choose one to give…</option>
                {offered.map((one) => (
                  <option key={one.id} value={one.id}>
                    {one.form === "position" ? "Position" : "Title or badge"} —{" "}
                    {one.name}
                  </option>
                ))}
              </select>
              <button type="button" onClick={give} disabled={!chosen}>
                Give
              </button>
            </div>
          </div>

          <Group
            heading="Illarin positions"
            hint="All of them show, in the order set under Illarin positions."
            plain
            empty="No job here yet."
            assignments={positions}
            onTakeBack={takeBack}
          />
          <Group
            heading="Titles and badges"
            hint="A profile shows the first six titles and the first six badges, in this order."
            empty="Nothing earned yet."
            assignments={earned}
            onTakeBack={takeBack}
            onReorder={reorder}
          />
        </div>
      ) : null}
    </Section>
  );
}

function Group({
  heading,
  hint,
  empty,
  plain,
  assignments,
  onTakeBack,
  onReorder,
}: {
  heading: string;
  hint: string;
  empty: string;
  plain?: boolean;
  assignments: DistinctionAssignment[];
  onTakeBack: (assignment: DistinctionAssignment) => void;
  onReorder?: (assignment: DistinctionAssignment, step: number) => void;
}) {
  return (
    <div className={styles.group}>
      <h3>{heading}</h3>
      <p className={styles.groupHint}>{hint}</p>
      {assignments.length === 0 ? (
        <p className={styles.none}>{empty}</p>
      ) : (
        <ol className={styles.held}>
          {assignments.map((one, index) => (
            <li className={styles.holding} key={one.id}>
              {plain ? null : (
                <span className={styles.holdingMark}>
                  {one.distinction.mark ? (
                    <Image
                      src={one.distinction.mark.url}
                      alt=""
                      width={26}
                      height={26}
                      unoptimized
                    />
                  ) : null}
                </span>
              )}
              <span className={styles.holdingName}>{one.distinction.name}</span>
              {onReorder ? (
                <>
                  <button
                    type="button"
                    className={rows.iconButton}
                    onClick={() => onReorder(one, -1)}
                    disabled={index === 0}
                    aria-label={`Move ${one.distinction.name} earlier`}
                  >
                    <ArrowUp size={14} strokeWidth={1.8} aria-hidden="true" />
                  </button>
                  <button
                    type="button"
                    className={rows.iconButton}
                    onClick={() => onReorder(one, 1)}
                    disabled={index === assignments.length - 1}
                    aria-label={`Move ${one.distinction.name} later`}
                  >
                    <ArrowDown size={14} strokeWidth={1.8} aria-hidden="true" />
                  </button>
                </>
              ) : null}
              <button
                type="button"
                className={rows.iconButton}
                onClick={() => onTakeBack(one)}
                aria-label={`Take back ${one.distinction.name}`}
              >
                <X size={14} strokeWidth={1.8} aria-hidden="true" />
              </button>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
