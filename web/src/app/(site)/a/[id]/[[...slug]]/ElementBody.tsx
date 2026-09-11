"use client";

import { UserRound } from "lucide-react";
import Image from "next/image";
import {
  type CSSProperties,
  type ReactNode,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";
import { Mosaic } from "@/components/media/Mosaic";
import { CopyButton } from "@/components/ui/copy-button";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import { Run, RunItem } from "@/components/ui/run";
import type {
  AssetElement,
  AssetImage,
  RecordListContent,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { elementLabel } from "@/lib/element-label";
import { contentItemCount, excerptDefinition } from "@/lib/page-arrangement";
import { nameSlot } from "@/lib/preset-slots";
import { formattingWasRemoved, richTextsOf } from "@/lib/rich-text";
import {
  CODE,
  ELEMENT_NAME,
  ELEMENT_RULE,
  ITEM_BODY,
  ITEM_META,
  ITEM_NAME,
  PASSAGE,
  PASSAGE_NAME,
  PROSE,
} from "./element-runs";
import { Lorebook } from "./Lorebook";
import {
  PromptList,
  ScriptList,
  SettingGroup,
  VariableSchema,
} from "./PresetElements";
import { ThemePalette, ThemeStyles } from "./ThemeElements";
import { Unfold } from "./Unfold";

const ROW_HEIGHTS = { small: 132, medium: 190, large: 260 };

export function ElementBody({
  element,
  isOwner,
  images = [],
  blockTitle,
  blockElements = 2,
  markEmpty = true,
  tools,
}: {
  element: AssetElement;
  isOwner: boolean;
  images?: AssetImage[];
  blockTitle?: string;
  blockElements?: number;
  markEmpty?: boolean;
  tools?: ReactNode;
}) {
  if (element.isEmpty && !isOwner) return null;

  const label = elementLabel(element, {
    elements: blockElements,
    title: blockTitle,
  });
  const facts = element.isEmpty ? "" : element.facts.join(" · ");
  return (
    <section className="group/element flex min-w-0 flex-col gap-3 [container-name:element] [container-type:inline-size]">
      {label || tools ? (
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
          {label ? (
            <h3 className={ELEMENT_NAME}>
              <span className="min-w-0">{label}</span>
              <span aria-hidden="true" className={ELEMENT_RULE} />
              {facts ? (
                <span className="shrink-0 font-ui text-meta font-normal text-mute">
                  {facts}
                </span>
              ) : null}
            </h3>
          ) : (
            <span />
          )}
          {tools}
        </div>
      ) : null}
      {element.isEmpty && markEmpty ? (
        <p className="font-ui text-label text-mute">Empty</p>
      ) : null}
      {element.isEmpty ? null : (
        <>
          <ExcerptedElementContent
            element={element}
            images={images}
            isOwner={isOwner}
          />
          {formattingWasRemoved(richTextsOf(element)) ? (
            <FormattingNotice />
          ) : null}
        </>
      )}
    </section>
  );
}

function ExcerptedElementContent({
  element,
  images,
  isOwner,
}: {
  element: AssetElement;
  images: AssetImage[];
  isOwner: boolean;
}) {
  const definition = excerptDefinition(element.type);
  const excerpt = useRef<HTMLDivElement>(null);
  const [lineCut, setLineCut] = useState(false);
  const [open, setOpen] = useState(false);
  const panelId = useId();
  const itemCount = visibleItemCount(element);
  const hasItemCut =
    definition.unit === "items" && itemCount > definition.limit;

  useEffect(() => {
    if (definition.unit !== "lines" || open) return;

    const node = excerpt.current;
    if (!node) return;
    const measure = () => {
      setLineCut(node.scrollHeight - node.clientHeight > 1);
    };
    const observer = new ResizeObserver(measure);
    measure();
    observer.observe(node);
    return () => observer.disconnect();
  }, [definition.unit, open]);

  if (definition.unit === "self" || element.type === "image_set") {
    return (
      <ElementContent element={element} images={images} isOwner={isOwner} />
    );
  }

  const isCut = definition.unit === "lines" ? lineCut : hasItemCut;
  const clipped = definition.unit === "lines" && !open;
  const itemLimit =
    definition.unit === "items" && !open ? definition.limit : undefined;

  function close() {
    setOpen(false);
    window.requestAnimationFrame(() => {
      const top = excerpt.current?.getBoundingClientRect().top ?? 0;
      if (top >= 0) return;
      excerpt.current?.scrollIntoView({ block: "center" });
    });
  }

  return (
    <Unfold
      id={`read-${element.id}`}
      isCut={isCut}
      more={excerptControlLabel(element, itemCount)}
      onToggle={() => (open ? close() : setOpen(true))}
      open={open}
      panelId={panelId}
    >
      <div
        className={cn(
          "relative min-w-0",
          clipped &&
            "max-h-[calc(var(--excerpt-lines)*1.78rem)] overflow-hidden",
          clipped &&
            isCut &&
            "[mask-image:linear-gradient(to_bottom,#000_calc(100%-62px),transparent)]",
        )}
        data-line-excerpt={definition.unit === "lines" ? true : undefined}
        data-truncated={isCut && !open ? true : undefined}
        id={panelId}
        ref={excerpt}
        style={
          definition.unit === "lines"
            ? ({ "--excerpt-lines": definition.limit } as CSSProperties)
            : undefined
        }
      >
        <ElementContent
          element={element}
          images={images}
          isOwner={isOwner}
          itemLimit={itemLimit}
        />
      </div>
    </Unfold>
  );
}

function visibleItemCount(element: AssetElement): number {
  if (element.type === "setting_group" && "settings" in element.content) {
    return element.content.settings.filter((setting) => setting.value != null)
      .length;
  }
  return contentItemCount(element);
}

function excerptControlLabel(element: AssetElement, itemCount: number): string {
  if (element.type !== "prose") {
    return `Show all ${itemCount} ${excerptNoun(element)}`;
  }
  const named = element.label.trim().toLocaleLowerCase();
  return named ? `Read the rest of the ${named}` : "Read the rest";
}

function excerptNoun(element: AssetElement): string {
  switch (element.type) {
    case "text_set":
      return element.role === "prompt_nudges" ? "nudges" : "items";
    case "field_list":
      return element.label.toLocaleLowerCase().includes("attribute")
        ? "attributes"
        : "fields";
    case "dialogue_sample":
      return "turns";
    case "image_set":
      return element.role === "expressions" ? "expressions" : "images";
    case "link_list":
      return "links";
    case "prompt_list":
      return "fragments";
    case "variable_schema":
      return "variables";
    case "setting_group":
      return "settings";
    case "color_set":
      return "colours";
    case "stylesheet_set":
      return "stylesheets";
    case "script_list":
      return "scripts";
    case "record_list":
      return "Lumia";
    default:
      return "items";
  }
}

export function ElementContent({
  element,
  images,
  isOwner = false,
  itemLimit,
}: {
  element: AssetElement;
  images: AssetImage[];
  isOwner?: boolean;
  itemLimit?: number;
}) {
  const { content } = element;

  if (element.type === "prose" && "text" in content) {
    return element.display === "verbatim" ? (
      <Verbatim text={content.text} />
    ) : (
      <RichText className={cn(PROSE, "max-w-[70ch]")} text={content.text} />
    );
  }

  if (element.type === "text_set" && "texts" in content) {
    const verbatim = element.display === "verbatim";
    const named = element.role === "prompt_nudges";
    return verbatim ? (
      <ol className="flex list-none flex-col gap-4">
        {content.texts.slice(0, itemLimit).map((item, index) => (
          <li key={`${index}-${item.name ?? ""}`}>
            {item.name ? (
              <p className={cn(ITEM_NAME, "mb-1.5")}>
                {named ? nameSlot(item.name).name : item.name}
              </p>
            ) : null}
            <Verbatim text={item.text} />
          </li>
        ))}
      </ol>
    ) : (
      <ol className={PASSAGE}>
        {content.texts.slice(0, itemLimit).map((item, index) => (
          <li className="min-w-0" key={`${index}-${item.name ?? ""}`}>
            {item.name ? <p className={PASSAGE_NAME}>{item.name}</p> : null}
            <RichText className={cn(PROSE, "max-w-[70ch]")} text={item.text} />
          </li>
        ))}
      </ol>
    );
  }

  if (element.type === "dialogue_sample" && "turns" in content) {
    return (
      <ol className="flex list-none flex-col gap-5">
        {content.turns.slice(0, itemLimit).map((turn, index) => (
          <li className="min-w-0" key={`${index}-${turn.speaker}`}>
            <p className={PASSAGE_NAME}>{turn.speaker}</p>
            <RichText className={cn(PROSE, "max-w-[70ch]")} text={turn.text} />
          </li>
        ))}
      </ol>
    );
  }

  if (element.type === "field_list" && "fields" in content) {
    return (
      <Run as="dl">
        {content.fields.slice(0, itemLimit).map((field, index) => (
          <RunItem
            as="div"
            className="!flex-row !gap-x-5 @max-[330px]:!flex-col @max-[330px]:!gap-y-0.5"
            itemKey={`${index}`}
            key={`${index}-${field.name ?? ""}`}
          >
            <dt className={cn(ITEM_META, "basis-[38%] shrink-0")}>
              {field.name || "Unnamed"}
            </dt>
            <dd className={cn(ITEM_BODY, "min-w-0 flex-1 text-ink")}>
              <RichText text={field.value} />
            </dd>
          </RunItem>
        ))}
      </Run>
    );
  }

  if (element.type === "link_list" && "links" in content) {
    return (
      <Run>
        {content.links.slice(0, itemLimit).map((link, index) => (
          <RunItem itemKey={`${index}`} key={`${index}-${link.url}`}>
            <a
              className="font-ui text-ui font-medium text-ink underline decoration-accent/55 underline-offset-[3px] [overflow-wrap:anywhere] hover:decoration-accent"
              href={link.url}
              rel="noreferrer nofollow"
              target="_blank"
            >
              {link.label || link.url}
            </a>
            {link.note ? (
              <RichText className={ITEM_BODY} text={link.note} />
            ) : null}
          </RunItem>
        ))}
      </Run>
    );
  }

  if (element.type === "image_set" && "images" in content) {
    return <Gallery content={content} element={element} images={images} />;
  }

  if (element.type === "entry_table" && "entries" in content) {
    return <Lorebook entries={content.entries} />;
  }

  if (
    element.type === "record_list" &&
    "schema" in content &&
    content.schema === "lumia"
  ) {
    return (
      <PackItems content={content} images={images} itemLimit={itemLimit} />
    );
  }

  if (element.type === "prompt_list" && "fragments" in content) {
    return (
      <PromptList content={content} isOwner={isOwner} itemLimit={itemLimit} />
    );
  }

  if (element.type === "setting_group" && "settings" in content) {
    return <SettingGroup itemLimit={itemLimit} settings={content.settings} />;
  }

  if (element.type === "color_set" && "modes" in content) {
    return <ThemePalette content={content} itemLimit={itemLimit} />;
  }

  if (element.type === "stylesheet_set" && "stylesheets" in content) {
    return <ThemeStyles content={content} itemLimit={itemLimit} />;
  }

  if (element.type === "variable_schema" && "variables" in content) {
    return (
      <VariableSchema itemLimit={itemLimit} variables={content.variables} />
    );
  }

  if (element.type === "script_list" && "scripts" in content) {
    return <ScriptList itemLimit={itemLimit} scripts={content.scripts} />;
  }

  return null;
}

function Verbatim({ text }: { text: string }) {
  return (
    <div className="relative">
      <pre className={cn(CODE, "max-w-[76ch] pr-14 text-ink/90")}>{text}</pre>
      <CopyButton
        className="absolute top-1 right-1"
        label="Copy this text"
        text={text}
      />
    </div>
  );
}

function Gallery({
  content,
  element,
  images,
}: {
  content: { images: { mediaId: string; name?: string }[] };
  element: AssetElement;
  images: AssetImage[];
}) {
  const imagesById = new Map(images.map((image) => [image.id, image]));
  const pictures = content.images
    .map((item) => {
      const image = imagesById.get(item.mediaId);
      return image
        ? {
            height: image.height,
            id: item.mediaId,
            name: item.name,
            src: image.detailUrl,
            width: image.width,
          }
        : null;
    })
    .filter((picture) => picture !== null);

  if (pictures.length === 0) return null;

  return (
    <Mosaic
      label={element.label || "Gallery"}
      pictures={pictures}
      rowHeight={ROW_HEIGHTS[element.itemSize ?? "medium"]}
    />
  );
}

const PACK_PRONOUNS: Record<
  RecordListContent["records"][number]["genderIdentity"],
  string
> = {
  0: "she / her",
  1: "he / him",
  2: "they / them",
};

function PackItems({
  content,
  images,
  itemLimit,
}: {
  content: RecordListContent;
  images: AssetImage[];
  itemLimit?: number;
}) {
  const imagesById = new Map(images.map((image) => [image.id, image]));
  return (
    <Run as="ol">
      {content.records.slice(0, itemLimit).map((record, index) => {
        const avatar = record.avatarUrl
          ? imagesById.get(record.avatarUrl)
          : undefined;
        return (
          <RunItem
            className="!flex-row !gap-x-4 py-4"
            itemKey={record.id ?? `${index}`}
            key={record.id ?? `${record.lumiaName}-${index}`}
          >
            <div className="grid aspect-square w-15 shrink-0 place-items-center self-start overflow-hidden rounded-control bg-media text-mute sm:w-20">
              {avatar ? (
                <Image
                  alt=""
                  className="size-full object-cover"
                  height={avatar.height}
                  sizes="96px"
                  src={avatar.thumbUrl}
                  unoptimized
                  width={avatar.width}
                />
              ) : (
                <UserRound aria-hidden="true" size={28} strokeWidth={1.3} />
              )}
            </div>
            <div className="min-w-0">
              <h4 className={ITEM_NAME}>
                {record.lumiaName || `Lumia ${index + 1}`}
              </h4>
              <p className={cn(ITEM_META, "mt-0.5")}>
                {PACK_PRONOUNS[record.genderIdentity]}
                {record.authorName ? ` · by ${record.authorName}` : ""}
                {` · v${record.version}`}
              </p>
              {record.lumiaDefinition ? (
                <RichText
                  className={cn(ITEM_BODY, "mt-2 max-w-[70ch]")}
                  text={record.lumiaDefinition}
                />
              ) : null}
            </div>
          </RunItem>
        );
      })}
    </Run>
  );
}
