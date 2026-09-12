import { type CodeRun, highlightCode } from "@/lib/code-highlight";
import { isPostLanguage } from "@/lib/post-document";

const REQUEST_LINE = /^([A-Z]+)( .*)$/;

const HEADER_LINE = /^([A-Za-z-]+)(:.*)$/;

/** Splits one code example into runs the code block can colour. */
export function codeRuns(source: string, language: string): CodeRun[] {
  if (language === "http") return httpRuns(source);
  if (isPostLanguage(language)) return highlightCode(source, language);
  return [{ text: source }];
}

function httpRuns(source: string): CodeRun[] {
  const [head, ...rest] = source.split("\n\n");
  const lines = head.split("\n");
  const runs: CodeRun[] = [];
  lines.forEach((line, at) => {
    const tail = at < lines.length - 1 ? "\n" : "";
    const request = at === 0 ? REQUEST_LINE.exec(line) : null;
    const header = at > 0 ? HEADER_LINE.exec(line) : null;
    if (request) {
      runs.push(
        { text: request[1], role: "keyword" },
        { text: request[2] + tail },
      );
    } else if (header) {
      runs.push({ text: header[1], role: "name" }, { text: header[2] + tail });
    } else {
      runs.push({ text: line + tail });
    }
  });
  if (rest.length === 0) return runs;
  const body = rest.join("\n\n");
  runs.push({ text: "\n\n" });
  return [...runs, ...highlightCode(body, "json")];
}
