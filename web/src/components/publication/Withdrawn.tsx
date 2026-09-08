import Link from "next/link";
import { Button } from "@/components/ui/button";
import { WITHDRAWAL_MESSAGE } from "@/lib/publication-withdrawal";
import styles from "./Withdrawn.module.css";

/** The whole of a withdrawn address: what happened, anything Illarin chose to say, and the way on. */
export function Withdrawn({ explanation }: { explanation: string }) {
  return (
    <section className={styles.gone}>
      <h1 className={styles.statement}>{WITHDRAWAL_MESSAGE}</h1>
      {explanation ? <p className={styles.said}>{explanation}</p> : null}
      <Button asChild className={styles.onward} size="large" variant="primary">
        <Link href="/blog">Read the rest of the blog</Link>
      </Button>
    </section>
  );
}
