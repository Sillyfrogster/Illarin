import { highlightCode, LANGUAGE_LABELS } from "@/lib/code-highlight";
import type { PostLanguage } from "@/lib/post-document";
import styles from "./PostBody.module.css";

export function PostCode({
  language,
  source,
}: {
  language: PostLanguage;
  source: string;
}) {
  const label = LANGUAGE_LABELS[language] ?? LANGUAGE_LABELS.plain;
  return (
    <div className={styles.code}>
      {language === "plain" ? null : (
        <span aria-hidden="true" className={styles.codeLabel}>
          {label}
        </span>
      )}
      <section
        aria-label={`${label} code`}
        className={styles.codeScroll}
        // biome-ignore lint/a11y/noNoninteractiveTabindex: A region that scrolls has to be reachable by keyboard.
        tabIndex={0}
      >
        <pre>
          <code>
            {highlightCode(source, language).map((run, index) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the code and hold no local state.
              <span className={run.role && styles[run.role]} key={index}>
                {run.text}
              </span>
            ))}
          </code>
        </pre>
      </section>
    </div>
  );
}
