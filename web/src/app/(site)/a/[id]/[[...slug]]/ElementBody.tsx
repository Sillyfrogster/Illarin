"use client";

import { Maximize2, UserRound } from "lucide-react";
import Image from "next/image";
import {
  type CSSProperties,
  Fragment,
  useEffect,
  useRef,
  useState,
} from "react";
import { ChipSet } from "@/components/ui/Chip";
import { CopyButton } from "@/components/ui/copy-button";
import { PerspectiveCarousel } from "@/components/ui/perspective-carousel";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import type {
  AssetElement,
  AssetImage,
  ColorSetContent,
  PresetSetting,
  PresetVariable,
  PromptListContent,
  RecordListContent,
  RegexScript,
  StylesheetSetContent,
  TypedValue,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { elementLabel } from "@/lib/element-label";
import {
  contentItemCount,
  excerptDefinition,
  opensFullScreen,
} from "@/lib/page-arrangement";
import { type NamedSlot, nameSlot, orderSettings } from "@/lib/preset-slots";
import { formattingWasRemoved, richTextsOf } from "@/lib/rich-text";
import { themeAccent, themeColorName } from "@/lib/theme-colors";
import { Lorebook } from "./Lorebook";

/** How wide one picture stands in a gallery, at each size a creator can pick */
const ITEM_WIDTHS = { small: 168, medium: 224, large: 296 };

const KEY_PREVIEW_LIMIT = 6;

/** A run of items down the page, each stood off a drawn line */
const STACK = "flex list-none flex-col gap-4.5";

const RUNG = "border-rule border-l-2 pl-4";

const ITEM_NAME = "mb-1 text-meta font-semibold text-ink";

const CODE =
  "overflow-x-auto rounded-control bg-deep p-4 font-mono text-meta leading-7 whitespace-pre-wrap [overflow-wrap:anywhere]";

export function ElementBody({
  element,
  isOwner,
  images = [],
  blockTitle,
  blockElements = 2,
  markEmpty = true,
  onExpand,
  onReadMore,
}: {
  element: AssetElement;
  isOwner: boolean;
  images?: AssetImage[];
  blockTitle?: string;
  /** How many elements the block renders, this one included. */
  blockElements?: number;
  /**
   * Whether an empty element says so. A page where nothing is filled in says
   * it once at the top instead, so the marker does not run down every label.
   */
  markEmpty?: boolean;
  onExpand?: () => void;
  onReadMore?: () => void;
}) {
  if (element.isEmpty && !isOwner) return null;

  const label = elementLabel(element, {
    elements: blockElements,
    title: blockTitle,
  });
  const expandable = isOwner && onExpand && opensFullScreen(element.type);

  return (
    <section className="flex min-w-0 flex-col gap-2.5 text-mute [container-name:element] [container-type:inline-size] [&_p]:text-prose">
      {label || expandable ? (
        <div className="flex items-start justify-between gap-4 max-sm:flex-col max-sm:gap-2.5">
          {label ? (
            <h3 className="font-prose text-label text-mute">{label}</h3>
          ) : (
            <span />
          )}
          {expandable ? (
            <button
              aria-label={`Edit ${element.label || "this content"} in full screen`}
              className="inline-flex min-h-11 shrink-0 items-center gap-1.5 rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink"
              data-measurement-ignore
              onClick={onExpand}
              type="button"
            >
              <Maximize2 aria-hidden="true" size={14} />
              Edit in full screen
            </button>
          ) : null}
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

  // A gallery turns rather than cuts, so every picture is already reachable.
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

/** Writing shown exactly as it was typed, with a way to take it away */
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

function ThemePalette({
  content,
  itemLimit,
}: {
  content: ColorSetContent;
  itemLimit?: number;
}) {
  let remaining = itemLimit ?? Number.POSITIVE_INFINITY;
  const modes = content.modes
    .map((mode) => {
      const colors = mode.colors.slice(0, remaining);
      remaining -= colors.length;
      return { ...mode, colors };
    })
    .filter((mode) => mode.colors.length > 0);
  return (
    <div className="flex flex-col gap-6">
      {modes.map((mode, modeIndex) => (
        <section className="min-w-0" key={mode.name || `mode-${modeIndex}`}>
          <h4 className="mb-3 font-display text-ui font-medium text-ink capitalize">
            {mode.name || "Palette"}
          </h4>
          <ul className="grid list-none gap-px overflow-hidden rounded-plate bg-rule [grid-template-columns:minmax(150px,1.45fr)_repeat(3,minmax(88px,1fr))] @max-[560px]:![grid-template-columns:repeat(2,minmax(0,1fr))]">
            {mode.colors.map((color, colorIndex) => (
              <li
                className={cn(
                  "grid min-w-0 grid-cols-[minmax(0,1fr)] gap-1 bg-plane px-3 pb-2.5",
                  colorIndex === 0 && "row-span-2 @max-[560px]:!row-span-1",
                )}
                key={color.id ?? `${color.name}-${colorIndex}`}
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "col-span-full -mx-3 mb-2",
                    colorIndex === 0
                      ? "h-full min-h-41 @max-[560px]:!h-17 @max-[560px]:!min-h-0"
                      : "h-17",
                  )}
                  style={{ backgroundColor: color.value }}
                />
                <span
                  className="min-w-0 truncate text-label font-semibold text-ink capitalize"
                  title={color.name}
                >
                  {themeColorName(color.name)}
                </span>
                <code
                  className="min-w-0 truncate font-mono text-[0.68rem] text-mute"
                  title={color.value}
                >
                  {color.value}
                </code>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

function ThemeStyles({
  content,
  itemLimit,
}: {
  content: StylesheetSetContent;
  itemLimit?: number;
}) {
  const sheets = (content.stylesheets ?? []).slice(
    0,
    Math.max(
      0,
      (itemLimit ?? Number.POSITIVE_INFINITY) - (content.global ? 1 : 0),
    ),
  );
  return (
    <div className="flex flex-col gap-4">
      {content.global ? (
        <Stylesheet css={content.global} name="Main stylesheet" />
      ) : null}
      {sheets.map((sheet, index) => (
        <Stylesheet
          css={sheet.css}
          key={sheet.id ?? `${sheet.name}-${index}`}
          name={sheet.name || `Component ${index + 1}`}
          off={!sheet.enabled}
        />
      ))}
      {(content.assets ?? []).length > 0 ? (
        <p className="!text-meta text-mute [overflow-wrap:anywhere]">
          {(content.assets ?? []).length.toLocaleString("en-GB")} attached{" "}
          {(content.assets ?? []).length === 1 ? "file" : "files"}:{" "}
          {(content.assets ?? []).map((asset) => asset.path).join(", ")}
        </p>
      ) : null}
    </div>
  );
}

function Stylesheet({
  name,
  css,
  off = false,
}: {
  name: string;
  css: string;
  off?: boolean;
}) {
  return (
    <section className={cn("min-w-0", off && "opacity-60")}>
      <div className="mb-2 flex items-center justify-between gap-3">
        <p className="!text-meta font-semibold text-ink">
          {name}
          {off ? <span className="ml-2 text-mute">Off</span> : null}
        </p>
        <CopyButton label={`Copy ${name}`} text={css} />
      </div>
      <pre className={cn(CODE, "max-h-44 overflow-auto text-ink")}>{css}</pre>
    </section>
  );
}

const PROMPT_ROLE_LABELS: Record<string, string> = {
  assistant: "Assistant",
  assistant_append: "Assistant, appended",
  system: "System",
  user: "User",
  user_append: "User, appended",
};

const PLACEMENT_LABELS: Record<string, string> = {
  in_history: "In the conversation",
  post_history: "After the conversation",
  pre_history: "Before the conversation",
};

function PromptList({
  content,
  isOwner,
  itemLimit,
}: {
  content: PromptListContent;
  isOwner: boolean;
  itemLimit?: number;
}) {
  const groups = content.groups ?? [];
  const fragments = (content.fragments ?? []).slice(0, itemLimit);
  const groupNames = new Map(groups.map((group) => [group.id, group.name]));
  const runs: { group?: string; fragments: PromptListContent["fragments"] }[] =
    [];
  for (const fragment of fragments) {
    const name = fragment.groupId
      ? groupNames.get(fragment.groupId)
      : undefined;
    const open = runs.at(-1);
    if (open && open.group === name) open.fragments.push(fragment);
    else runs.push({ group: name, fragments: [fragment] });
  }
  return (
    <div className="flex flex-col gap-6">
      {runs.map((run, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs hold no local state.
        <section key={index}>
          {run.group ? (
            <h4 className="mb-3 border-rule border-b pb-2 font-display text-ui font-medium text-ink">
              {run.group}
            </h4>
          ) : null}
          <Fragments
            fragments={run.fragments}
            grouped={Boolean(run.group)}
            isOwner={isOwner}
          />
        </section>
      ))}
    </div>
  );
}

function Fragments({
  fragments,
  grouped,
  isOwner,
}: {
  fragments: PromptListContent["fragments"];
  grouped: boolean;
  isOwner: boolean;
}) {
  return (
    <ol className={cn(STACK, grouped && "pl-4.5")}>
      {fragments.map((fragment, index) => (
        <li
          className={cn(RUNG, !fragment.enabled && "border-dashed opacity-60")}
          // biome-ignore lint/suspicious/noArrayIndexKey: Fragments hold no local state.
          key={index}
        >
          <div className="mb-1.5 flex flex-wrap items-baseline gap-x-2.5 gap-y-1">
            <span className="text-meta font-semibold text-ink [overflow-wrap:anywhere]">
              {fragment.name?.trim() ||
                fragment.marker?.trim() ||
                `Fragment ${index + 1}`}
            </span>
            <span className="text-meta text-mute">
              {fragment.role ? PROMPT_ROLE_LABELS[fragment.role] : null}
              {fragment.placement
                ? ` · ${PLACEMENT_LABELS[fragment.placement]}`
                : null}
              {fragment.enabled ? null : " · Off"}
              {fragment.protected ? " · Sealed prompt" : null}
            </span>
          </div>
          {fragment.protected && !isOwner ? (
            <p className="!text-ui text-mute italic">Sealed prompt</p>
          ) : fragment.marker ? (
            <p className="!text-ui text-mute italic">
              The app splices its own content in here.
            </p>
          ) : (
            <Paragraphs text={fragment.text} />
          )}
        </li>
      ))}
    </ol>
  );
}

function SettingGroup({
  settings,
  itemLimit,
}: {
  settings: PresetSetting[];
  itemLimit?: number;
}) {
  const shown = orderSettings(
    settings.filter((setting) => setting.value != null),
  ).slice(0, itemLimit);
  if (shown.length === 0) return null;
  const named = shown.filter((setting) => setting.slot.rank !== "unrecognised");
  const raw = shown.filter((setting) => setting.slot.rank === "unrecognised");
  return (
    <>
      {named.length > 0 ? <Settings settings={named} /> : null}
      {raw.length > 0 ? (
        <>
          <p className="mt-5 border-rule border-t pt-4 !text-label text-mute">
            As the file names them
          </p>
          <Settings raw settings={raw} />
        </>
      ) : null}
    </>
  );
}

function Settings({
  settings,
  raw,
}: {
  settings: Array<PresetSetting & { slot: NamedSlot }>;
  raw?: boolean;
}) {
  return (
    <dl
      className={cn(
        "grid items-baseline gap-x-5 gap-y-3 @max-[330px]:![grid-template-columns:minmax(0,1fr)] @max-[330px]:gap-y-1",
        raw
          ? "[grid-template-columns:fit-content(62%)_minmax(0,1fr)]"
          : "[grid-template-columns:fit-content(38%)_minmax(0,1fr)]",
      )}
    >
      {settings.map((setting) => (
        <div className="contents" key={setting.id ?? setting.name}>
          <dt
            className={cn(
              "text-mute [overflow-wrap:anywhere]",
              raw ? "font-mono text-meta" : "text-label",
            )}
          >
            {setting.slot.name}
          </dt>
          <dd className="text-ui text-ink tabular-nums [overflow-wrap:anywhere]">
            <SettingValue name={setting.name} value={setting.value} />
            {setting.slot.note ? (
              <span className="mt-0.5 block max-w-[46ch] text-meta text-mute text-pretty">
                {setting.slot.note}
              </span>
            ) : null}
          </dd>
        </div>
      ))}
    </dl>
  );
}

/**
 * A setting's value. Text made only of spaces and newlines is shown as it is
 * written, because saying a setting holds nothing when it holds two blank
 * lines is a lie a reader would act on.
 */
function SettingValue({
  name,
  value,
}: {
  name: string;
  value: TypedValue | undefined;
}) {
  const accent = name === "accent" ? themeAccent(value?.text) : null;
  if (accent) {
    return (
      <span className="inline-flex items-center gap-2">
        <span
          aria-hidden="true"
          className="size-4.5 shrink-0 rounded-full inset-ring inset-ring-rule"
          style={{ backgroundColor: accent.css }}
        />
        {accent.label}
      </span>
    );
  }
  if (value?.text != null && value.text !== "" && value.text.trim() === "") {
    return (
      <code className="block rounded-control bg-deep px-2 py-0.5 font-mono text-meta whitespace-pre-wrap">
        {value.text}
      </code>
    );
  }
  return <>{writeValue(value)}</>;
}

function writeValue(value: TypedValue | undefined): string {
  if (!value) return "";
  if (value.number != null) {
    return value.number.toLocaleString("en-GB", { maximumFractionDigits: 20 });
  }
  if (value.boolean != null) return value.boolean ? "Yes" : "No";
  if (value.strings) {
    return value.strings.length === 0 ? "Nothing" : value.strings.join(", ");
  }
  return value.text === "" ? "Nothing" : (value.text ?? "");
}

function VariableSchema({
  variables,
  itemLimit,
}: {
  variables: PresetVariable[];
  itemLimit?: number;
}) {
  return (
    <ul className="flex list-none flex-col gap-4">
      {variables.slice(0, itemLimit).map((variable, index) => (
        <li className={RUNG} key={variable.id ?? `${index}-${variable.name}`}>
          <p className={ITEM_NAME}>{variable.label?.trim() || variable.name}</p>
          {variable.description ? (
            <RichText text={variable.description} />
          ) : null}
          {variable.options && variable.options.length > 0 ? (
            <ChipSet
              className="mt-2"
              items={variable.options.map((option, position) => ({
                id: `${position}-${option.value}`,
                label: option.label || option.value,
              }))}
              limit={KEY_PREVIEW_LIMIT}
            />
          ) : null}
        </li>
      ))}
    </ul>
  );
}

function ScriptList({
  scripts,
  itemLimit,
}: {
  scripts: RegexScript[];
  itemLimit?: number;
}) {
  return (
    <ul className="flex list-none flex-col gap-4">
      {scripts.slice(0, itemLimit).map((script, index) => (
        <li
          className={cn(RUNG, !script.enabled && "border-dashed opacity-60")}
          key={script.id ?? index}
        >
          <p className={ITEM_NAME}>
            {script.name?.trim() || `Script ${index + 1}`}
            {script.enabled ? null : (
              <span className="font-normal text-mute"> · Off</span>
            )}
          </p>
          <p className="mt-1 flex flex-wrap items-center gap-2 [&_code]:rounded-control [&_code]:bg-deep [&_code]:px-2 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-meta [&_code]:whitespace-pre-wrap [&_code]:[overflow-wrap:anywhere]">
            <code>{script.find}</code>
            <span aria-hidden="true">→</span>
            <code>{script.replace || "nothing"}</code>
          </p>
        </li>
      ))}
    </ul>
  );
}

function Paragraphs({ text }: { text: string }) {
  const paragraphs = text.split(/\n{2,}/).filter((line) => line.trim() !== "");
  return (
    <div className="[&>p+p]:mt-[0.85em]">
      {paragraphs.map((paragraph, index) => (
        <p key={`${index}-${paragraph.slice(0, 24)}`}>{paragraph}</p>
      ))}
    </div>
  );
}
