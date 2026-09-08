"use client";

import { ArrowUpRight, Check, Copy } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import type { Element } from "./assets";
import { KIND_LABEL } from "./assets";
import { cn, type Direction, Label } from "./ui";

const PROSE: Record<Direction, string> = {
  vitrine: "vd:text-[1rem] vd:leading-[1.82]",
  ambient: "vd:text-[1rem] vd:leading-[1.8]",
  ledger: "vd:text-[1rem] vd:leading-[1.78]",
};

const ENTRY: Record<Direction, string> = {
  vitrine: "vd:text-[1rem] vd:leading-[1.8]",
  ambient: "vd:text-[1rem] vd:leading-[1.8]",
  ledger: "vd:text-[1.0625rem] vd:leading-[1.76]",
};

/** Verbatim text belongs to whoever wrote it, so it is shown exactly and offered for copying */
function Verbatim({ body }: { body: string }) {
  const [taken, setTaken] = useState(false);
  return (
    <div className="vd:relative vd:mt-3">
      <pre className="vd:overflow-x-auto vd:rounded-plate vd:bg-ink/4 vd:py-4 vd:pr-14 vd:pl-4 vd:bg-deep">
        <code className="vd:font-mono vd:text-[0.875rem] vd:leading-7 vd:whitespace-pre-wrap vd:text-ink/90">
          {body}
        </code>
      </pre>
      <button
        type="button"
        aria-label={taken ? "Copied" : "Copy"}
        onClick={() => {
          navigator.clipboard?.writeText(body);
          setTaken(true);
          setTimeout(() => setTaken(false), 1400);
        }}
        className="vd:absolute vd:top-2 vd:right-2 vd:inline-flex vd:size-11 vd:items-center vd:justify-center vd:rounded-control vd:text-mute vd:hover:bg-ink/8 vd:hover:text-ink"
      >
        {taken ? (
          <Check className="vd:size-4" />
        ) : (
          <Copy className="vd:size-4" />
        )}
      </button>
    </div>
  );
}

/** A short piece of state next to a name, used wherever an item can be switched off */
function Flag({ children, off = false }: { children: string; off?: boolean }) {
  return (
    <span
      className={cn(
        "vd:inline-flex vd:shrink-0 vd:items-center vd:rounded-control vd:px-2 vd:py-0.5 vd:text-label vd:font-bold ",
        off ? "vd:text-faint vd:bg-deep" : "vd:text-ink/70 vd:bg-deep",
      )}
    >
      {children}
    </span>
  );
}

function Keys({ keys, weak = false }: { keys: string[]; weak?: boolean }) {
  return (
    <span className="vd:flex vd:flex-wrap vd:gap-1.5">
      {keys.map((key) => (
        <span
          key={key}
          className={cn(
            "vd:rounded-control vd:px-2 vd:py-0.5 vd:font-mono vd:text-[0.75rem]",
            weak ? "vd:text-faint vd:bg-deep" : "vd:text-mute vd:bg-deep",
          )}
        >
          {key}
        </span>
      ))}
    </span>
  );
}

/** The name a list item is filed under, sharing one shape across every list type */
function ItemHead({
  index,
  name,
  direction,
  numbered,
  trailing,
}: {
  index: number;
  name: string;
  direction: Direction;
  numbered: boolean;
  trailing?: React.ReactNode;
}) {
  return (
    <div className="vd:flex vd:items-baseline vd:gap-3">
      <span
        className={cn(
          "vd:shrink-0 vd:text-meta vd:font-semibold",
          direction === "ledger"
            ? "v-tabular"
            : "vd:font-display vd:text-[1rem]",
        )}
        style={{ color: "var(--v-accent)" }}
      >
        {numbered
          ? String(index + 1).padStart(2, "0")
          : "ABCDEFGHIJKLMNOPQRSTUVWXYZ"[index]}
      </span>
      <h4 className="vd:min-w-0 vd:flex-1 vd:font-display vd:text-[1.3125rem] vd:leading-[1.2] vd:font-medium">
        {name}
      </h4>
      {trailing}
    </div>
  );
}

function Prose({
  element,
  direction,
}: {
  element: Extract<Element, { type: "prose" }>;
  direction: Direction;
}) {
  if (element.display === "verbatim") return <Verbatim body={element.body} />;
  return (
    <div className="v-prose vd:mt-3 vd:grid vd:gap-flow">
      {element.body.split("\n\n").map((paragraph) => (
        <p
          key={paragraph.slice(0, 24)}
          className={cn("vd:font-prose vd:text-ink/92", PROSE[direction])}
        >
          {paragraph}
        </p>
      ))}
    </div>
  );
}

function TextSet({
  element,
  direction,
}: {
  element: Extract<Element, { type: "text_set" }>;
  direction: Direction;
}) {
  return (
    <ol className="vd:mt-4 vd:grid vd:gap-group">
      {element.items.map((item, index) => (
        <li key={item.id} className="vd:min-w-0">
          <ItemHead
            index={index}
            name={item.name}
            direction={direction}
            numbered={direction === "ledger"}
          />
          <div className="vd:mt-2.5 vd:grid vd:gap-flow vd:pl-[calc(0.875rem+0.75rem)]">
            {item.body.split("\n\n").map((paragraph) => (
              <p
                key={paragraph.slice(0, 24)}
                className={cn(
                  "vd:max-w-[62ch] vd:font-prose vd:text-ink/92",
                  ENTRY[direction],
                )}
              >
                {paragraph}
              </p>
            ))}
          </div>
        </li>
      ))}
    </ol>
  );
}

function FieldList({
  element,
  direction,
}: {
  element: Extract<Element, { type: "field_list" }>;
  direction: Direction;
}) {
  return (
    <dl className="v-tabular vd:mt-3">
      {element.fields.map((field) => (
        <div
          key={field.name}
          className="vd:flex vd:items-baseline vd:gap-3 vd:py-2.5 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
        >
          <dt className="vd:shrink-0 vd:text-meta vd:text-mute">
            {field.name}
          </dt>
          {direction === "ledger" && (
            <span
              aria-hidden="true"
              className="v-leader vd:mb-[0.4em] vd:h-px vd:min-w-4 vd:flex-1 vd:self-end"
            />
          )}
          <dd
            className={cn(
              "vd:text-ui",
              direction === "ledger" ? "vd:text-right" : "vd:ml-auto",
            )}
          >
            {field.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

/** Two speakers, told apart by where the line starts rather than by a bubble around it */
function DialogueSample({
  element,
  direction,
}: {
  element: Extract<Element, { type: "dialogue_sample" }>;
  direction: Direction;
}) {
  return (
    <ol className="vd:mt-4 vd:grid vd:gap-4">
      {element.turns.map((turn, index) => (
        <li
          key={`${turn.speaker}-${turn.line.slice(0, 16)}`}
          className={cn(
            "vd:grid vd:gap-1.5",
            turn.speaker === "user" && "vd:pl-8 vd:sm:pl-14",
          )}
        >
          <span className="vd:text-label vd:font-bold  vd:text-faint">
            {turn.speaker === "char" ? "Character" : "You"}
          </span>
          <p
            className={cn(
              "vd:max-w-[58ch] vd:font-prose vd:text-ink/92",
              ENTRY[direction],
              turn.speaker === "user" &&
                "vd:pl-4 vd:shadow-[inset_2px_0_0_var(--v-hair)]",
            )}
          >
            {turn.line}
          </p>
          {index < element.turns.length - 1 && <span className="vd:sr-only" />}
        </li>
      ))}
    </ol>
  );
}

/** An entry is judged on its keys and its firing rules as much as on its text */
function EntryTable({
  element,
  direction,
}: {
  element: Extract<Element, { type: "entry_table" }>;
  direction: Direction;
}) {
  return (
    <div className="vd:mt-3">
      <p className="vd:text-meta vd:text-mute">
        {element.total} entries, {element.enabledTotal} enabled
        <span aria-hidden="true" className="vd:px-2 vd:opacity-40">
          ·
        </span>
        showing {element.entries.length}
      </p>
      <ol className="vd:mt-4">
        {element.entries.map((entry) => (
          <li
            key={entry.id}
            className={cn(
              "vd:grid vd:gap-2.5 vd:py-5 vd:shadow-[inset_0_-1px_0_var(--v-hair)]",
              !entry.enabled && "vd:opacity-55",
            )}
          >
            <div className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-3 vd:gap-y-2">
              <span
                className="v-tabular vd:shrink-0 vd:text-meta vd:font-semibold"
                style={{ color: "var(--v-accent)" }}
              >
                {String(entry.order).padStart(3, "0")}
              </span>
              <h4 className="vd:min-w-0 vd:flex-1 vd:font-display vd:text-[1.3125rem] vd:leading-[1.2] vd:font-medium">
                {entry.name}
              </h4>
              {entry.constant && <Flag>Always on</Flag>}
              {!entry.enabled && <Flag off>Disabled</Flag>}
            </div>
            <Keys keys={entry.keys} />
            {entry.secondaryKeys && (
              <span className="vd:flex vd:items-baseline vd:gap-2">
                <span className="vd:text-label vd:font-bold  vd:text-faint">
                  and
                </span>
                <Keys keys={entry.secondaryKeys} weak />
              </span>
            )}
            <p
              className={cn(
                "vd:max-w-[68ch] vd:font-prose vd:text-ink/92",
                ENTRY[direction],
              )}
            >
              {entry.body}
            </p>
            <p className="vd:flex vd:flex-wrap vd:gap-x-4 vd:text-meta vd:text-faint">
              <span>Inserted {entry.position}</span>
              {entry.recursion && <span>Recursion {entry.recursion}</span>}
            </p>
          </li>
        ))}
      </ol>
    </div>
  );
}

const IMAGE_GRID = {
  small: "vd:grid-cols-3 vd:sm:grid-cols-4 vd:lg:grid-cols-6",
  medium: "vd:grid-cols-2 vd:sm:grid-cols-3",
  large: "vd:grid-cols-1 vd:sm:grid-cols-2",
};

function ImageSet({
  element,
}: {
  element: Extract<Element, { type: "image_set" }>;
}) {
  const named = element.role === "expressions";
  return (
    <ul
      className={cn("vd:mt-4 vd:grid vd:gap-3", IMAGE_GRID[element.itemSize])}
    >
      {element.images.map((image) => (
        <li key={image.id}>
          <figure className="vd:grid vd:gap-2">
            <Image
              src={image.src}
              alt={image.name ?? ""}
              sizes="(max-width: 40rem) 45vw, 18rem"
              className="vd:aspect-square vd:w-full vd:rounded-plate vd:object-cover"
            />
            {image.name && (
              <figcaption
                className={cn(
                  "vd:truncate vd:text-meta",
                  named ? "vd:font-mono vd:text-faint" : "vd:text-mute",
                )}
              >
                {image.name}
              </figcaption>
            )}
          </figure>
        </li>
      ))}
    </ul>
  );
}

function LinkList({
  element,
  direction,
}: {
  element: Extract<Element, { type: "link_list" }>;
  direction: Direction;
}) {
  return (
    <ul className="vd:mt-2">
      {element.links.map((link) => (
        <li key={link.id}>
          <a
            href="#top"
            className="v-row vd:group vd:block vd:py-4 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
          >
            <span className="vd:flex vd:items-start vd:justify-between vd:gap-4">
              <span className="vd:min-w-0">
                <span className="vd:block vd:text-label vd:font-bold  vd:text-faint">
                  {KIND_LABEL[link.kind]}
                </span>
                <span className="vd:mt-1.5 vd:block vd:font-display vd:text-[1.25rem] vd:leading-[1.24] vd:font-medium">
                  {link.label}
                </span>
                <span className="vd:mt-1 vd:block vd:text-meta vd:leading-6 vd:text-mute">
                  {link.note}
                </span>
              </span>
              <ArrowUpRight
                className={cn(
                  "vd:mt-1 vd:size-4 vd:shrink-0 vd:text-mute vd:transition vd:group-hover:text-ink vd:motion-reduce:transition-none",
                  direction !== "ledger" && "vd:group-hover:-translate-y-0.5",
                )}
              />
            </span>
          </a>
        </li>
      ))}
    </ul>
  );
}

/** A fragment's placement is the thing a reader is actually comparing presets on */
function PromptList({
  element,
}: {
  element: Extract<Element, { type: "prompt_list" }>;
}) {
  return (
    <div className="vd:mt-3">
      <p className="vd:text-meta vd:text-mute">
        {element.total} fragments
        <span aria-hidden="true" className="vd:px-2 vd:opacity-40">
          ·
        </span>
        showing {element.fragments.length}
      </p>
      <ol className="vd:mt-4">
        {element.fragments.map((fragment) => (
          <li
            key={fragment.id}
            className={cn(
              "vd:grid vd:gap-2.5 vd:py-5 vd:shadow-[inset_0_-1px_0_var(--v-hair)]",
              !fragment.enabled && "vd:opacity-55",
            )}
          >
            <div className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-3 vd:gap-y-2">
              <h4 className="vd:min-w-0 vd:flex-1 vd:font-display vd:text-[1.3125rem] vd:leading-[1.2] vd:font-medium">
                {fragment.name}
              </h4>
              <Flag>{fragment.role}</Flag>
              <Flag>
                {fragment.placement === "absolute"
                  ? `depth ${fragment.depth}`
                  : "in order"}
              </Flag>
              {!fragment.enabled && <Flag off>Disabled</Flag>}
            </div>
            {fragment.group && (
              <p className="vd:text-meta vd:text-faint">
                Group {fragment.group}
              </p>
            )}
            <p className="vd:max-w-[68ch] vd:font-mono vd:text-[0.875rem] vd:leading-7 vd:text-ink/90">
              {fragment.body}
            </p>
          </li>
        ))}
      </ol>
    </div>
  );
}

function VariableSchema({
  element,
}: {
  element: Extract<Element, { type: "variable_schema" }>;
}) {
  return (
    <ol className="vd:mt-3">
      {element.variables.map((variable) => (
        <li
          key={variable.id}
          className="vd:grid vd:gap-2 vd:py-4 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
        >
          <div className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-3 vd:gap-y-2">
            <code className="vd:font-mono vd:text-ui vd:font-semibold vd:text-ink">
              {variable.name}
            </code>
            <Flag>{variable.type}</Flag>
            <span className="vd:text-meta vd:text-faint">
              falls back to{" "}
              <code className="vd:font-mono">{variable.fallback}</code>
            </span>
          </div>
          {variable.choices && <Keys keys={variable.choices} />}
          <p className="vd:max-w-[62ch] vd:text-meta vd:leading-6 vd:text-mute">
            {variable.note}
          </p>
        </li>
      ))}
    </ol>
  );
}

function SettingGroup({
  element,
  direction,
}: {
  element: Extract<Element, { type: "setting_group" }>;
  direction: Direction;
}) {
  return (
    <dl className="v-tabular vd:mt-3">
      {element.settings.map((setting) => (
        <div
          key={setting.name}
          className="vd:flex vd:items-baseline vd:gap-3 vd:py-2.5 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
        >
          <dt className="vd:shrink-0 vd:font-mono vd:text-meta vd:text-mute">
            {setting.name}
          </dt>
          {direction === "ledger" && (
            <span
              aria-hidden="true"
              className="v-leader vd:mb-[0.4em] vd:h-px vd:min-w-4 vd:flex-1 vd:self-end"
            />
          )}
          <dd
            className={cn(
              "vd:font-mono vd:text-ui",
              direction === "ledger" ? "vd:text-right" : "vd:ml-auto",
            )}
          >
            {setting.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

function ScriptList({
  element,
}: {
  element: Extract<Element, { type: "script_list" }>;
}) {
  return (
    <ol className="vd:mt-3">
      {element.scripts.map((script) => (
        <li
          key={script.id}
          className={cn(
            "vd:grid vd:gap-2.5 vd:py-4 vd:shadow-[inset_0_-1px_0_var(--v-hair)]",
            !script.enabled && "vd:opacity-55",
          )}
        >
          <div className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-3 vd:gap-y-2">
            <h4 className="vd:min-w-0 vd:flex-1 vd:text-ui vd:font-semibold">
              {script.name}
            </h4>
            {!script.enabled && <Flag off>Disabled</Flag>}
          </div>
          <p className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-2 vd:font-mono vd:text-[0.8125rem]">
            <span className="vd:rounded-control vd:bg-ink/5 vd:px-2 vd:py-1 vd:text-ink/90">
              {script.find}
            </span>
            <span aria-hidden="true" className="vd:text-faint">
              →
            </span>
            <span className="vd:rounded-control vd:bg-ink/5 vd:px-2 vd:py-1 vd:text-ink/90">
              {script.replace || "(nothing)"}
            </span>
          </p>
          <p className="vd:text-meta vd:text-faint">
            Runs on {script.targets.join(" and ")} messages
          </p>
        </li>
      ))}
    </ol>
  );
}

/** A palette is judged as colour, so the swatch is the content and the value is the label */
function ColorSet({
  element,
}: {
  element: Extract<Element, { type: "color_set" }>;
}) {
  return (
    <ul className="vd:mt-4 vd:grid vd:grid-cols-2 vd:gap-x-6 vd:gap-y-3 vd:sm:grid-cols-3">
      {element.tokens.map((token) => (
        <li key={token.name} className="vd:flex vd:items-center vd:gap-3">
          <span
            aria-hidden="true"
            className="vd:size-11 vd:shrink-0 vd:rounded-control vd:bg-deep"
            style={{ background: token.value }}
          />
          <span className="vd:min-w-0">
            <span className="vd:block vd:truncate vd:text-ui">
              {token.name}
            </span>
            <span className="vd:block vd:font-mono vd:text-meta vd:text-faint">
              {token.value}
            </span>
          </span>
        </li>
      ))}
    </ul>
  );
}

function StylesheetSet({
  element,
}: {
  element: Extract<Element, { type: "stylesheet_set" }>;
}) {
  return (
    <ul className="vd:mt-3 vd:grid vd:gap-4 vd:lg:grid-cols-2">
      {element.sheets.map((sheet) => (
        <li key={sheet.id}>
          <div className="vd:flex vd:items-baseline vd:justify-between vd:gap-4">
            <code className="vd:font-mono vd:text-ui vd:font-semibold">
              {sheet.name}
            </code>
            <span className="vd:text-meta vd:text-faint">{sheet.size}</span>
          </div>
          <pre className="vd:mt-2 vd:overflow-x-auto vd:rounded-plate vd:bg-ink/4 vd:p-4 vd:bg-deep">
            <code className="vd:font-mono vd:text-[0.8125rem] vd:leading-6 vd:text-ink/85">
              {sheet.lines.join("\n")}
            </code>
          </pre>
        </li>
      ))}
    </ul>
  );
}

/** A pack is read as its members, so each record leads with a face and a name */
function RecordList({
  element,
  direction,
}: {
  element: Extract<Element, { type: "record_list" }>;
  direction: Direction;
}) {
  return (
    <ol className="vd:mt-4 vd:grid vd:gap-6 vd:lg:grid-cols-2 vd:lg:gap-x-14">
      {element.records.map((record) => (
        <li
          key={record.id}
          className="vd:flex vd:gap-5 vd:py-4 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
        >
          <Image
            src={record.avatar}
            alt=""
            sizes="6rem"
            className="vd:size-20 vd:shrink-0 vd:rounded-plate vd:object-cover"
          />
          <div className="vd:min-w-0">
            <div className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-3">
              <h4 className="vd:font-display vd:text-[1.375rem] vd:leading-[1.2] vd:font-medium">
                {record.lumiaName}
              </h4>
              <span className="v-tabular vd:text-meta vd:text-faint">
                {record.version}
              </span>
              <span className="vd:text-meta vd:text-faint">
                {record.genderIdentity}
              </span>
            </div>
            <p
              className={cn(
                "vd:mt-2 vd:max-w-[46ch] vd:font-prose vd:text-ink/92",
                ENTRY[direction],
              )}
            >
              {record.lumiaDefinition}
            </p>
            <p className="vd:mt-2 vd:text-meta vd:leading-6 vd:text-mute">
              {record.lumiaPersonality} {record.lumiaBehavior}
            </p>
          </div>
        </li>
      ))}
    </ol>
  );
}

export function ElementView({
  element,
  direction,
  labelled,
}: {
  element: Element;
  direction: Direction;
  labelled: boolean;
}) {
  return (
    <div className="vd:min-w-0">
      {labelled && <Label>{element.label}</Label>}
      {element.type === "prose" && (
        <Prose element={element} direction={direction} />
      )}
      {element.type === "text_set" && (
        <TextSet element={element} direction={direction} />
      )}
      {element.type === "field_list" && (
        <FieldList element={element} direction={direction} />
      )}
      {element.type === "dialogue_sample" && (
        <DialogueSample element={element} direction={direction} />
      )}
      {element.type === "entry_table" && (
        <EntryTable element={element} direction={direction} />
      )}
      {element.type === "image_set" && <ImageSet element={element} />}
      {element.type === "link_list" && (
        <LinkList element={element} direction={direction} />
      )}
      {element.type === "prompt_list" && <PromptList element={element} />}
      {element.type === "variable_schema" && (
        <VariableSchema element={element} />
      )}
      {element.type === "setting_group" && (
        <SettingGroup element={element} direction={direction} />
      )}
      {element.type === "script_list" && <ScriptList element={element} />}
      {element.type === "color_set" && <ColorSet element={element} />}
      {element.type === "stylesheet_set" && <StylesheetSet element={element} />}
      {element.type === "record_list" && (
        <RecordList element={element} direction={direction} />
      )}
    </div>
  );
}
