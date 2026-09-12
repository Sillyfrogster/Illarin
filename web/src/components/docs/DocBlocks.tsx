import {
  CircleAlert,
  Info,
  Lightbulb,
  Link2,
  TriangleAlert,
} from "lucide-react";
import Link from "next/link";
import { cn } from "@/lib/cn";
import type { DocBlock, DocInline, DocTone } from "@/lib/docs/read-doc";
import { DocCode } from "./DocCode";

const TONES: Record<
  DocTone,
  { label: string; mark: typeof Info; wash: string; ink: string }
> = {
  note: { label: "Note", mark: Info, wash: "bg-deep", ink: "text-ink" },
  tip: { label: "Tip", mark: Lightbulb, wash: "bg-deep", ink: "text-ink" },
  important: {
    label: "Important",
    mark: CircleAlert,
    wash: "bg-accent-wash",
    ink: "text-accent",
  },
  warning: {
    label: "Warning",
    mark: TriangleAlert,
    wash: "bg-stop-wash",
    ink: "text-stop",
  },
};

const METHODS: Record<string, string> = {
  GET: "bg-accent-wash text-accent",
  POST: "bg-accent text-on-accent",
  PUT: "bg-accent text-on-accent",
  PATCH: "bg-accent text-on-accent",
  DELETE: "bg-stop-wash text-stop",
};

const HEADING = {
  2: "mt-12 mb-3 font-display text-[1.375rem] leading-snug font-semibold tracking-[-0.01em]",
  3: "mt-8 mb-2 font-display text-[1.125rem] leading-snug font-semibold",
} as const;

const PARAGRAPH = "my-3 font-prose text-prose leading-7 text-ink text-pretty";

const INLINE_CODE =
  "rounded-[5px] bg-deep px-[0.35em] py-[0.1em] font-mono text-[0.86em] text-ink";

const LIST =
  "my-3 pl-6 font-prose text-prose leading-7 text-ink marker:text-mute [&>li]:mt-1.5 [&>li]:pl-1 [&_p]:my-0 [&_ul]:my-1.5 [&_ol]:my-1.5";

export function DocBlocks({ blocks }: { blocks: DocBlock[] }) {
  return (
    <>
      {blocks.map((block, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Blocks follow the page and hold no local state.
        <Block block={block} key={index} />
      ))}
    </>
  );
}

function Block({ block }: { block: DocBlock }) {
  switch (block.kind) {
    case "paragraph":
      return (
        <p className={PARAGRAPH}>
          <Inlines nodes={block.children} />
        </p>
      );
    case "endpoint":
      return <Endpoint method={block.method} path={block.path} />;
    case "heading":
      return <Heading block={block} />;
    case "list": {
      const Tag = block.ordered ? "ol" : "ul";
      return (
        <Tag className={cn(LIST, block.ordered ? "list-decimal" : "list-disc")}>
          {block.items.map((item, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: Items follow the list and hold no local state.
            <li key={index}>
              <DocBlocks blocks={item} />
            </li>
          ))}
        </Tag>
      );
    }
    case "code":
      return (
        <DocCode
          label={block.label}
          language={block.language}
          source={block.source}
        />
      );
    case "table":
      return <Table head={block.head} rows={block.rows} />;
    case "callout":
      return <Callout blocks={block.children} tone={block.tone} />;
    case "quote":
      return (
        <blockquote className="my-4 border-l border-rule pl-4 [&_p]:text-mute">
          <DocBlocks blocks={block.children} />
        </blockquote>
      );
    case "divider":
      return <hr className="my-8 h-px border-0 bg-rule" />;
  }
}

function Endpoint({ method, path }: { method: string; path: string }) {
  return (
    <p className="my-3 flex flex-wrap items-center gap-2 rounded-control bg-deep px-3 py-2 font-mono text-meta">
      <span
        className={cn(
          "rounded-[5px] px-1.5 py-0.5 text-label font-semibold tracking-[0.04em]",
          METHODS[method] ?? METHODS.GET,
        )}
      >
        {method}
      </span>
      <span className="text-ink [overflow-wrap:anywhere]">{path}</span>
    </p>
  );
}

function Heading({ block }: { block: Extract<DocBlock, { kind: "heading" }> }) {
  const Tag = block.level === 2 ? "h2" : "h3";
  return (
    <Tag
      className={cn(
        "group scroll-mt-[calc(var(--header-height)+1.5rem)]",
        HEADING[block.level],
      )}
      id={block.anchor}
    >
      <a
        className="inline-flex items-baseline gap-2 text-ink no-underline outline-offset-4"
        href={`#${block.anchor}`}
      >
        <Inlines nodes={block.children} />
        <Link2
          aria-hidden="true"
          className="size-4 shrink-0 self-center text-accent opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100 motion-reduce:transition-none"
        />
        <span className="sr-only">, link to this section</span>
      </a>
    </Tag>
  );
}

function Table({ head, rows }: { head: DocInline[][]; rows: DocInline[][][] }) {
  const cell = "px-3 py-2.5 align-top font-prose text-meta leading-6";
  return (
    <section
      aria-label="Table"
      className="my-4 overflow-x-auto"
      // biome-ignore lint/a11y/noNoninteractiveTabindex: A region that scrolls has to be reachable by keyboard.
      tabIndex={0}
    >
      <table className="w-full min-w-[28rem] border-collapse text-left">
        <thead>
          <tr className="border-b border-rule">
            {head.map((nodes, index) => (
              <th
                className={cn(cell, "font-ui font-semibold text-ink")}
                // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the table and hold no local state.
                key={index}
                scope="col"
              >
                <Inlines nodes={nodes} />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr
              className="border-b border-rule/60"
              // biome-ignore lint/suspicious/noArrayIndexKey: Rows follow the table and hold no local state.
              key={index}
            >
              {row.map((nodes, at) => (
                <td
                  className={cn(cell, at === 0 ? "text-ink" : "text-mute")}
                  // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the table and hold no local state.
                  key={at}
                >
                  <Inlines nodes={nodes} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Callout({ blocks, tone }: { blocks: DocBlock[]; tone: DocTone }) {
  const { label, mark: Mark, wash, ink } = TONES[tone];
  return (
    <div className={cn("my-4 rounded-plate px-5 py-4", wash)} role="note">
      <p
        className={cn(
          "flex items-center gap-2 text-meta font-semibold tracking-[0.02em]",
          ink,
        )}
      >
        <Mark aria-hidden="true" className="size-4" />
        {label}
      </p>
      <div className="[&>p]:my-0 [&>p+p]:mt-2 [&>p]:text-meta [&>p]:leading-6">
        <DocBlocks blocks={blocks} />
      </div>
    </div>
  );
}

function Inlines({ nodes }: { nodes: DocInline[] }) {
  return (
    <>
      {nodes.map((node, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the text and hold no local state.
        <Inline key={index} node={node} />
      ))}
    </>
  );
}

function Inline({ node }: { node: DocInline }) {
  switch (node.kind) {
    case "text":
      return node.text;
    case "code":
      return <code className={INLINE_CODE}>{node.text}</code>;
    case "strong":
      return (
        <strong className="font-semibold text-ink">
          <Inlines nodes={node.children} />
        </strong>
      );
    case "emphasis":
      return (
        <em>
          <Inlines nodes={node.children} />
        </em>
      );
    case "link":
      return <DocLink href={node.href} nodes={node.children} />;
  }
}

function DocLink({ href, nodes }: { href: string; nodes: DocInline[] }) {
  const classes =
    "text-accent underline decoration-accent/40 underline-offset-[3px] hover:decoration-accent";
  if (href.startsWith("/") || href.startsWith("#")) {
    return (
      <Link className={classes} href={href}>
        <Inlines nodes={nodes} />
      </Link>
    );
  }
  return (
    <a className={classes} href={href} rel="noreferrer noopener">
      <Inlines nodes={nodes} />
    </a>
  );
}
