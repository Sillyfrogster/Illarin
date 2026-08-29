import type { PostSpan } from "@/lib/post-document";
import { leavesIllarin } from "@/lib/post-link";
import styles from "./PostBody.module.css";

export function PostSpans({ spans }: { spans: PostSpan[] }) {
  return (
    <>
      {spans.map((span, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the writing and hold no local state.
        <Span key={index} span={span} />
      ))}
    </>
  );
}

function Span({ span }: { span: PostSpan }) {
  let rendered = <>{span.text}</>;
  for (const mark of span.marks ?? []) {
    switch (mark.type) {
      case "code":
        rendered = <code className={styles.inlineCode}>{rendered}</code>;
        break;
      case "strike":
        rendered = <s>{rendered}</s>;
        break;
      case "italic":
        rendered = <em>{rendered}</em>;
        break;
      case "bold":
        rendered = <strong>{rendered}</strong>;
        break;
      case "link":
        rendered = <Away href={mark.href}>{rendered}</Away>;
        break;
    }
  }
  return rendered;
}

function Away({ href, children }: { href: string; children: React.ReactNode }) {
  const away = leavesIllarin(href);
  return (
    <a
      className={styles.link}
      href={href}
      rel="noreferrer nofollow"
      target={away ? "_blank" : undefined}
    >
      {children}
      {away ? (
        <span className={styles.aside}> (opens in a new tab)</span>
      ) : null}
    </a>
  );
}
