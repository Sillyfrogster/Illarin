import { highlightCode, LANGUAGE_LABELS } from "@/lib/code-highlight";
import type { PostLanguage } from "@/lib/post-document";

/** Structure carries the reading, not colour, so the runs differ only in weight and tone. */
const RUN: Record<string, string> = {
  comment: "text-mute italic",
  keyword: "font-semibold",
  name: "text-ink",
  literal: "text-mute",
};

export function PostCode({
  language,
  source,
}: {
  language: PostLanguage;
  source: string;
}) {
  const label = LANGUAGE_LABELS[language] ?? LANGUAGE_LABELS.plain;
  return (
    <div className="my-8 rounded-plate bg-deep py-4">
      {language === "plain" ? null : (
        <span
          aria-hidden="true"
          className="block px-5 pb-1 text-right text-label font-semibold tracking-[0.06em] text-mute uppercase"
        >
          {label}
        </span>
      )}
      <section
        aria-label={`${label} code`}
        className="overflow-x-auto px-5"
        // biome-ignore lint/a11y/noNoninteractiveTabindex: A region that scrolls has to be reachable by keyboard.
        tabIndex={0}
      >
        <pre className="font-mono text-meta leading-7 [tab-size:2]">
          <code className="block min-w-max whitespace-pre">
            {highlightCode(source, language).map((run, index) => (
              <span
                className={run.role ? RUN[run.role] : undefined}
                // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the code and hold no local state.
                key={index}
              >
                {run.text}
              </span>
            ))}
          </code>
        </pre>
      </section>
    </div>
  );
}
