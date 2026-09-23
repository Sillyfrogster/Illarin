import { execFileSync } from "node:child_process";
import { readdir, readFile } from "node:fs/promises";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

type Occurrence = { text: string; file: string; line: number };

const scripts = dirname(fileURLToPath(import.meta.url));
const visibleNames =
  /^(?:alt|title|label|description|placeholder|summary|detail|message|text|heading|hint|children|aria-label|aria-description|emptyLabel|pendingLabel|confirmLabel)$/i;
const technicalNames =
  /^(?:className|class|style|id|key|href|src|srcSet|sizes|type|role|name|htmlFor|target|rel|method|action|autoComplete|inputMode|pattern|accept|variant|size|as|align|side|sideOffset|orientation|layout|width|height|color|fill|stroke|viewBox|d|xmlns|xmlnsXlink|property|httpEquiv|contentType|cache|credentials|mode|locale|numeric|dateStyle|timeStyle|timeZone|year|month|day|hour|minute|second|weekday|fontFamily|fontWeight|fontSize|display|position|overflow|padding|margin|border|background|justifyContent|alignItems|flexDirection|objectFit|textAlign|textTransform|whiteSpace|wordBreak|pointerEvents|userSelect|transition|transform|animation|filter|opacity|zIndex|boxShadow|borderRadius|gridTemplateColumns|gridColumn|gridRow|gap|inset|top|left|right|bottom|flex|flexWrap|flexShrink|flexGrow|fontStyle|lineHeight|letterSpacing|cursor|textDecoration|listStyle|outline|boxSizing|objectPosition|minWidth|maxWidth|minHeight|maxHeight|maskImage|backgroundImage|backgroundSize|backgroundPosition|backgroundRepeat|backdropFilter|appearance|resize|verticalAlign|clipPath|aspectRatio|scrollBehavior|touchAction|overscrollBehavior|Webkit.*|data-.*|aria-(?!label$|description$).*)$/;

function readable(text: string): boolean {
  return (
    text.trim().length > 0 &&
    !/^(?:SELECT|INSERT|UPDATE|DELETE|WITH|CREATE|ALTER)\b[\s\S]*\b(?:FROM|INTO|SET|TABLE|AS)\b/i.test(
      text.trim(),
    )
  );
}

function likelyCopy(text: string): boolean {
  if (!readable(text) || !/[\p{L}\p{N}]/u.test(text)) return false;
  const staticText = text.replace(/\$\{[^}]*\}/g, "").trim();
  if (!/[\p{L}\p{N}]/u.test(staticText)) return false;
  if (
    /^(?:https?:|\/|\.{1,2}\/|@\/|data:|image\/|application\/|text\/|--|#[\da-f]{3,8}$)/i.test(
      staticText,
    )
  )
    return false;
  if (/^[\w.-]+\/[\w./:*{}-]*$/.test(text)) return false;
  if (
    /\b(?:linear-gradient|radial-gradient|rgb|rgba|hsl|oklch|calc|var)\(/.test(
      text,
    ) ||
    /(?:^|\s)(?:items|justify|rounded|font|text|bg|border|gap|space|grid-cols|p[trblxy]?|m[trblxy]?)-[\w[.-]+/.test(
      text,
    )
  )
    return false;
  return true;
}

function entityText(text: string): string {
  return text.replace(
    /&(#x[\da-f]+|#\d+|amp|lt|gt|quot|apos|nbsp|rsquo|lsquo|rdquo|ldquo|mdash|ndash|hellip);/gi,
    (entity, code: string) => {
      if (code.startsWith("#")) {
        const value =
          code[1].toLowerCase() === "x"
            ? Number.parseInt(code.slice(2), 16)
            : Number(code.slice(1));
        return value <= 0x10ffff ? String.fromCodePoint(value) : entity;
      }
      return (
        (
          {
            amp: "&",
            lt: "<",
            gt: ">",
            quot: '"',
            apos: "'",
            nbsp: " ",
            rsquo: "’",
            lsquo: "‘",
            rdquo: "”",
            ldquo: "“",
            mdash: "—",
            ndash: "–",
            hellip: "…",
          } as Record<string, string>
        )[code] ?? entity
      );
    },
  );
}

function typescriptCopy(file: string, source: string): Occurrence[] {
  const tree = ts.createSourceFile(
    file,
    source,
    ts.ScriptTarget.Latest,
    true,
    file.endsWith(".svg") ? ts.ScriptKind.TSX : undefined,
  );
  const result: Occurrence[] = [];
  function add(node: ts.Node, text: string) {
    if (readable(text))
      result.push({
        text,
        file,
        line: tree.getLineAndCharacterOfPosition(node.getStart(tree)).line + 1,
      });
  }
  function visit(node: ts.Node): void {
    if (
      ts.isImportDeclaration(node) ||
      ts.isExportDeclaration(node) ||
      ts.isTypeNode(node)
    )
      return;
    if (ts.isJsxText(node)) {
      add(node, entityText(node.text));
      return;
    }
    if (
      ts.isStringLiteral(node) ||
      ts.isNoSubstitutionTemplateLiteral(node) ||
      ts.isTemplateExpression(node)
    ) {
      let visible = false;
      let excluded = false;
      let current: ts.Node = node;
      for (
        let parent = node.parent;
        parent;
        current = parent, parent = parent.parent
      ) {
        if (ts.isJsxAttribute(parent)) {
          const name = parent.name.getText(tree);
          visible = visibleNames.test(name);
          excluded = technicalNames.test(name);
          break;
        }
        if (ts.isPropertyAssignment(parent)) {
          if (parent.name === current) {
            excluded = true;
            break;
          }
          const name = parent.name.getText(tree).replace(/^["']|["']$/g, "");
          if (visibleNames.test(name) || name === "name") {
            visible = true;
            break;
          }
          if (
            technicalNames.test(name) &&
            !/^(type|role|mode|size|width|height|color|layout|action|value)$/.test(
              name,
            )
          ) {
            excluded = true;
            break;
          }
        }
        if (ts.isJsxExpression(parent) && !ts.isJsxAttribute(parent.parent)) {
          visible = true;
          break;
        }
        if (
          ts.isBinaryExpression(parent) &&
          [
            ts.SyntaxKind.EqualsEqualsEqualsToken,
            ts.SyntaxKind.ExclamationEqualsEqualsToken,
            ts.SyntaxKind.EqualsEqualsToken,
            ts.SyntaxKind.ExclamationEqualsToken,
          ].includes(parent.operatorToken.kind)
        ) {
          excluded = true;
          break;
        }
        if (ts.isCaseClause(parent) && parent.expression === current) {
          excluded = true;
          break;
        }
        if (
          ts.isElementAccessExpression(parent) &&
          parent.argumentExpression === current
        ) {
          excluded = true;
          break;
        }
        if (ts.isCallExpression(parent)) {
          const name = parent.expression.getText(tree);
          if (/^(?:api|ask)$/.test(name) && current === parent.arguments[0]) {
            excluded = true;
            break;
          }
          if (
            /^(?:console\..*|(?:.*\.)?(?:cn|cva|fetch|querySelector|querySelectorAll|closest|setAttribute|getAttribute|addEventListener|removeEventListener|matchMedia|getPropertyValue|setProperty|split|startsWith|endsWith|includes|indexOf|lastIndexOf|padStart|padEnd|toString|toLocaleString|toLocaleDateString|toLocaleTimeString|format))$/.test(
              name,
            )
          ) {
            excluded = true;
            break;
          }
        }
        if (ts.isStatement(parent) || ts.isFunctionLike(parent)) break;
      }
      const text = ts.isTemplateExpression(node)
        ? node.head.text +
          node.templateSpans
            .map(
              (span) =>
                `\u0024{${span.expression.getText(tree)}}${span.literal.text}`,
            )
            .join("")
        : node.text;
      if (
        ts.isExpressionStatement(node.parent) &&
        /^use (client|server|strict)$/.test(text)
      )
        return;
      if (!excluded && (visible || likelyCopy(text)))
        add(node, ts.isJsxAttribute(node.parent) ? entityText(text) : text);
      if (ts.isTemplateExpression(node)) {
        for (const span of node.templateSpans) visit(span.expression);
      }
      return;
    }
    ts.forEachChild(node, visit);
  }
  visit(tree);
  return result;
}

const groups: [string, RegExp][] = [
  ["Legal pages", /(?:\/legal\/|legal-documents)/],
  ["Staff console", /(?:\/staff\/|\/register\/|\/lib\/report\.)/],
  [
    "Blog administration",
    /(?:\/admin\/blog\/|\/blog\/admin\/|blog-admin|api\/internal\/integration\/blog\/|api\/internal\/blog\/writer)/,
  ],
  [
    "Writing posts",
    /(?:\/posts\/|\/blog\/writing\/|post-writing|post-body|post-unpublishing|post-standing|post-history|schedule-time|api\/internal\/blog\/(?:post|publish|revision|schedule|body\/))/,
  ],
  [
    "Reading the blog and link cards",
    /(?:\/blog\/|blog-|\/post-|article-|\/byline|link-card)/,
  ],
  [
    "Work editor",
    /(?:\/workspace\/|Editors?\.|\/PresetEditors|\/ThemeEditors|drafted-changes|page-arrangement|readiness|work-publish|preset-slots|theme-colors|lorebook-entry|markdown-edit|api\/internal\/block\/)/,
  ],
  [
    "Work pages and versions",
    /(?:\/a\/|\/work-|\/work\/|\/version\/|\/private\/|preserved|extension-|element-label|empty-page)/,
  ],
  [
    "Uploads and file formats",
    /(?:\/upload\/|\/format\/|\/summary\/|\/download\/|\/storage\/|import-stage|replacement-subject)/,
  ],
  [
    "Accounts and email",
    /(?:\/auth\/|\/account\/|\/(?:sign-in|sign-up|verify-email|forgot-password|reset-password)\/|\/settings\/page|account-access|nsfw-preference|\/auth\.)/,
  ],
  [
    "Profiles and your work",
    /(?:\/profile\/|\/\[profile\]\/|\/settings\/profile\/|profile-|\/collection\/)/,
  ],
  [
    "Connected apps",
    /(?:\/connect\/|connected-app|connection-request|permissions|install-track|installed-app)/,
  ],
  [
    "Integrations",
    /(?:\/updates\/|\/integration\/|\/integrations|\/settings\/integrations\/|attempt-standing|announcement)/,
  ],
  [
    "Notifications and following",
    /(?:\/notifications\/|\/notify\/|notification-inbox|work-follow|\/follow\/)/,
  ],
  [
    "Landing and browse",
    /(?:\/landing\/|\/browse\/|browse-|\/\(site\)\/page\.|work-types)/,
  ],
];

function groupFor(file: string): string {
  return (
    groups.find(([, pattern]) => pattern.test(file))?.[0] ??
    "Shared navigation, controls and errors"
  );
}

async function siteFiles(directory: string): Promise<string[]> {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = await Promise.all(
    entries.map(async (entry) => {
      const path = join(directory, entry.name);
      if (entry.isDirectory())
        return entry.name === "generated" ? [] : siteFiles(path);
      return /(?:\.[cm]?tsx?|\.svg|\.css)$/.test(entry.name) &&
        !/\.(?:test|spec|d)\.[cm]?tsx?$/.test(entry.name)
        ? [path]
        : [];
    }),
  );
  return files.flat().sort();
}

function code(text: string): string {
  const fence = "`".repeat(
    Math.max(
      0,
      ...Array.from(text.matchAll(/`+/g), (match) => match[0].length),
    ) + 1,
  );
  const padding = text.startsWith("`") || text.endsWith("`") ? " " : "";
  return `${fence}${padding}${text}${padding}${fence}`;
}

export async function inventory(root: string): Promise<string> {
  const occurrences: Occurrence[] = [];
  for (const path of [
    ...(await siteFiles(join(root, "web/src"))),
    ...(await siteFiles(join(root, "web/public"))),
  ]) {
    const file = relative(root, path);
    const source = await readFile(path, "utf8");
    if (path.endsWith(".css")) {
      for (const match of source.matchAll(/\bcontent\s*:\s*(["'])(.*?)\1/g)) {
        if (match[2].trim())
          occurrences.push({
            text: match[2],
            file,
            line: source.slice(0, match.index).split("\n").length,
          });
      }
    } else {
      occurrences.push(...typescriptCopy(file, source));
    }
  }
  const go = execFileSync("go", ["run", join(scripts, "copy-go.go"), root], {
    encoding: "utf8",
    maxBuffer: 16 * 1024 * 1024,
  });
  for (const line of go.trim().split("\n").filter(Boolean)) {
    const occurrence: Occurrence = JSON.parse(line);
    if (likelyCopy(occurrence.text)) occurrences.push(occurrence);
  }
  const unique = new Map<string, Occurrence[]>();
  for (const occurrence of occurrences) {
    const text = occurrence.text.replace(/\s+/g, " ").trim();
    if (!text) continue;
    const locations = unique.get(text) ?? [];
    if (
      !locations.some(
        (location) =>
          location.file === occurrence.file &&
          location.line === occurrence.line,
      )
    )
      locations.push(occurrence);
    unique.set(text, locations);
  }
  const byGroup = new Map<
    string,
    { text: string; locations: Occurrence[] }[]
  >();
  for (const [text, locations] of unique) {
    locations.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);
    const owners = new Set(
      locations.map((location) => groupFor(location.file)),
    );
    const group =
      owners.size === 1
        ? [...owners][0]
        : "Shared navigation, controls and errors";
    const entries = byGroup.get(group) ?? [];
    entries.push({ text, locations });
    byGroup.set(group, entries);
  }
  const sections: {
    title: string;
    group: string;
    entries: { text: string; locations: Occurrence[] }[];
  }[] = [];
  for (const [group, entries] of [...byGroup].sort(([a], [b]) =>
    a.localeCompare(b),
  )) {
    entries.sort(
      (a, b) =>
        a.locations[0].file.localeCompare(b.locations[0].file) ||
        a.locations[0].line - b.locations[0].line ||
        a.text.localeCompare(b.text),
    );
    for (let start = 0; start < entries.length; start += 160) {
      const part =
        entries.length > 160 ? `, part ${Math.floor(start / 160) + 1}` : "";
      sections.push({
        title: `Copy sweep: ${group.toLowerCase()}${part}`,
        group,
        entries: entries.slice(start, start + 160),
      });
    }
  }
  const lines = [
    "# Copy inventory",
    "",
    "Generated from source by `make copy-inventory`. Redirect its output to the local checklist. Regeneration replaces checkmarks; keep the reviewed copy before regenerating.",
    "",
    "One checkbox per distinct source string or template, with every occurrence listed. Whitespace is folded for reading. Interpolations and Go format verbs stand for runtime values; creator content and browser-generated dates are not copied from a database.",
    "",
    "Helper literals and API errors are kept conservatively, including errors an upload can expose through a wrapped failure. During each sweep, check callers before changing a short value that may also be a protocol value. Source paths are relative to the repository root. Shared wording is reviewed once in the shared group.",
    "",
    `${unique.size} strings across ${new Set(occurrences.map((item) => item.file)).size} source files. Each sweep holds at most 160 strings.`,
    "",
    "## Sweep tickets",
    "",
    "| Title | Page group | Strings |",
    "| --- | --- | ---: |",
    ...sections.map(
      (section) =>
        `| ${section.title} | ${section.group} | ${section.entries.length} |`,
    ),
    "",
  ];
  for (const section of sections) {
    lines.push(
      `## ${section.group}`,
      "",
      `Sweep ticket: **${section.title}**`,
      "",
    );
    let component = "";
    for (const entry of section.entries) {
      const file = entry.locations[0].file;
      if (file !== component) {
        lines.push("", `### ${file}`, "");
        component = file;
      }
      lines.push(
        `- [ ] ${code(entry.text)} — ${entry.locations.map((location) => `\u0060${location.file}:${location.line}\u0060`).join(", ")}`,
      );
    }
    lines.push("");
  }
  return `${lines.join("\n")}\n`;
}

if (import.meta.main)
  process.stdout.write(await inventory(join(scripts, "../..")));
