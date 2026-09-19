"use client";

import { Plus, Trash2 } from "lucide-react";
import { Children, Fragment } from "react";
import type { WorkBlock, WorkElement, WorkImage } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { elementLabel } from "@/lib/element-label";
import { ElementBody } from "../ElementBody";
import { ELEMENT_NAME, ELEMENT_RULE, ITEM_NAME } from "../element-runs";
import { EditableText } from "./EditableText";
import { ElementTools } from "./ElementTools";
import { isEmptyContent, writesInPlace } from "./save";
import { useWorkspace } from "./state";

const EDITABLE_ITEM =
  "flex min-w-0 flex-col gap-1.5 px-4 py-3.5 not-first:border-rule/45 not-first:border-t";

const ADD =
  "inline-flex min-h-11 items-center gap-1.5 rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink";

const DROP =
  "inline-flex size-9 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-stop";

const TEXT_SET_NOUNS: Record<string, string> = {
  greetings: "greeting",
  group_greetings: "group-only greeting",
  prompt_nudges: "nudge",
};

export function elementCursor(
  elementId: string,
  ...parts: (string | number)[]
) {
  return [elementId, ...parts].join(":");
}

export function EditableElementSection({
  block,
  element,
  images,
  markEmpty,
}: {
  block: WorkBlock;
  element: WorkElement;
  images: WorkImage[];
  markEmpty: boolean;
}) {
  const tools = <ElementTools block={block} element={element} />;

  if (element.fromFile || !writesInPlace(element)) {
    const body = (
      <ElementBody
        blockElements={block.elements.length}
        blockTitle={block.title}
        element={element}
        images={images}
        isOwner
        markEmpty={markEmpty}
        tools={tools}
      />
    );
    if (!element.fromFile) return body;
    return (
      <div className="flex min-w-0 flex-col gap-2">
        {body}
        <p className="text-meta text-mute">
          From the file. Upload a new version to change it.
        </p>
      </div>
    );
  }

  const label = elementLabel(element, {
    elements: block.elements.length,
    title: block.title,
  });
  return (
    <section className="group/element flex min-w-0 flex-col gap-3 [container-name:element] [container-type:inline-size]">
      <div className="flex items-center justify-between gap-4">
        {label ? (
          <h3 className={ELEMENT_NAME}>
            <span className="min-w-0">{label}</span>
            <span aria-hidden="true" className={ELEMENT_RULE} />
          </h3>
        ) : (
          <span />
        )}
        {tools}
      </div>
      <EditableElement blockId={block.id} element={element} />
    </section>
  );
}

export function EditableElement({
  blockId,
  element,
}: {
  blockId: string;
  element: WorkElement;
}) {
  const workspace = useWorkspace();

  function write(content: WorkElement["content"]) {
    const next = { ...element, content } as WorkElement;
    workspace.writeElement(blockId, { ...next, isEmpty: isEmptyContent(next) });
  }

  const field = (parts: (string | number)[], options: FieldOptions) => (
    <Field
      cursor={elementCursor(element.id, ...parts)}
      {...options}
      key={parts.join(":")}
    />
  );

  const { content } = element;
  const takesMarkdown = element.display !== "verbatim";

  if (element.type === "prose" && "text" in content) {
    return field(["text"], {
      className: "max-w-[70ch] font-prose text-prose text-ink",
      label: element.label || "Text",
      onChange: (text) => write({ text }),
      placeholder: "Write here",
      rich: takesMarkdown,
      value: content.text,
    });
  }

  if (element.type === "text_set" && "texts" in content) {
    const texts = content.texts;
    const noun = TEXT_SET_NOUNS[element.role ?? ""] ?? "passage";
    return (
      <Run
        addLabel={addLabel(noun)}
        onAdd={() => write({ texts: [...texts, { name: "", text: "" }] })}
      >
        {texts.map((item, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: A key that moved with the writing would take the caret with it.
          <li className={cn(EDITABLE_ITEM, "group/item relative")} key={index}>
            <Drop
              label={`Remove ${item.name || `${noun} ${index + 1}`}`}
              onDrop={() =>
                write({ texts: texts.filter((_, at) => at !== index) })
              }
            />
            {field([index, "name"], {
              as: "p",
              className: ITEM_NAME,
              label: `Name of ${noun} ${index + 1}`,
              onChange: (name) =>
                write({ texts: replace(texts, index, { ...item, name }) }),
              placeholder: "Name this one",
              singleLine: true,
              value: item.name ?? "",
            })}
            {field([index, "text"], {
              className: "font-prose text-prose text-ink",
              label: `${noun.charAt(0).toUpperCase()}${noun.slice(1)} ${index + 1}`,
              onChange: (text) =>
                write({ texts: replace(texts, index, { ...item, text }) }),
              placeholder: "Write here",
              rich: takesMarkdown,
              value: item.text,
            })}
          </li>
        ))}
      </Run>
    );
  }

  if (element.type === "dialogue_sample" && "turns" in content) {
    const turns = content.turns;
    return (
      <Run
        addLabel="Add a turn"
        onAdd={() => write({ turns: [...turns, { speaker: "", text: "" }] })}
      >
        {turns.map((turn, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: A key that moved with the writing would take the caret with it.
          <li className={cn(EDITABLE_ITEM, "group/item relative")} key={index}>
            <Drop
              label={`Remove turn ${index + 1}`}
              onDrop={() =>
                write({ turns: turns.filter((_, at) => at !== index) })
              }
            />
            {field([index, "speaker"], {
              as: "p",
              className: ITEM_NAME,
              label: `Speaker of turn ${index + 1}`,
              onChange: (speaker) =>
                write({ turns: replace(turns, index, { ...turn, speaker }) }),
              placeholder: "Who speaks",
              singleLine: true,
              value: turn.speaker,
            })}
            {field([index, "text"], {
              className: "font-prose text-prose text-ink",
              label: `Turn ${index + 1}`,
              onChange: (text) =>
                write({ turns: replace(turns, index, { ...turn, text }) }),
              placeholder: "What they say",
              rich: true,
              value: turn.text,
            })}
          </li>
        ))}
      </Run>
    );
  }

  if (element.type === "field_list" && "fields" in content) {
    const fields = content.fields;
    return (
      <>
        <dl className="grid items-baseline gap-x-5 gap-y-3 [grid-template-columns:fit-content(38%)_minmax(0,1fr)] @max-[330px]:![grid-template-columns:minmax(0,1fr)]">
          {fields.map((item, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: A key that moved with the writing would take the caret with it.
            <Fragment key={index}>
              <dt className="text-label text-mute">
                {field([index, "name"], {
                  as: "span",
                  className: "text-label text-mute",
                  label: `Name of field ${index + 1}`,
                  onChange: (name) =>
                    write({
                      fields: replace(fields, index, { ...item, name }),
                    }),
                  placeholder: "Unnamed",
                  singleLine: true,
                  value: item.name ?? "",
                })}
              </dt>
              <dd className="group/item relative flex items-baseline gap-2 text-ui text-ink">
                {field([index, "value"], {
                  as: "span",
                  className: "min-w-0 flex-1 text-ui text-ink",
                  label: `Field ${index + 1}`,
                  onChange: (value) =>
                    write({
                      fields: replace(fields, index, { ...item, value }),
                    }),
                  placeholder: "Write here",
                  rich: true,
                  value: item.value,
                })}
                <Drop
                  inline
                  label={`Remove ${item.name || `field ${index + 1}`}`}
                  onDrop={() =>
                    write({ fields: fields.filter((_, at) => at !== index) })
                  }
                />
              </dd>
            </Fragment>
          ))}
        </dl>
        <Add
          label="Add a field"
          onAdd={() => write({ fields: [...fields, { name: "", value: "" }] })}
        />
      </>
    );
  }

  if (element.type === "link_list" && "links" in content) {
    const links = content.links;
    return (
      <Run
        addLabel="Add a link"
        list="ul"
        onAdd={() => write({ links: [...links, { label: "", url: "" }] })}
      >
        {links.map((link, index) => (
          <li
            className={cn(EDITABLE_ITEM, "group/item relative")}
            // biome-ignore lint/suspicious/noArrayIndexKey: A key that moved with the writing would take the caret with it.
            key={index}
          >
            <Drop
              label={`Remove ${link.label || `link ${index + 1}`}`}
              onDrop={() =>
                write({ links: links.filter((_, at) => at !== index) })
              }
            />
            {field([index, "label"], {
              as: "span",
              className: "text-ui font-medium text-ink",
              label: `Name of link ${index + 1}`,
              onChange: (label) =>
                write({ links: replace(links, index, { ...link, label }) }),
              placeholder: "Name this link",
              singleLine: true,
              value: link.label ?? "",
            })}
            {field([index, "url"], {
              as: "span",
              className: "font-mono text-meta text-mute",
              label: `Address of link ${index + 1}`,
              onChange: (url) =>
                write({ links: replace(links, index, { ...link, url }) }),
              placeholder: "https://",
              singleLine: true,
              value: link.url,
            })}
            {field([index, "note"], {
              as: "span",
              className: "text-meta text-mute",
              label: `Note on link ${index + 1}`,
              onChange: (note) =>
                write({ links: replace(links, index, { ...link, note }) }),
              placeholder: "Add a note",
              rich: true,
              value: link.note ?? "",
            })}
          </li>
        ))}
      </Run>
    );
  }

  return null;
}

type FieldOptions = {
  as?: "p" | "span" | "div";
  className?: string;
  label: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rich?: boolean;
  singleLine?: boolean;
  value: string;
};

function Field({ cursor, ...options }: FieldOptions & { cursor: string }) {
  const workspace = useWorkspace();
  return (
    <EditableText
      active={workspace.cursor === cursor}
      activate={() => workspace.setCursor(cursor)}
      done={() => workspace.setCursor(null)}
      live={workspace.editing}
      {...options}
    />
  );
}

function Run({
  addLabel,
  children,
  list: List = "ol",
  onAdd,
}: {
  addLabel: string;
  children: React.ReactNode;
  list?: "ol" | "ul";
  onAdd: () => void;
}) {
  return (
    <>
      {Children.count(children) === 0 ? null : (
        <List className="flex list-none flex-col rounded-plate bg-inset">
          {children}
        </List>
      )}
      <Add label={addLabel} onAdd={onAdd} />
    </>
  );
}

function Add({ label, onAdd }: { label: string; onAdd: () => void }) {
  const workspace = useWorkspace();
  if (!workspace.editing) return null;
  return (
    <button
      className={cn(ADD, "-ml-3 mt-1 self-start")}
      onClick={onAdd}
      type="button"
    >
      <Plus aria-hidden="true" size={15} />
      {label}
    </button>
  );
}

function Drop({
  inline = false,
  label,
  onDrop,
}: {
  inline?: boolean;
  label: string;
  onDrop: () => void;
}) {
  const workspace = useWorkspace();
  if (!workspace.editing) return null;
  return (
    <button
      aria-label={label}
      className={cn(
        DROP,
        "opacity-0 group-focus-within/item:opacity-100 group-hover/item:opacity-100 focus-visible:opacity-100",
        inline ? "self-start" : "absolute top-0 right-0",
      )}
      onClick={onDrop}
      type="button"
    >
      <Trash2 aria-hidden="true" size={15} />
    </button>
  );
}

function addLabel(noun: string): string {
  return `Add ${/^[aeiou]/i.test(noun) ? "an" : "a"} ${noun}`;
}

function replace<Item>(items: Item[], index: number, item: Item): Item[] {
  return items.map((existing, at) => (at === index ? item : existing));
}
