import {
  Check,
  CircleAlert,
  Info,
  Lightbulb,
  TriangleAlert,
} from "lucide-react";
import Image from "next/image";
import type { PostMedia } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import type {
  PostBlock,
  PostBody,
  PostCalloutKind,
  PostGalleryImage,
  PostImage,
  PostItem,
  PostRow,
  PostSpan,
  PostTask,
} from "@/lib/post-body";
import { PostCode } from "./PostCode";
import { PostSpans } from "./PostSpans";

const CALLOUTS: Record<
  PostCalloutKind,
  { label: string; mark: typeof Info; wash: string; kind: string }
> = {
  note: { label: "Note", mark: Info, wash: "bg-deep", kind: "text-ink" },
  tip: { label: "Tip", mark: Lightbulb, wash: "bg-deep", kind: "text-ink" },
  important: {
    label: "Important",
    mark: CircleAlert,
    wash: "bg-accent-wash",
    kind: "text-accent",
  },
  warning: {
    label: "Warning",
    mark: TriangleAlert,
    wash: "bg-stop-wash",
    kind: "text-stop",
  },
};

const HEADING = {
  2: "mt-12 mb-4 font-display text-title font-medium leading-tight",
  3: "mt-10 mb-3 font-display text-section font-medium leading-snug",
  4: "mt-8 mb-2 font-display text-ui font-semibold tracking-[0.02em] uppercase",
} as const;

const CAPTION = "mt-3 font-prose text-meta leading-6 text-mute text-pretty";

const PICTURE = "h-auto w-full rounded-plate bg-deep";

export function ArticleBody({
  body,
  media,
}: {
  body: PostBody;
  media: PostMedia[];
}) {
  return (
    <div className="flow-root font-prose text-article break-words text-ink [&>*+*]:mt-5">
      {body.content.map((block, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Blocks follow the writing and hold no local state.
        <Block block={block} key={index} media={media} />
      ))}
    </div>
  );
}

function Blocks({ blocks }: { blocks: PostBlock[] }) {
  return (
    <>
      {blocks.map((block, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Blocks follow the writing and hold no local state.
        <Block block={block} key={index} media={[]} />
      ))}
    </>
  );
}

function Block({ block, media }: { block: PostBlock; media: PostMedia[] }) {
  switch (block.type) {
    case "paragraph":
      return (
        <p className="text-pretty">
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
        <ul className="my-6 list-disc pl-6 marker:text-mute [&>li+li]:mt-3">
          <Items items={block.content} />
        </ul>
      );
    case "orderedList":
      return (
        <ol className="my-6 list-decimal pl-6 marker:text-mute [&>li+li]:mt-3">
          <Items items={block.content} />
        </ol>
      );
    case "taskList":
      return (
        <ul className="my-6 list-none [&>li+li]:mt-3">
          <Tasks tasks={block.content} />
        </ul>
      );
    case "quote":
      return (
        <blockquote className="my-10 font-display text-title leading-snug text-accent [&>*+*]:mt-4">
          <Blocks blocks={block.content} />
        </blockquote>
      );
    case "codeBlock":
      return <PostCode language={block.language} source={block.source} />;
    case "table":
      return <Table rows={block.content} />;
    case "callout":
      return <Callout blocks={block.content} kind={block.kind} />;
    case "image":
      return <Picture className="my-9" media={media} picture={block} />;
    case "gallery":
      return <Gallery media={media} pictures={block.content} />;
    case "divider":
      return (
        <p aria-hidden="true" className="my-10 text-center text-mute">
          <span className="inline-flex gap-2">
            <span className="size-1 rounded-full bg-current" />
            <span className="size-1 rounded-full bg-current" />
            <span className="size-1 rounded-full bg-current" />
          </span>
        </p>
      );
  }
}

function Picture({
  className,
  media,
  picture,
  thumb,
}: {
  className: string;
  media: PostMedia[];
  picture: PostImage | PostGalleryImage;
  thumb?: boolean;
}) {
  const held = media.find((one) => one.id === picture.mediaId);
  if (!held) return null;
  return (
    <figure className={className}>
      <Image
        alt={picture.alt}
        className={thumb ? cn(PICTURE, "aspect-[4/3] object-contain") : PICTURE}
        height={held.height}
        src={thumb ? held.thumbUrl : held.url}
        unoptimized
        width={held.width}
      />
      {picture.caption ? (
        <figcaption className={thumb ? cn(CAPTION, "mt-2") : CAPTION}>
          {picture.caption}
        </figcaption>
      ) : null}
    </figure>
  );
}

function Gallery({
  media,
  pictures,
}: {
  media: PostMedia[];
  pictures: PostGalleryImage[];
}) {
  return (
    <div
      className={cn(
        "my-9 grid w-full gap-3 sm:gap-4",
        pictures.length === 1 ? "grid-cols-1" : "grid-cols-2",
        pictures.length >= 3 ? "lg:grid-cols-3" : null,
      )}
    >
      {pictures.map((picture) => (
        <Picture
          className="min-w-0"
          key={picture.mediaId}
          media={media}
          picture={picture}
          thumb
        />
      ))}
    </div>
  );
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
  const step = Math.min(Math.max(level, 2), 4) as 2 | 3 | 4;
  const Tag = `h${step}` as "h2" | "h3" | "h4";
  return (
    <Tag
      className={cn(
        HEADING[step],
        "scroll-mt-[calc(var(--header-height)+1.5rem)] text-balance",
      )}
      id={anchor}
    >
      <PostSpans spans={spans} />
    </Tag>
  );
}

function Items({ items }: { items: PostItem[] }) {
  return (
    <>
      {items.map((item, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Items follow the writing and hold no local state.
        <li className="pl-1 [&>*+*]:mt-3" key={index}>
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
        <li
          className="group flex items-start gap-3"
          data-done={task.done}
          // biome-ignore lint/suspicious/noArrayIndexKey: Tasks follow the writing and hold no local state.
          key={index}
        >
          <span
            aria-hidden="true"
            className="mt-1.5 grid size-5 shrink-0 place-items-center rounded-[6px] bg-deep text-on-accent group-data-[done=true]:bg-action"
          >
            {task.done ? <Check className="size-3" strokeWidth={3} /> : null}
          </span>
          <span className="sr-only">{task.done ? "Done: " : "To do: "}</span>
          <div className="min-w-0 flex-1 group-data-[done=true]:text-mute [&>*+*]:mt-3">
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
  const {
    label,
    mark: Mark,
    wash,
    kind: tone,
  } = CALLOUTS[kind] ?? CALLOUTS.note;
  return (
    <div
      className={cn("my-8 rounded-plate p-5 sm:px-6", wash)}
      data-kind={kind}
      role="note"
    >
      <p
        className={cn(
          "flex items-center gap-2 text-meta font-semibold tracking-[0.02em]",
          tone,
        )}
      >
        <Mark aria-hidden="true" className="size-4" />
        {label}
      </p>
      <div className="mt-2 text-prose [&>*+*]:mt-3">
        <Blocks blocks={blocks} />
      </div>
    </div>
  );
}

function Table({ rows }: { rows: PostRow[] }) {
  const headingRow = rows[0]?.content.every((cell) => cell.heading) ?? false;
  return (
    <section
      aria-label="Table"
      className="my-8 overflow-x-auto rounded-plate"
      // biome-ignore lint/a11y/noNoninteractiveTabindex: A region that scrolls has to be reachable by keyboard.
      tabIndex={0}
    >
      <table className="w-full min-w-[32rem] border-collapse text-left text-meta">
        {headingRow ? (
          <thead className="bg-deep">
            <tr>
              <Cells row={rows[0]} scope="col" />
            </tr>
          </thead>
        ) : null}
        <tbody>
          {(headingRow ? rows.slice(1) : rows).map((row, index) => (
            <tr
              className="even:bg-deep/45"
              // biome-ignore lint/suspicious/noArrayIndexKey: Rows follow the table and hold no local state.
              key={index}
            >
              <Cells row={row} scope="row" />
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Cells({ row, scope }: { row: PostRow; scope: "col" | "row" }) {
  const shape = "min-w-[10ch] px-4 py-3 align-top [&>*+*]:mt-2";
  return (
    <>
      {row.content.map((cell, index) =>
        cell.heading ? (
          <th
            className={cn(shape, "font-medium text-ink")}
            // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the table and hold no local state.
            key={index}
            scope={scope}
          >
            <Blocks blocks={cell.content} />
          </th>
        ) : (
          <td
            className={shape}
            // biome-ignore lint/suspicious/noArrayIndexKey: Cells follow the table and hold no local state.
            key={index}
          >
            <Blocks blocks={cell.content} />
          </td>
        ),
      )}
    </>
  );
}
