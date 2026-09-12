import { CopyButton } from "@/components/ui/copy-button";
import { LANGUAGE_LABELS } from "@/lib/code-highlight";
import { codeRuns } from "@/lib/docs/code-runs";
import { isPostLanguage } from "@/lib/post-document";

const RUN: Record<string, string> = {
  comment: "text-mute italic",
  keyword: "font-semibold text-accent",
  name: "text-ink",
  literal: "text-mute",
};

const LABELS: Record<string, string> = {
  http: "HTTP",
  python: "Python",
};

export function DocCode({
  label,
  language,
  source,
}: {
  label: string;
  language: string;
  source: string;
}) {
  const shown = label === language ? languageLabel(language) : label;
  return (
    <figure className="my-4 overflow-hidden rounded-plate bg-deep">
      <figcaption className="flex min-h-11 items-center justify-between gap-3 border-b border-rule/60 pl-4">
        <span className="min-w-0 truncate font-ui text-label font-medium text-mute">
          {shown}
        </span>
        <CopyButton label={`Copy ${shown}`} text={source} />
      </figcaption>
      <div
        className="overflow-x-auto px-4 py-3"
        // biome-ignore lint/a11y/noNoninteractiveTabindex: A region that scrolls has to be reachable by keyboard.
        tabIndex={0}
      >
        <pre className="font-mono text-meta leading-6 [tab-size:2]">
          <code className="block min-w-max whitespace-pre">
            {codeRuns(source, language).map((run, index) => (
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
      </div>
    </figure>
  );
}

function languageLabel(language: string): string {
  if (isPostLanguage(language)) return LANGUAGE_LABELS[language];
  return LABELS[language] ?? language;
}
