import Image from "next/image";
import styles from "./CreatorMark.module.css";

type Portrait = { url: string; width: number; height: number };

/** Spreads handles across the tonal steps and mark angles without storing a value. */
function shadeOf(handle: string) {
  let hash = 0;
  for (const character of handle) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 100000;
  }
  return hash;
}

export function CreatorMark({
  handle,
  portrait,
  compact = false,
}: {
  handle: string;
  portrait?: Portrait | null;
  compact?: boolean;
}) {
  const hash = shadeOf(handle);

  if (portrait) {
    return (
      <span className={styles.mark} data-compact={compact || undefined}>
        <Image
          className={styles.portrait}
          src={portrait.url}
          alt=""
          fill
          sizes="128px"
          unoptimized
        />
      </span>
    );
  }

  return (
    <span
      className={styles.mark}
      data-tone={hash % 4}
      data-compact={compact || undefined}
      style={{ "--mark-angle": `${hash % 90}deg` } as React.CSSProperties}
      aria-hidden="true"
    >
      <span className={styles.figure} />
      <span className={styles.initial}>{handle.slice(0, 1).toUpperCase()}</span>
    </span>
  );
}
