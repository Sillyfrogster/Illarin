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
      The page shows the words, not the formatting written into this text. The
      download is unchanged.
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
    /** Page headings already occupy h1 through h3. */
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
