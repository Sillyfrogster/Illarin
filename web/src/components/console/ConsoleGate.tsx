import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import styles from "./ConsoleGate.module.css";

export function ConsoleGate({
  heading,
  line,
  href,
  action,
}: {
  heading: string;
  line: string;
  href: string;
  action: string;
}) {
  return (
    <section className={styles.gate}>
      <ShieldCheck size={26} strokeWidth={1.35} aria-hidden="true" />
      <h2>{heading}</h2>
      <p>{line}</p>
      <Link href={href}>{action}</Link>
    </section>
  );
}
