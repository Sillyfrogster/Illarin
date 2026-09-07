"use client";

import { CircleCheck, CircleDashed, CircleSlash, Plus } from "lucide-react";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Section } from "@/components/console/Section";
import { disableDestination, verifyDestination } from "@/lib/api/publication";
import type { PublicationDestination } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import { EVENT_WORDS } from "@/lib/publication-delivery";
import { DestinationDialog } from "./DestinationDialog";
import styles from "./DestinationList.module.css";
import hub from "./PublicationHub.module.css";
import { RotateSecret } from "./RotateSecret";

export function DestinationList({
  destinations,
  onChanged,
  onFailure,
}: {
  destinations: PublicationDestination[];
  onChanged: (destinations: PublicationDestination[]) => void;
  onFailure: (message: string) => void;
}) {
  const [editing, setEditing] = useState<PublicationDestination | null>(null);
  const [rotating, setRotating] = useState<PublicationDestination | null>(null);
  const [adding, setAdding] = useState(false);
  const [working, setWorking] = useState("");

  function replace(saved: PublicationDestination, added: boolean) {
    onChanged(
      added
        ? [...destinations, saved]
        : destinations.map((one) => (one.id === saved.id ? saved : one)),
    );
    if (editing?.id === saved.id) setEditing(saved);
  }

  async function prove(one: PublicationDestination) {
    setWorking(one.id);
    const answer = await verifyDestination(one.id);
    setWorking("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    replace(answer.value, false);
  }

  async function switchOff(one: PublicationDestination) {
    setWorking(one.id);
    const answer = await disableDestination(one.id);
    setWorking("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    replace(answer.value, false);
  }

  return (
    <Section
      title="Destinations"
      count={destinations.length}
      action={
        <button
          type="button"
          className={hub.add}
          onClick={() => setAdding(true)}
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          Add a destination
        </button>
      }
    >
      {destinations.length === 0 ? (
        <p className={rows.empty}>
          Nowhere is set up to receive an announcement, so every post publishes
          quietly.
        </p>
      ) : (
        <ul className={rows.list}>
          {destinations.map((one) => (
            <li className={rows.row} key={one.id}>
              <span className={styles.state} data-state={one.state}>
                <StateMark state={one.state} />
              </span>
              <span className={rows.name}>
                {one.name} <span className={rows.slug}>{one.host}</span>
              </span>
              <span className={rows.detail}>
                {standing(one)}
                <span className={styles.takes}>{takes(one)}</span>
              </span>
              <span className={rows.actions}>
                {one.state === "active" ? (
                  <button
                    type="button"
                    className={rows.textButton}
                    disabled={working === one.id}
                    onClick={() => switchOff(one)}
                  >
                    Switch off
                  </button>
                ) : (
                  <button
                    type="button"
                    className={rows.textButton}
                    disabled={working === one.id}
                    onClick={() => prove(one)}
                  >
                    {working === one.id ? "Asking…" : "Verify"}
                  </button>
                )}
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setRotating(one)}
                >
                  New secret
                </button>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setEditing(one)}
                >
                  Edit
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      {rotating ? (
        <RotateSecret
          destination={rotating}
          key={rotating.id}
          onClose={() => setRotating(null)}
          onFailure={onFailure}
          onRotated={(saved) => replace(saved, false)}
        />
      ) : null}
      {adding ? (
        <DestinationDialog
          key="adding"
          existing={null}
          onClose={() => setAdding(false)}
          onFailure={onFailure}
          onRemoved={() => undefined}
          onSaved={replace}
        />
      ) : null}
      {editing ? (
        <DestinationDialog
          key={editing.id}
          existing={editing}
          onClose={() => setEditing(null)}
          onFailure={onFailure}
          onRemoved={() =>
            onChanged(destinations.filter((one) => one.id !== editing.id))
          }
          onSaved={replace}
        />
      ) : null}
    </Section>
  );
}

function StateMark({ state }: { state: PublicationDestination["state"] }) {
  if (state === "active") {
    return <CircleCheck size={18} strokeWidth={1.8} aria-hidden="true" />;
  }
  if (state === "disabled") {
    return <CircleSlash size={18} strokeWidth={1.8} aria-hidden="true" />;
  }
  return <CircleDashed size={18} strokeWidth={1.8} aria-hidden="true" />;
}

// standing says in one line what this endpoint is doing and since when.
function standing(one: PublicationDestination): string {
  if (
    one.previousSecretUntil &&
    new Date(one.previousSecretUntil) > new Date()
  ) {
    return `Both signing secrets are accepted until ${readableDate(one.previousSecretUntil)}.`;
  }
  if (one.state === "active" && one.verifiedAt) {
    return `Receiving. Proved it was listening on ${readableDate(one.verifiedAt)}.`;
  }
  if (one.state === "disabled") {
    return "Switched off. Verify it again to start sending here.";
  }
  return "Waiting to prove it is listening. Nothing is sent until it does.";
}

// takes names the public transitions this endpoint asked for.
function takes(one: PublicationDestination): string {
  if (one.events.length === 0) return "Takes nothing.";
  return one.events.map((event) => EVENT_WORDS[event].word).join(" · ");
}
