import type { PhrasingContent, RootContent } from "mdast";
import { fromMarkdown } from "mdast-util-from-markdown";
import { gfmFromMarkdown } from "mdast-util-gfm";
import { gfm } from "micromark-extension-gfm";
import type { ReactNode } from "react";
import { PostCode } from "@/components/blog/PostCode";
import { LANGUAGE_LABELS } from "@/lib/code-highlight";
import { headingId } from "@/lib/docs";
import type { PostLanguage } from "@/lib/post-body";

function words(nodes: readonly PhrasingContent[]): string {
  return nodes
    .map((node) =>
      "value" in node
        ? node.value
        : "children" in node
          ? words(node.children as PhrasingContent[])
          : "",
    )
    .join("");
}

function inline(nodes: readonly PhrasingContent[]): ReactNode {
  return nodes.map((node, index) => {
    const key = index;
    switch (node.type) {
      case "text":
        return node.value;
      case "inlineCode":
        return (
          <code
            className="rounded bg-inset px-1.5 py-0.5 font-mono text-[0.88em] text-ink"
            key={key}
          >
            {node.value}
          </code>
        );
      case "emphasis":
        return <em key={key}>{inline(node.children)}</em>;
      case "strong":
        return (
          <strong className="font-semibold text-ink" key={key}>
            {inline(node.children)}
          </strong>
        );
      case "link":
        return (
          <a
            className="text-accent underline underline-offset-4 hover:text-ink"
            href={node.url}
            key={key}
          >
            {inline(node.children)}
          </a>
        );
      case "break":
        return <br key={key} />;
      default:
        return null;
    }
  });
}

function blocks(nodes: readonly RootContent[]): ReactNode {
  return nodes.map((node, index) => {
    const key = index;
    switch (node.type) {
      case "heading": {
        const title = words(node.children);
        const id = headingId(title);
        const className =
          node.depth === 1
            ? "text-[clamp(2rem,4vw,3.25rem)] leading-tight tracking-[-0.045em]"
            : node.depth === 2
              ? "mt-14 text-title tracking-[-0.025em]"
              : "mt-8 text-section";
        const content = (
          <a
            className="group inline-flex items-baseline gap-2 text-ink hover:text-accent"
            href={`#${id}`}
          >
            {inline(node.children)}
            <span
              aria-hidden="true"
              className="text-meta opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100"
            >
              #
            </span>
          </a>
        );
        if (node.depth === 1)
          return (
            <h1
              className={`font-display font-medium text-ink ${className}`}
              id={id}
              key={key}
            >
              {content}
            </h1>
          );
        if (node.depth === 2)
          return (
            <h2
              className={`scroll-mt-28 font-display font-medium text-ink ${className}`}
              id={id}
              key={key}
            >
              {content}
            </h2>
          );
        return (
          <h3
            className={`scroll-mt-28 font-ui font-semibold text-ink ${className}`}
            id={id}
            key={key}
          >
            {content}
          </h3>
        );
      }
      case "paragraph":
        return (
          <p className="mt-5 text-pretty leading-7 text-ink" key={key}>
            {inline(node.children)}
          </p>
        );
      case "list": {
        const Tag = node.ordered ? "ol" : "ul";
        return (
          <Tag
            className={`${node.ordered ? "list-decimal" : "list-disc"} mt-5 space-y-2 pl-6 leading-7 marker:text-mute`}
            key={key}
            start={node.ordered ? (node.start ?? 1) : undefined}
          >
            {node.children.map((item) => (
              <li className="pl-1" key={item.position?.start.offset}>
                {blocks(item.children)}
              </li>
            ))}
          </Tag>
        );
      }
      case "code":
        return (
          <PostCode
            key={key}
            language={
              node.lang && node.lang in LANGUAGE_LABELS
                ? (node.lang as PostLanguage)
                : "plain"
            }
            source={node.value}
          />
        );
      case "table":
        return (
          <div
            className="mt-6 overflow-x-auto rounded-plate ring-1 ring-rule"
            key={key}
          >
            <table className="w-full min-w-[34rem] border-collapse text-left text-[0.9rem]">
              <thead className="bg-inset">
                <tr>
                  {node.children[0]?.children.map((cell) => (
                    <th
                      className="p-3 font-medium text-ink"
                      key={cell.position?.start.offset}
                    >
                      {inline(cell.children)}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {node.children.slice(1).map((row) => (
                  <tr
                    className="border-t border-rule"
                    key={row.position?.start.offset}
                  >
                    {row.children.map((cell) => (
                      <td
                        className="p-3 align-top leading-6"
                        key={cell.position?.start.offset}
                      >
                        {inline(cell.children)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );
      case "blockquote":
        return (
          <blockquote
            className="mt-6 border-l-2 border-accent pl-5 text-mute"
            key={key}
          >
            {blocks(node.children)}
          </blockquote>
        );
      case "thematicBreak":
        return <hr className="my-10 border-rule" key={key} />;
      default:
        return null;
    }
  });
}

export function DocMarkdown({ source }: { source: string }) {
  const document = fromMarkdown(source, {
    extensions: [gfm()],
    mdastExtensions: [gfmFromMarkdown()],
  });
  return (
    <div className="min-w-0 font-prose text-ui text-mute [&_li>p]:mt-0">
      {blocks(document.children)}
    </div>
  );
}

export function docSections(source: string) {
  const document = fromMarkdown(source);
  return document.children.flatMap((node) =>
    node.type === "heading" && node.depth === 2
      ? [{ title: words(node.children), id: headingId(words(node.children)) }]
      : [],
  );
}
