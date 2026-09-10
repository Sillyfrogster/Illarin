"use client";

import { UserRound } from "lucide-react";
import Image from "next/image";
import {
  type CSSProperties,
  Fragment,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";
import { CopyButton } from "@/components/ui/copy-button";
import { PerspectiveCarousel } from "@/components/ui/perspective-carousel";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
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
import { CODE, ITEM_NAME, RUNG, STACK } from "./element-runs";
import { Lorebook } from "./Lorebook";
import {
  PromptList,
  ScriptList,
  SettingGroup,
  VariableSchema,
} from "./PresetElements";
import { ThemePalette, ThemeStyles } from "./ThemeElements";

const ITEM_WIDTHS = { small: 168, medium: 224, large: 296 };

export function ElementBody({
  element,
  isOwner,
  images = [],
  blockTitle,
  blockElements = 2,
  markEmpty = true,
  onReadMore,
  tools,
}: {
  element: AssetElement;
  isOwner: boolean;
  images?: AssetImage[];
  blockTitle?: string;
  blockElements?: number;
  markEmpty?: boolean;
  onReadMore?: () => void;
  tools?: ReactNode;
}) {
  if (element.isEmpty && !isOwner) return null;

  const label = elementLabel(element, {
    elements: blockElements,
    title: blockTitle,
  });
  return (
    <section className="group/element flex min-w-0 flex-col gap-2.5 text-mute [container-name:element] [container-type:inline-size] [&_p]:text-prose">
      {label || tools ? (
        <div className="flex items-start justify-between gap-4">
          {label ? (
            <h3 className="font-prose text-label text-mute">{label}</h3>
          ) : (
            <span />
          )}
          {tools}
        </div>
      ) : null}
      {element.isEmpty && markEmpty ? (
        <p className="!text-label text-mute">Empty</p>
      ) : null}
      {element.isEmpty ? null : (
        <>
          <ExcerptedElementContent
            element={element}
            images={images}
            isOwner={isOwner}
            onReadMore={onReadMore}
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
  onReadMore,
}: {
  element: AssetElement;
  images: AssetImage[];
  isOwner: boolean;
  onReadMore?: () => void;
}) {
  const definition = excerptDefinition(element.type);
  const excerpt = useRef<HTMLDivElement>(null);
  const [lineCut, setLineCut] = useState(false);
  const itemCount = visibleItemCount(element);
  const hasItemCut =
    definition.unit === "items" && itemCount > definition.limit;

  useEffect(() => {
    if (definition.unit !== "lines") return;

    const node = excerpt.current;
    if (!node) return;
    const measure = () => {
      setLineCut(node.scrollHeight - node.clientHeight > 1);
    };
    const observer = new ResizeObserver(measure);
    measure();
    observer.observe(node);
    return () => observer.disconnect();
  }, [definition.unit]);

  const isCut = definition.unit === "lines" ? lineCut : hasItemCut;
  const itemLimit = definition.unit === "items" ? definition.limit : undefined;

  if (definition.unit === "self" || element.type === "image_set") {
    return (
      <ElementContent element={element} images={images} isOwner={isOwner} />
    );
  }

  return (
    <>
      <div
        className={cn(
          "relative min-w-0",
          definition.unit === "lines" &&
            "max-h-[calc(var(--excerpt-lines)*1.78rem)] overflow-hidden",
          isCut &&
            "[mask-image:linear-gradient(to_bottom,#000_calc(100%-62px),transparent)]",
        )}
        data-line-excerpt={definition.unit === "lines" ? true : undefined}
        data-truncated={isCut ? true : undefined}
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
      {isCut && onReadMore ? (
        <button
          className="self-start text-meta font-semibold text-accent underline underline-offset-4 outline-offset-3 hover:text-ink"
          data-read-more
          id={`read-${element.id}`}
          onClick={onReadMore}
          type="button"
        >
          {excerptControlLabel(element, itemCount)}
        </button>
      ) : null}
    </>
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
  if (element.type === "prose") {
    return `the rest of ${element.label.trim().toLocaleLowerCase() || "this text"}`;
  }
  return `all ${itemCount} ${excerptNoun(element)}`;
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
      <RichText className="max-w-[70ch]" text={content.text} />
    );
  }

  if (element.type === "text_set" && "texts" in content) {
    const verbatim = element.display === "verbatim";
    const named = element.role === "prompt_nudges";
    return (
      <ol className={STACK}>
        {content.texts.slice(0, itemLimit).map((item, index) => (
          <li className={RUNG} key={`${index}-${item.name ?? ""}`}>
            {item.name ? (
              <p className={cn(ITEM_NAME, "!text-meta")}>
                {named ? nameSlot(item.name).name : item.name}
              </p>
            ) : null}
            {verbatim ? (
              <Verbatim text={item.text} />
            ) : (
              <RichText text={item.text} />
            )}
          </li>
        ))}
      </ol>
    );
  }

  if (element.type === "dialogue_sample" && "turns" in content) {
    return (
      <ol className={STACK}>
        {content.turns.slice(0, itemLimit).map((turn, index) => (
          <li className={RUNG} key={`${index}-${turn.speaker}`}>
            <p className={cn(ITEM_NAME, "!text-meta")}>{turn.speaker}</p>
            <RichText text={turn.text} />
          </li>
        ))}
      </ol>
    );
  }

  if (element.type === "field_list" && "fields" in content) {
    return (
      <dl className="grid items-baseline gap-x-5 gap-y-3 [grid-template-columns:fit-content(38%)_minmax(0,1fr)] @max-[330px]:![grid-template-columns:minmax(0,1fr)] @max-[330px]:gap-y-1">
        {content.fields.slice(0, itemLimit).map((field, index) => (
          <Fragment key={`${index}-${field.name ?? ""}`}>
            <dt className="text-label text-mute [overflow-wrap:anywhere]">
              {field.name || "Unnamed"}
            </dt>
            <dd className="text-ui text-ink [overflow-wrap:anywhere] [&_p]:!text-ui [&_p]:!leading-normal [&_p]:text-ink">
              <RichText text={field.value} />
            </dd>
          </Fragment>
        ))}
      </dl>
    );
  }

  if (element.type === "link_list" && "links" in content) {
    return (
      <ul className="flex list-none flex-col gap-3">
        {content.links.slice(0, itemLimit).map((link, index) => (
          <li
            className={cn(RUNG, "flex flex-col gap-1")}
            key={`${index}-${link.url}`}
          >
            <a
              className="text-ui font-medium text-ink underline decoration-accent/55 underline-offset-[3px] [overflow-wrap:anywhere] hover:decoration-accent"
              href={link.url}
              rel="noreferrer nofollow"
              target="_blank"
            >
              {link.label || link.url}
            </a>
            {link.note ? (
              <RichText
                className="[&_p]:!text-meta [&_p]:!leading-normal"
                text={link.note}
              />
            ) : null}
          </li>
        ))}
      </ul>
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
      <pre className={cn(CODE, "max-w-[70ch] pr-14")}>{text}</pre>
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

  if (pictures.length === 1) {
    const only = pictures[0];
    return (
      <figure className="max-w-90">
        <Image
          alt={only.name || ""}
          className="h-auto w-full rounded-plate bg-deep"
          height={only.height}
          sizes="360px"
          src={only.src}
          unoptimized
          width={only.width}
        />
        {only.name ? (
          <figcaption className="mt-2 text-label text-mute [overflow-wrap:anywhere]">
            {only.name}
          </figcaption>
        ) : null}
      </figure>
    );
  }

  return (
    <PerspectiveCarousel
      label={element.label || "Gallery"}
      pictures={pictures}
      slideWidth={ITEM_WIDTHS[element.itemSize ?? "medium"]}
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
    <ol className="flex list-none flex-col">
      {content.records.slice(0, itemLimit).map((record, index) => {
        const avatar = record.avatarUrl
          ? imagesById.get(record.avatarUrl)
          : undefined;
        return (
          <li
            className="grid grid-cols-[62px_minmax(0,1fr)] gap-4 py-4 not-first:border-rule not-first:border-t sm:grid-cols-[80px_minmax(0,1fr)]"
            key={record.id ?? `${record.lumiaName}-${index}`}
          >
            <div className="grid aspect-square w-full place-items-center self-start overflow-hidden rounded-plate bg-deep text-mute">
              {avatar ? (
                <Image
                  alt=""
                  className="size-full object-contain"
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
              <div className="mb-2 flex flex-wrap items-baseline gap-x-3.5 gap-y-1">
                <h4 className="font-display text-ui font-medium text-ink [overflow-wrap:anywhere]">
                  {record.lumiaName || `Lumia ${index + 1}`}
                </h4>
                <span className="text-label text-mute [overflow-wrap:anywhere]">
                  {PACK_PRONOUNS[record.genderIdentity]}
                  {record.authorName ? ` · by ${record.authorName}` : ""}
                  {` · v${record.version}`}
                </span>
              </div>
              {record.lumiaDefinition ? (
                <RichText
                  className="max-w-[70ch] [&_p]:!text-ui [&_p]:!leading-normal"
                  text={record.lumiaDefinition}
                />
              ) : null}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
