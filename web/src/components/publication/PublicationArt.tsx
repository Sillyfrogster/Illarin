import styles from "./PublicationArt.module.css";

/** The paired artwork behind the blog's mastheads, with the scrim that keeps copy readable. */
export function PublicationArt() {
  return (
    <>
      <span aria-hidden="true" className={styles.art} />
      <span aria-hidden="true" className={styles.scrim} />
    </>
  );
}
