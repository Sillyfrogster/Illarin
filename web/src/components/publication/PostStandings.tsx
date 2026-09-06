"use client";

import { Field } from "@/components/console/Field";
import { STANDINGS, type Standing, standingName } from "@/lib/post-standing";
import styles from "./PostStandings.module.css";

export function PostStandings({
  chosen,
  counts,
  onChoose,
}: {
  chosen: Standing;
  counts: Record<Standing, number>;
  onChoose: (standing: Standing) => void;
}) {
  return (
    <>
      <fieldset className={styles.rail}>
        <legend className={styles.legend}>
          Which posts you are looking at
        </legend>
        {STANDINGS.map((standing) => (
          <button
            aria-pressed={chosen === standing}
            className={styles.standing}
            key={standing}
            onClick={() => onChoose(standing)}
            type="button"
          >
            <span className={styles.name}>{standingName(standing)}</span>
            <span className={styles.count}>{counts[standing]}</span>
          </button>
        ))}
      </fieldset>

      <div className={styles.compact}>
        <Field htmlFor="which-posts" label="Which posts">
          <select
            id="which-posts"
            onChange={(event) => onChoose(event.target.value as Standing)}
            value={chosen}
          >
            {STANDINGS.map((standing) => (
              <option key={standing} value={standing}>
                {standingName(standing)} ({counts[standing]})
              </option>
            ))}
          </select>
        </Field>
      </div>
    </>
  );
}
