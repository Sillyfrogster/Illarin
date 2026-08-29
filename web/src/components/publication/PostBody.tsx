import {
  Check,
  CircleAlert,
  Info,
  Lightbulb,
  TriangleAlert,
} from "lucide-react";
import type {
  PostBlock,
  PostCalloutKind,
  PostDocument,
  PostItem,
  PostRow,
  PostSpan,
  PostTask,
} from "@/lib/post-document";
import styles from "./PostBody.module.css";
import { PostCode } from "./PostCode";
import { PostSpans } from "./PostSpans";

/** The mark and the word each kind of callout wears. */
const CALLOUTS: Record<PostCalloutKind, { label: string; mark: typeof Info }> =
  {
    note: { label: "Note", mark: Info },
    tip: { label: "Tip", mark: Lightbulb },
    important: { label: "Important", mark: CircleAlert },
    warning: { label: "Warning", mark: TriangleAlert },
  };

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
          <PostSpans spans={block.content} />
        </p>
      );
    case "heading":
      return (
        <Heading
          anchor={block.anchor}
          level={block.level}
          spans={block.content}
        />
      );
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
    case "taskList":
      return (
        <ul className={styles.tasks}>
          <Tasks tasks={block.content} />
        </ul>
      );
    case "quote":
      return (
        <blockquote className={styles.quote}>
          <Blocks blocks={block.content} />
        </blockquote>
      );
    case "codeBlock":
      return <PostCode language={block.language} source={block.source} />;
    case "table":
      return <Table rows={block.content} />;
    case "callout":
      return <Callout blocks={block.content} kind={block.kind} />;
    case "divider":
      return <hr className={styles.divider} />;
  }
}

function Heading({
  anchor,
  level,
  spans,
}: {
  anchor?: string;
  level: number;
  spans: PostSpan[];
}) {
  const Tag = `h${Math.min(Math.max(level, 2), 4)}` as "h2" | "h3" | "h4";
  return (
    <Tag className={styles.heading} id={anchor}>
      <PostSpans spans={spans} />
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

function Tasks({ tasks }: { tasks: PostTask[] }) {
  return (
    <>
      {tasks.map((task, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Tasks follow the writing and hold no local state.
        <li className={styles.task} data-done={task.done} key={index}>
          <span aria-hidden="true" className={styles.tick}>
            {task.done ? <Check size={13} strokeWidth={3} /> : null}
          </span>
          <span className={styles.aside}>
            {task.done ? "Done: " : "To do: "}
          </span>
          <div className={styles.taskSaid}>
            <Blocks blocks={task.content} />
          </div>
        </li>
      ))}
    </>
  );
}

function Callout({
  blocks,
  kind,
}: {
  blocks: PostBlock[];
  kind: PostCalloutKind;
}) {
  const { label, mark: Mark } = CALLOUTS[kind] ?? CALLOUTS.note;
  return (
    <aside className={styles.callout} data-kind={kind}>
      <p className={styles.calloutKind}>
        <Mark aria-hidden="true" size={17} strokeWidth={1.8} />
        {label}
      </p>
      <div className={styles.calloutSaid}>
        <Blocks blocks={blocks} />
      </div>
    </aside>
  );
}

function Table({ rows }: { rows: PostRow[] }) {
  const headingRow = rows[0]?.content.every((cell) => cell.heading) ?? false;
  return (
    <section
      aria-label="Table"
      className={styles.tableScroll}
      // biome-ignore lint/a11y/noNoninteractiveTabindex: A region that scrolls has to be reachable by keyboard.
      tabIndex={0}
    >
      <table className={styles.table}>
        {headingRow ? (
          <thead>
            <Line row={rows[0]} scope="col" />
          </thead>
        ) : null}
        <tbody>
          {(headingRow ? rows.slice(1) : rows).map((row, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: Rows follow the table and hold no local state.
            <Line key={index} row={row} scope="row" />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Line({ row, scope }: { row: PostRow; scope: "col" | "row" }) {
  return (
    <tr>
      {row.content.map((cell, index) =>
        cell.heading ? (
          // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the table and hold no local state.
          <th className={styles.cell} key={index} scope={scope}>
            <Blocks blocks={cell.content} />
          </th>
        ) : (
          // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the table and hold no local state.
          <td className={styles.cell} key={index}>
            <Blocks blocks={cell.content} />
          </td>
        ),
      )}
    </tr>
  );
}
