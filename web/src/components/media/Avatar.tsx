import Image from "next/image";
import styles from "./Avatar.module.css";

type Portrait = { url: string; width: number; height: number };

/** Spreads handles across the tonal steps without storing a value. */
function shadeOf(handle: string) {
  let hash = 0;
  for (const character of handle) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 100000;
  }
  return hash;
}

export function Avatar({
  handle,
  portrait,
  size = "md",
}: {
  handle: string;
  portrait?: Portrait | null;
  size?: "sm" | "md" | "lg";
}) {
  if (portrait) {
    return (
      <span className={styles.avatar} data-size={size}>
        <Image
          className={styles.portrait}
          src={portrait.url}
          alt=""
          fill
          sizes="256px"
          unoptimized
        />
      </span>
    );
  }

  return (
    <span
      className={styles.avatar}
      data-size={size}
      data-tone={shadeOf(handle) % 4}
      aria-hidden="true"
    >
      <span className={styles.initial}>{handle.slice(0, 1).toUpperCase()}</span>
    </span>
  );
}
