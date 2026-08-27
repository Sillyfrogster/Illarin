"use client";

import { ArrowDown, ArrowUp, Search, X } from "lucide-react";
import Image from "next/image";
import { type FormEvent, useState } from "react";
import {
  giveDistinction,
  orderAccountDistinctions,
  readAccountDistinctions,
  takeBackDistinction,
} from "@/lib/api/distinctions";
import type { Distinction, DistinctionAssignment } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./AccountDistinctions.module.css";

const FORM_NAMES: Record<string, string> = {
  position: "Illarin position",
  title: "Title",
  badge: "Badge",
};

const GROUPS: { form: string; heading: string; ordered: boolean }[] = [
  { form: "position", heading: "Illarin positions", ordered: false },
  { form: "title", heading: "Titles", ordered: true },
  { form: "badge", heading: "Badges", ordered: true },
];

export function AccountDistinctions({
  definitions,
}: {
  definitions: Distinction[];
}) {
  const [typed, setTyped] = useState("");
  const [handle, setHandle] = useState("");
  const [assignments, setAssignments] = useState<DistinctionAssignment[]>([]);
  const [failure, setFailure] = useState("");
  const [looking, setLooking] = useState(false);
  const [chosen, setChosen] = useState("");

  const active = assignments.filter((assignment) => assignment.active);
  const past = assignments.filter((assignment) => !assignment.active);
  const held = new Set(active.map((assignment) => assignment.distinction.id));
  const offered = definitions.filter(
    (definition) => !definition.retired && !held.has(definition.id),
  );

  async function look(event: FormEvent) {
    event.preventDefault();
    const wanted = typed.trim().replace(/^@/, "");
    if (!wanted || looking) return;
    setLooking(true);
    const answer = await readAccountDistinctions(wanted);
    setLooking(false);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      setHandle("");
      setAssignments([]);
      return;
    }
    setFailure("");
    setHandle(answer.value.handle);
    setAssignments(answer.value.assignments);
    setChosen("");
  }

  async function give() {
    if (!chosen || !handle) return;
    const answer = await giveDistinction(handle, chosen);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setChosen("");
    setAssignments([...assignments, answer.value]);
  }

  async function takeBack(assignment: DistinctionAssignment) {
    const answer = await takeBackDistinction(handle, assignment.id);
    if (answer.error) {
      setFailure(answer.error);
      return;
    }
    setFailure("");
    setAssignments(
      assignments.map((one) =>
        one.id === assignment.id ? { ...one, active: false } : one,
      ),
    );
  }

  async function reorder(
    group: DistinctionAssignment[],
    index: number,
    step: number,
  ) {
    const target = index + step;
    if (target < 0 || target >= group.length) return;
    const order = active.map((assignment) => assignment.id);
    const answer = await orderAccountDistinctions(
      handle,
      moved(
        order,
        order.indexOf(group[index].id),
        order.indexOf(group[target].id),
      ),
    );
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setAssignments(answer.value.assignments);
  }

  return (
    <section className={styles.region}>
      <header className={styles.heading}>
        <h2>Give them to an account</h2>
        <p>
          A profile shows every position this account holds, in the order set
          for the positions themselves. Titles and badges show in the order you
          set here.
        </p>
      </header>

      <form className={styles.lookup} onSubmit={look}>
        <label className="sr-only" htmlFor="account-handle">
          Handle
        </label>
        <input
          id="account-handle"
          value={typed}
          placeholder="Handle"
          autoComplete="off"
          onChange={(event) => setTyped(event.target.value)}
        />
        <button type="submit" disabled={looking || !typed.trim()}>
          <Search size={15} strokeWidth={2} aria-hidden="true" />
          Look up
        </button>
      </form>

      {failure ? (
        <p className={styles.failure} role="alert">
          {failure}
        </p>
      ) : null}

      {handle ? (
        <div className={styles.account}>
          <h3>@{handle}</h3>
          {active.length > 0 ? (
            GROUPS.map((group) => {
              const rows = active.filter(
                (assignment) => assignment.distinction.form === group.form,
              );
              if (rows.length === 0) return null;
              return (
                <div className={styles.group} key={group.form}>
                  <h4>{group.heading}</h4>
                  <ol className={styles.list}>
                    {rows.map((assignment, index) => (
                      <li className={styles.row} key={assignment.id}>
                        {assignment.distinction.mark ? (
                          <Image
                            className={styles.mark}
                            src={assignment.distinction.mark.url}
                            alt=""
                            width={26}
                            height={26}
                            unoptimized
                          />
                        ) : null}
                        <span className={styles.name}>
                          {assignment.distinction.name}
                        </span>
                        <span className={styles.rowActions}>
                          {group.ordered ? (
                            <>
                              <button
                                type="button"
                                onClick={() => reorder(rows, index, -1)}
                                disabled={index === 0}
                                aria-label={`Move ${assignment.distinction.name} earlier on @${handle}`}
                              >
                                <ArrowUp
                                  size={15}
                                  strokeWidth={1.8}
                                  aria-hidden="true"
                                />
                              </button>
                              <button
                                type="button"
                                onClick={() => reorder(rows, index, 1)}
                                disabled={index === rows.length - 1}
                                aria-label={`Move ${assignment.distinction.name} later on @${handle}`}
                              >
                                <ArrowDown
                                  size={15}
                                  strokeWidth={1.8}
                                  aria-hidden="true"
                                />
                              </button>
                            </>
                          ) : null}
                          <button
                            type="button"
                            onClick={() => takeBack(assignment)}
                            aria-label={`Take back ${assignment.distinction.name}`}
                          >
                            <X size={15} strokeWidth={1.8} aria-hidden="true" />
                          </button>
                        </span>
                      </li>
                    ))}
                  </ol>
                </div>
              );
            })
          ) : (
            <p className={styles.empty}>This account holds none of them yet.</p>
          )}

          <div className={styles.give}>
            <label className="sr-only" htmlFor="give-distinction">
              What to give
            </label>
            <select
              id="give-distinction"
              value={chosen}
              onChange={(event) => setChosen(event.target.value)}
            >
              <option value="">Choose one to give…</option>
              {offered.map((definition) => (
                <option key={definition.id} value={definition.id}>
                  {FORM_NAMES[definition.form]} — {definition.name}
                </option>
              ))}
            </select>
            <button type="button" onClick={give} disabled={!chosen}>
              Give
            </button>
          </div>

          {past.length > 0 ? (
            <div className={styles.past}>
              <h4>Taken back</h4>
              <ul>
                {past.map((assignment) => (
                  <li key={assignment.id}>
                    {assignment.distinction.name}
                    <span>
                      {new Date(assignment.assignedAt).toLocaleDateString()}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
