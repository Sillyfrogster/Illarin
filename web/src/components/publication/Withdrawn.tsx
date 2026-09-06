import { Button } from "@/components/ui/Button";
import { WITHDRAWAL_MESSAGE } from "@/lib/publication-withdrawal";
import styles from "./Withdrawn.module.css";

/** The whole of a withdrawn address: what happened, anything Illarin chose to say, and the way on. */
export function Withdrawn({ explanation }: { explanation: string }) {
  return (
    <section className={styles.gone}>
      <h1 className={styles.statement}>{WITHDRAWAL_MESSAGE}</h1>
      {explanation ? <p className={styles.said}>{explanation}</p> : null}
      <Button
        className={styles.onward}
        href="/blog"
        size="large"
        variant="solid"
      >
        Read the rest of the blog
      </Button>
    </section>
  );
}
