import bash from "highlight.js/lib/languages/bash";
import css from "highlight.js/lib/languages/css";
import diff from "highlight.js/lib/languages/diff";
import go from "highlight.js/lib/languages/go";
import ini from "highlight.js/lib/languages/ini";
import javascript from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import markdown from "highlight.js/lib/languages/markdown";
import python from "highlight.js/lib/languages/python";
import rust from "highlight.js/lib/languages/rust";
import sql from "highlight.js/lib/languages/sql";
import typescript from "highlight.js/lib/languages/typescript";
import xml from "highlight.js/lib/languages/xml";
import yaml from "highlight.js/lib/languages/yaml";
import { createLowlight } from "lowlight";
import type { PostLanguage } from "./post-body";

export const LANGUAGE_LABELS: Record<PostLanguage, string> = {
  plain: "Plain text",
  bash: "Bash",
  css: "CSS",
  diff: "Diff",
  go: "Go",
  html: "HTML",
  javascript: "JavaScript",
  json: "JSON",
  markdown: "Markdown",
  python: "Python",
  rust: "Rust",
  sql: "SQL",
  toml: "TOML",
  typescript: "TypeScript",
  yaml: "YAML",
};

export type CodeRole = "comment" | "keyword" | "literal" | "name";

export type CodeRun = { text: string; role?: CodeRole };

const ROLES: Record<string, CodeRole> = {
  comment: "comment",
  quote: "comment",
  doctag: "comment",
  keyword: "keyword",
  built_in: "keyword",
  literal: "keyword",
  type: "keyword",
  meta: "keyword",
  "meta keyword": "keyword",
  section: "keyword",
  selector_tag: "keyword",
  tag: "keyword",
  string: "literal",
  regexp: "literal",
  char: "literal",
  number: "literal",
  symbol: "literal",
  addition: "literal",
  deletion: "literal",
  "meta string": "literal",
  title: "name",
  "title function_": "name",
  "title class_": "name",
  attr: "name",
  attribute: "name",
  property: "name",
  variable: "name",
  "variable language_": "name",
  params: "name",
  name: "name",
  selector_class: "name",
  selector_id: "name",
};

const lowlight = createLowlight({
  bash,
  css,
  diff,
  go,
  html: xml,
  javascript,
  json,
  markdown,
  python,
  rust,
  sql,
  toml: ini,
  typescript,
  yaml,
});

export function highlightCode(
  source: string,
  language: PostLanguage,
): CodeRun[] {
  if (language === "plain" || !lowlight.registered(language)) {
    return [{ text: source }];
  }
  const runs: CodeRun[] = [];
  collect(lowlight.highlight(language, source), undefined, runs);
  return runs;
}

type HastNode = {
  type: string;
  value?: string;
  properties?: { className?: unknown };
  children?: HastNode[];
};

function collect(
  node: HastNode,
  inherited: CodeRole | undefined,
  runs: CodeRun[],
) {
  if (node.type === "text" && node.value) {
    const last = runs.at(-1);
    if (last && last.role === inherited) last.text += node.value;
    else
      runs.push(
        inherited
          ? { text: node.value, role: inherited }
          : { text: node.value },
      );
    return;
  }
  const role = roleOf(node.properties?.className) ?? inherited;
  for (const child of node.children ?? []) collect(child, role, runs);
}

function roleOf(className: unknown): CodeRole | undefined {
  if (!Array.isArray(className)) return undefined;
  const scope = className
    .filter((one): one is string => typeof one === "string")
    .map((one) => one.replace(/^hljs-/, ""))
    .join(" ");
  return ROLES[scope] ?? ROLES[scope.split(" ")[0]];
}
