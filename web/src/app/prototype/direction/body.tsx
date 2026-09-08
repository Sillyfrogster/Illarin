"use client";

import Image from "next/image";
import type { Body as BodyBlock } from "./data";
import { Arrive } from "./motion";
import { cn, type Direction, Inline, Label } from "./ui";

const HEADING: Record<Direction, string> = {
  vitrine:
    "vd:mt-section vd:mb-flow vd:font-display vd:text-title vd:font-medium",
  ambient:
    "vd:mt-section vd:mb-flow vd:font-display vd:text-title vd:font-medium",
  ledger:
    "vd:mt-section vd:mb-flow vd:font-display vd:text-section vd:font-medium",
};

const PROSE: Record<Direction, string> = {
  vitrine: "vd:font-prose vd:text-[1.1875rem] vd:leading-[1.82] vd:text-ink/92",
  ambient: "vd:font-prose vd:text-[1.1875rem] vd:leading-[1.8] vd:text-ink/92",
  ledger: "vd:font-prose vd:text-[1.125rem] vd:leading-[1.78] vd:text-ink/92",
};

function Figure({
  block,
  direction,
  index,
}: {
  block: Extract<BodyBlock, { kind: "figure" }>;
  direction: Direction;
  index: number;
}) {
  if (direction === "ambient") {
    return (
      <Arrive className="v-full vd:my-section">
        <figure className="vd:relative vd:mx-auto vd:max-w-[76rem] vd:overflow-hidden vd:rounded-plate">
          <Image
            src={block.image}
            alt={block.alt}
            sizes="(max-width: 76rem) 100vw, 76rem"
            className="vd:h-auto vd:w-full"
          />
          <figcaption className="v-scrim vd:absolute vd:inset-x-0 vd:bottom-0 vd:px-6 vd:pt-14 vd:pb-6 vd:sm:px-10">
            <p className="v-caption vd:text-over vd:text-meta vd:leading-6">
              {block.caption}
            </p>
          </figcaption>
        </figure>
      </Arrive>
    );
  }
  if (direction === "ledger") {
    return (
      <Arrive className="v-wide vd:my-group">
        <figure className="vd:grid vd:gap-4 vd:sm:grid-cols-[1fr_13rem]">
          <div className="vd:overflow-hidden vd:rounded-plate vd:shadow-[inset_0_0_0_1px_var(--v-rule)]">
            <Image
              src={block.image}
              alt={block.alt}
              sizes="(max-width: 40rem) 100vw, 44rem"
              className="vd:h-auto vd:w-full"
            />
          </div>
          <figcaption className="vd:pt-1">
            <Label style={{ color: "var(--v-accent)" }}>
              Plate {String(index).padStart(2, "0")}
            </Label>
            <p className="vd:mt-2 vd:text-meta vd:leading-6 vd:text-mute">
              {block.caption}
            </p>
          </figcaption>
        </figure>
      </Arrive>
    );
  }
  return (
    <Arrive className="v-wide vd:my-section">
      <figure>
        <div className="vd:relative vd:overflow-hidden vd:rounded-plate">
          <Image
            src={block.image}
            alt={block.alt}
            sizes="(max-width: 60rem) 100vw, 60rem"
            className="vd:h-auto vd:w-full"
          />
        </div>
        <figcaption className="vd:mt-4 vd:flex vd:gap-4">
          <span
            aria-hidden="true"
            className="v-accent-rule vd:mt-3 vd:h-px vd:w-10 vd:shrink-0"
          />
          <p className="v-caption vd:text-meta vd:leading-6 vd:text-mute">
            {block.caption}
          </p>
        </figcaption>
      </figure>
    </Arrive>
  );
}

function Quote({
  block,
  direction,
}: {
  block: Extract<BodyBlock, { kind: "quote" }>;
  direction: Direction;
}) {
  if (direction === "ledger") {
    return (
      <blockquote className="vd:my-group vd:pl-6 vd:shadow-[inset_2px_0_0_var(--v-accent)]">
        <p className="vd:font-prose vd:text-[1.1875rem] vd:leading-8 vd:italic">
          {block.text}
        </p>
      </blockquote>
    );
  }
  return (
    <blockquote
      className={cn(
        "vd:my-section",
        direction === "vitrine" ? "v-wide" : "vd:mx-auto",
      )}
    >
      <p
        className={cn(
          "vd:font-display vd:text-[clamp(1.5rem,2.6vw,2.125rem)] vd:leading-[1.28] vd:font-medium vd:tracking-[-0.01em]",
          direction === "ambient" && "vd:text-[var(--v-tint-ink)]",
        )}
      >
        {block.text}
      </p>
      <span
        aria-hidden="true"
        className="v-accent-rule vd:mt-6 vd:block vd:w-24"
      />
    </blockquote>
  );
}

function Table({ block }: { block: Extract<BodyBlock, { kind: "table" }> }) {
  return (
    <div className="v-wide vd:my-group vd:overflow-x-auto">
      <table className="v-tabular vd:w-full vd:min-w-[34rem] vd:border-collapse vd:text-left vd:text-meta">
        <thead>
          <tr>
            {block.head.map((cell) => (
              <th
                key={cell}
                scope="col"
                className="vd:pb-3 vd:pr-6 vd:text-label vd:font-bold vd:uppercase vd:text-mute vd:shadow-[inset_0_-1px_0_var(--v-rule)]"
              >
                {cell}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {block.rows.map((row) => (
            <tr key={row[0]}>
              {row.map((cell, column) => (
                <td
                  key={cell + String(column)}
                  className={cn(
                    "vd:py-3 vd:pr-6 vd:align-top vd:shadow-[inset_0_-1px_0_var(--v-hair)]",
                    column === 0
                      ? "vd:font-semibold vd:text-ink"
                      : "vd:text-mute",
                  )}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function Body({
  blocks,
  direction,
}: {
  blocks: BodyBlock[];
  direction: Direction;
}) {
  let figures = 0;
  let headings = 0;
  return (
    <>
      {blocks.map((block, index) => {
        const key = `${block.kind}-${index}`;
        switch (block.kind) {
          case "h": {
            headings += 1;
            const number = headings;
            return (
              <h2
                key={key}
                id={`section-${number}`}
                className={cn(HEADING[direction], "vd:scroll-mt-28")}
              >
                {direction === "ledger" && (
                  <span
                    className="v-tabular vd:mr-3 vd:text-title vd:font-normal"
                    style={{ color: "var(--v-accent)" }}
                  >
                    {String(number).padStart(2, "0")}
                  </span>
                )}
                {block.text}
              </h2>
            );
          }
          case "p":
            return (
              <p key={key} className={cn(PROSE[direction], "vd:mt-flow")}>
                <Inline text={block.text} />
              </p>
            );
          case "list":
            return (
              <ul
                key={key}
                className={cn(PROSE[direction], "vd:mt-flow vd:space-y-3")}
              >
                {block.items.map((item) => (
                  <li key={item} className="vd:flex vd:gap-4">
                    <span
                      aria-hidden="true"
                      className="vd:mt-[0.85em] vd:h-px vd:w-5 vd:shrink-0 vd:bg-accent"
                    />
                    <span>
                      <Inline text={item} />
                    </span>
                  </li>
                ))}
              </ul>
            );
          case "quote":
            return <Quote key={key} block={block} direction={direction} />;
          case "figure": {
            figures += 1;
            return (
              <Figure
                key={key}
                block={block}
                direction={direction}
                index={figures}
              />
            );
          }
          case "table":
            return <Table key={key} block={block} />;
          case "code":
            return (
              <pre
                key={key}
                className="v-wide vd:my-group vd:overflow-x-auto vd:rounded-plate vd:bg-ink/5 vd:p-5 vd:text-meta vd:leading-7 vd:shadow-[inset_0_0_0_1px_var(--v-hair)]"
              >
                <code className="vd:font-mono">{block.lines.join("\n")}</code>
              </pre>
            );
          default:
            return null;
        }
      })}
    </>
  );
}
