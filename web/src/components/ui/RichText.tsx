import { cn } from "@/lib/cn";
import { type RichBlock, type RichInline, readRichText } from "@/lib/rich-text";

export function RichText({
  text,
  className,
}: {
  text: string;
  className?: string;
}) {
  const { blocks } = readRichText(text);
  if (blocks.length === 0) return null;
  return (
    <div
      className={cn(
        "[&>*+*]:mt-[0.85em] [&>*+:is(h4,h5,h6)]:mt-[1.5em]",
        className,
      )}
    >
      <Blocks blocks={blocks} />
    </div>
  );
}

export function FormattingNotice() {
  return (
    <small className="block text-meta opacity-70">
      Unsupported formatting is hidden here. The original text is preserved in
      downloads.
    </small>
  );
}

function Blocks({ blocks }: { blocks: RichBlock[] }) {
  return (
    <>
      {blocks.map((block, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Blocks follow the writing and hold no local state.
        <Block block={block} key={index} />
      ))}
    </>
  );
}

const LIST =
  "list-disc pl-[1.35em] [&_li+li]:mt-[0.35em] [&_ul]:mt-[0.35em] [&_ul]:list-[circle] [&_ol]:mt-[0.35em]";

function Block({ block }: { block: RichBlock }) {
  if (block.kind === "paragraph") {
    return (
      <p>
        <Inline nodes={block.children} />
      </p>
    );
  }

  if (block.kind === "heading") {
    const Tag = `h${Math.min(block.depth + 3, 6)}` as "h4" | "h5" | "h6";
    return (
      <Tag
        className={cn(
          "font-display font-semibold text-ink leading-tight",
          Tag === "h4" ? "text-[1.1em]" : "text-[1em]",
        )}
      >
        <Inline nodes={block.children} />
      </Tag>
    );
  }

  if (block.kind === "code") {
    return (
      <pre className="overflow-x-auto rounded-control bg-deep px-4 py-3 font-mono text-[0.86em] leading-relaxed text-ink/90">
        {block.text}
      </pre>
    );
  }

  if (block.kind === "table") {
    return <Table block={block} />;
  }

  if (block.kind === "quote") {
    return (
      <blockquote className="border-rule border-l-2 pl-[1em] text-mute italic">
        <Blocks blocks={block.children} />
      </blockquote>
    );
  }

  if (block.ordered) {
    return (
      <ol className={cn(LIST, "list-decimal [&_ol]:list-[lower-alpha]")}>
        {block.items.map((item, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: Items follow the writing and hold no local state.
          <li key={index}>
            <Blocks blocks={item} />
          </li>
        ))}
      </ol>
    );
  }

  return (
    <ul className={LIST}>
      {block.items.map((item, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Items follow the writing and hold no local state.
        <li key={index}>
          <Blocks blocks={item} />
        </li>
      ))}
    </ul>
  );
}

const CELL = "px-3 py-2 align-top first:pl-0 last:pr-0";

function Table({ block }: { block: Extract<RichBlock, { kind: "table" }> }) {
  return (
    <div className="max-w-full overflow-x-auto">
      <table className="w-full border-collapse text-left">
        {block.head ? (
          <thead>
            <tr className="border-rule border-b">
              {block.head.map((cell, index) => (
                <th
                  className={cn(CELL, "font-semibold text-ink")}
                  // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the writing and hold no local state.
                  key={index}
                  scope="col"
                >
                  <Inline nodes={cell} />
                </th>
              ))}
            </tr>
          </thead>
        ) : null}
        <tbody>
          {block.rows.map((row, index) => (
            <tr
              className="border-rule/60 border-b last:border-b-0"
              // biome-ignore lint/suspicious/noArrayIndexKey: Rows follow the writing and hold no local state.
              key={index}
            >
              {row.map((cell, at) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the writing and hold no local state.
                <td className={CELL} key={at}>
                  <Inline nodes={cell} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Inline({ nodes }: { nodes: RichInline[] }) {
  return (
    <>
      {nodes.map((node, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the writing and hold no local state.
        <InlineNode node={node} key={index} />
      ))}
    </>
  );
}

function InlineNode({ node }: { node: RichInline }) {
  switch (node.kind) {
    case "text":
      return <>{node.text}</>;
    case "break":
      return <br />;
    case "code":
      return (
        <code className="rounded-control bg-deep px-[0.35em] py-[0.1em] font-mono text-[0.86em] [overflow-wrap:anywhere]">
          {node.text}
        </code>
      );
    case "emphasis":
      return (
        <em>
          <Inline nodes={node.children} />
        </em>
      );
    case "strong":
      return (
        <strong>
          <Inline nodes={node.children} />
        </strong>
      );
    default: {
      const away = !node.href.startsWith("/");
      return (
        <a
          className="text-ink underline decoration-accent/55 underline-offset-[3px] [overflow-wrap:anywhere] hover:decoration-accent"
          href={node.href}
          rel="noreferrer nofollow"
          target={away ? "_blank" : undefined}
        >
          <Inline nodes={node.children} />
        </a>
      );
    }
  }
}
