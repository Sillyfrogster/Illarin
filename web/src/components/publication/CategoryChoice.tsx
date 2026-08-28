"use client";

import type { PublicationCategory } from "@/lib/api/query";
import styles from "./CategoryChoice.module.css";

export function CategoryChoice({
  name,
  categories,
  allowed,
  fallback,
  onAllowed,
  onFallback,
}: {
  name: string;
  categories: PublicationCategory[];
  allowed: string[];
  fallback: string;
  onAllowed: (allowed: string[]) => void;
  onFallback: (fallback: string) => void;
}) {
  const offered = categories.filter((category) => !category.retired);

  function toggle(id: string, on: boolean) {
    const next = on
      ? [...allowed, id]
      : allowed.filter((allowedId) => allowedId !== id);
    onAllowed(next);
    if (!on && fallback === id) onFallback(next[0] ?? "");
    if (on && !fallback) onFallback(id);
  }

  return (
    <fieldset className={styles.choice}>
      <legend>What they may publish</legend>
      <ul>
        {offered.map((category) => {
          const on = allowed.includes(category.id);
          return (
            <li className={styles.line} key={category.id}>
              <label className={styles.allow}>
                <input
                  type="checkbox"
                  checked={on}
                  onChange={(event) =>
                    toggle(category.id, event.target.checked)
                  }
                />
                <span className={styles.label}>{category.label}</span>
                <span className={styles.slug}>{category.slug}</span>
              </label>
              <label className={styles.fallback}>
                <input
                  type="radio"
                  name={`${name}-default`}
                  checked={fallback === category.id}
                  disabled={!on}
                  onChange={() => onFallback(category.id)}
                />
                Default
              </label>
            </li>
          );
        })}
      </ul>
    </fieldset>
  );
}
