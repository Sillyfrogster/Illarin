import type {
  PostBlock,
  PostDocument,
  PostItem,
  PostSpan,
} from "@/lib/post-document";
import styles from "./PostBody.module.css";

export function PostBody({ document }: { document: PostDocument }) {
  return (
    <div className={styles.body}>
      <Blocks blocks={document.content} />
    </div>
  );
}

function Blocks({ blocks }: { blocks: PostBlock[] }) {
  return (
    <>
      {blocks.map((block, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Blocks follow the writing and hold no local state.
        <Block block={block} key={index} />
      ))}
    </>
  );
}

function Block({ block }: { block: PostBlock }) {
  switch (block.type) {
    case "paragraph":
      return (
        <p className={styles.paragraph}>
          <Spans spans={block.content} />
        </p>
      );
    case "heading":
      return <Heading level={block.level} spans={block.content} />;
    case "bulletList":
      return (
        <ul className={styles.list}>
          <Items items={block.content} />
        </ul>
      );
    case "orderedList":
      return (
        <ol className={`${styles.list} ${styles.ordered}`}>
          <Items items={block.content} />
        </ol>
      );
    case "quote":
      return (
        <blockquote className={styles.quote}>
          <Blocks blocks={block.content} />
        </blockquote>
      );
    case "divider":
      return <hr className={styles.divider} />;
  }
}

function Heading({ level, spans }: { level: number; spans: PostSpan[] }) {
  const Tag = `h${Math.min(Math.max(level, 2), 4)}` as "h2" | "h3" | "h4";
  return (
    <Tag className={styles.heading}>
      <Spans spans={spans} />
    </Tag>
  );
}

function Items({ items }: { items: PostItem[] }) {
  return (
    <>
      {items.map((item, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Items follow the writing and hold no local state.
        <li className={styles.item} key={index}>
          <Blocks blocks={item.content} />
        </li>
      ))}
    </>
  );
}

function Spans({ spans }: { spans: PostSpan[] }) {
  return (
    <>
      {spans.map((span, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the writing and hold no local state.
        <Span key={index} span={span} />
      ))}
    </>
  );
}

function Span({ span }: { span: PostSpan }) {
  let rendered = <>{span.text}</>;
  for (const mark of span.marks ?? []) {
    switch (mark.type) {
      case "code":
        rendered = <code className={styles.code}>{rendered}</code>;
        break;
      case "italic":
        rendered = <em>{rendered}</em>;
        break;
      case "bold":
        rendered = <strong>{rendered}</strong>;
        break;
      case "link":
        rendered = (
          <a
            className={styles.link}
            href={mark.href}
            rel="noreferrer nofollow"
            target={mark.href.startsWith("https://") ? "_blank" : undefined}
          >
            {rendered}
          </a>
        );
        break;
    }
  }
  return rendered;
}
