"use client";

import type { PublicationDestination } from "@/lib/api/query";
import styles from "./CategoryChoice.module.css";
import own from "./DestinationChoice.module.css";

/** Which destinations a policy allows, and which of them a publication starts with. */
export function DestinationChoice({
  name,
  legend,
  destinations,
  allowed,
  defaults,
  inherit,
  onAllowed,
  onDefaults,
  onInherit,
}: {
  name: string;
  legend: string;
  destinations: PublicationDestination[];
  allowed: string[];
  defaults: string[];
  inherit?: { label: string; on: boolean };
  onAllowed: (allowed: string[]) => void;
  onDefaults: (defaults: string[]) => void;
  onInherit?: (on: boolean) => void;
}) {
  const offered = destinations.filter((one) => one.state !== "disabled");

  function toggle(id: string, on: boolean) {
    onAllowed(on ? [...allowed, id] : allowed.filter((held) => held !== id));
    if (!on) onDefaults(defaults.filter((held) => held !== id));
  }

  function toggleDefault(id: string, on: boolean) {
    onDefaults(on ? [...defaults, id] : defaults.filter((held) => held !== id));
  }

  return (
    <fieldset className={styles.choice}>
      <legend>{legend}</legend>
      {inherit && onInherit ? (
        <label className={own.inherit}>
          <input
            type="checkbox"
            checked={inherit.on}
            onChange={(event) => onInherit(event.target.checked)}
          />
          <span>{inherit.label}</span>
        </label>
      ) : null}
      {inherit?.on ? null : offered.length === 0 ? (
        <p className={own.none}>
          Nothing is set up to receive an announcement yet.
        </p>
      ) : (
        <ul>
          {offered.map((one) => {
            const on = allowed.includes(one.id);
            return (
              <li className={styles.line} key={one.id}>
                <label className={styles.allow}>
                  <input
                    type="checkbox"
                    checked={on}
                    onChange={(event) => toggle(one.id, event.target.checked)}
                  />
                  <span className={styles.label}>{one.name}</span>
                  <span className={styles.slug}>{one.host}</span>
                </label>
                <label className={styles.fallback}>
                  <input
                    type="checkbox"
                    name={`${name}-default`}
                    checked={defaults.includes(one.id)}
                    disabled={!on}
                    onChange={(event) =>
                      toggleDefault(one.id, event.target.checked)
                    }
                  />
                  Default
                </label>
              </li>
            );
          })}
        </ul>
      )}
    </fieldset>
  );
}
