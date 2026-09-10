"use client";

import { ImagePlus } from "lucide-react";
import Image from "next/image";
import { useMemo, useRef, useState } from "react";
import {
  type AssetElement,
  type AssetImage,
  addAssetImage,
} from "@/lib/api/query";
import { useWorkingCopy } from "@/lib/working-copy";
import { EntryTableEditor } from "./EntryTableEditor";
import { PackEditor } from "./PackEditor";
import {
  PromptListEditor,
  ScriptListEditor,
  SettingGroupEditor,
  VariableSchemaEditor,
} from "./PresetEditors";
import { ColorSetEditor, StylesheetSetEditor } from "./ThemeEditors";
import { moveItem, replaceAt, without } from "./workspace/collection";
import { Field, InlineItem, Note, Switch, TextField } from "./workspace/fields";

type ImageItem = {
  mediaId: string;
  name?: string;
  omitFromDownloads?: boolean;
};

export function ElementFields({
  assetId,
  chosen,
  element,
  images,
  onChange,
  onChoose,
  onImageAdded,
  pending,
}: {
  assetId: string;
  chosen: string | null;
  element: AssetElement;
  images: AssetImage[];
  onChange: (element: AssetElement) => void;
  onChoose: (key: string | null) => void;
  onImageAdded: () => void;
  pending: boolean;
}) {
  if (element.type === "image_set" && "images" in element.content) {
    return (
      <ImageEditor
        assetId={assetId}
        images={images}
        isGallery={element.role === "gallery"}
        items={element.content.images}
        mediaRole={element.role === "expressions" ? "expression" : "gallery"}
        onAdded={onImageAdded}
        onChange={(added) =>
          onChange({
            ...element,
            content: { images: added },
            isEmpty: added.length === 0,
          })
        }
        pending={pending}
      />
    );
  }

  if (element.type === "entry_table" && "entries" in element.content) {
    return (
      <EntryTableEditor
        chosen={chosen}
        entries={element.content.entries}
        onChange={(entries) =>
          onChange({
            ...element,
            content: { entries },
            isEmpty: entries.every((entry) => entry.text.trim() === ""),
          })
        }
        onChoose={onChoose}
        pending={pending}
      />
    );
  }

  if (
    element.type === "record_list" &&
    "schema" in element.content &&
    element.content.schema === "lumia"
  ) {
    return (
      <PackEditor
        assetId={assetId}
        chosen={chosen}
        content={element.content}
        images={images}
        onChange={(content) =>
          onChange({
            ...element,
            content,
            isEmpty: content.records.length === 0,
          })
        }
        onChoose={onChoose}
        onImageAdded={onImageAdded}
        pending={pending}
      />
    );
  }

  if (element.type === "prompt_list" && "fragments" in element.content) {
    return (
      <PromptListEditor
        chosen={chosen}
        content={element.content}
        onChange={(content) =>
          onChange({
            ...element,
            content,
            isEmpty: content.fragments.length === 0,
          })
        }
        onChoose={onChoose}
        pending={pending}
      />
    );
  }

  if (element.type === "setting_group" && "settings" in element.content) {
    return (
      <SettingGroupEditor
        onChange={(settings) =>
          onChange({
            ...element,
            content: { settings },
            isEmpty: settings.every((setting) => setting.value == null),
          })
        }
        pending={pending}
        settings={element.content.settings}
      />
    );
  }

  if (element.type === "color_set" && "modes" in element.content) {
    return (
      <ColorSetEditor
        content={element.content}
        onChange={(content) =>
          onChange({
            ...element,
            content,
            isEmpty: content.modes.every((mode) =>
              mode.colors.every((color) => color.value.trim() === ""),
            ),
          })
        }
        pending={pending}
      />
    );
  }

  if (element.type === "stylesheet_set" && "stylesheets" in element.content) {
    return (
      <StylesheetSetEditor
        content={element.content}
        onChange={(content) =>
          onChange({
            ...element,
            content,
            isEmpty:
              content.global.trim() === "" &&
              (content.stylesheets ?? []).every(
                (sheet) => sheet.css.trim() === "",
              ) &&
              (content.assets ?? []).length === 0,
          })
        }
        pending={pending}
      />
    );
  }

  if (element.type === "variable_schema" && "variables" in element.content) {
    return (
      <VariableSchemaEditor
        chosen={chosen}
        onChange={(variables) =>
          onChange({
            ...element,
            content: { variables },
            isEmpty: variables.length === 0,
          })
        }
        onChoose={onChoose}
        pending={pending}
        variables={element.content.variables}
      />
    );
  }

  if (element.type === "script_list" && "scripts" in element.content) {
    return (
      <ScriptListEditor
        chosen={chosen}
        onChange={(scripts) =>
          onChange({
            ...element,
            content: { scripts },
            isEmpty: scripts.length === 0,
          })
        }
        onChoose={onChoose}
        pending={pending}
        scripts={element.content.scripts}
      />
    );
  }

  return null;
}

function ImageEditor({
  assetId,
  images,
  isGallery,
  items,
  mediaRole,
  onAdded,
  onChange,
  pending,
}: {
  assetId: string;
  images: AssetImage[];
  isGallery: boolean;
  items: ImageItem[];
  mediaRole: "expression" | "gallery";
  onAdded: () => void;
  onChange: (items: ImageItem[]) => void;
  pending: boolean;
}) {
  const candidate = useWorkingCopy();
  const [uploading, setUploading] = useState(false);
  const [message, setMessage] = useState("");
  const [previews, setPreviews] = useState<Record<string, string>>({});
  const file = useRef<HTMLInputElement>(null);
  const imagesById = useMemo(
    () => new Map(images.map((image) => [image.id, image])),
    [images],
  );

  async function upload(chosen: File | null) {
    if (!chosen || uploading) return;
    setUploading(true);
    setMessage("");
    try {
      const mediaId = await addAssetImage(
        candidate,
        assetId,
        chosen,
        mediaRole,
      );
      setPreviews((current) => ({
        ...current,
        [mediaId]: URL.createObjectURL(chosen),
      }));
      onChange([...items, { mediaId }]);
      onAdded();
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "The image could not be added. Try again.",
      );
    } finally {
      setUploading(false);
      if (file.current) file.current.value = "";
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {items.length === 0 ? (
        <Note>No images are in this block yet.</Note>
      ) : (
        <ol className="flex flex-col gap-7">
          {items.map((item, index) => {
            const stored = imagesById.get(item.mediaId);
            const source = previews[item.mediaId] ?? stored?.thumbUrl;
            return (
              <li key={item.mediaId}>
                <InlineItem
                  moves={movesFor(items, index, onChange)}
                  name={item.name?.trim() || `Image ${index + 1}`}
                  onRemove={() => onChange(without(items, index))}
                  pending={pending}
                  removeLabel="Remove image"
                >
                  <div className="flex flex-wrap items-start gap-4">
                    <div className="grid size-24 shrink-0 place-items-center overflow-hidden rounded-plate bg-media">
                      {source ? (
                        <Image
                          alt=""
                          height={stored?.height ?? 200}
                          sizes="120px"
                          src={source}
                          unoptimized
                          width={stored?.width ?? 200}
                        />
                      ) : (
                        <span className="text-label text-on-media">
                          Missing
                        </span>
                      )}
                    </div>
                    <div className="flex min-w-0 flex-1 flex-col gap-4">
                      <Field hint="optional" label="Name">
                        <TextField
                          disabled={pending}
                          onChange={(event) =>
                            onChange(
                              replaceAt(items, index, {
                                name: event.target.value || undefined,
                              }),
                            )
                          }
                          value={item.name ?? ""}
                        />
                      </Field>
                      {isGallery ? (
                        <Switch
                          checked={item.omitFromDownloads !== true}
                          hint="Readers can change this for their own copy."
                          label="Include in downloads"
                          onChange={(included) =>
                            onChange(
                              replaceAt(items, index, {
                                omitFromDownloads: included ? undefined : true,
                              }),
                            )
                          }
                          pending={pending}
                        />
                      ) : null}
                    </div>
                  </div>
                </InlineItem>
              </li>
            );
          })}
        </ol>
      )}
      {message ? (
        <p className="text-meta text-stop" role="alert">
          {message}
        </p>
      ) : null}
      <label className="inline-flex min-h-11 cursor-pointer items-center gap-2 self-start rounded-control bg-deep px-4 text-meta font-medium text-ink hover:bg-rule/45 has-disabled:opacity-45">
        <ImagePlus aria-hidden="true" size={16} />
        {uploading ? "Adding…" : "Add image"}
        <input
          accept="image/*"
          className="sr-only"
          disabled={pending || uploading}
          onChange={(event) => void upload(event.target.files?.[0] ?? null)}
          ref={file}
          type="file"
        />
      </label>
    </div>
  );
}

function movesFor<T>(
  items: T[],
  index: number,
  onChange: (items: T[]) => void,
) {
  return {
    onEarlier: () => onChange(moveItem(items, index, index - 1)),
    onLater: () => onChange(moveItem(items, index, index + 1)),
    position: index,
    total: items.length,
  };
}

export function elementHint(type: AssetElement["type"]): string {
  switch (type) {
    case "prose":
      return "Write this at full width; it will keep the page’s reading layout.";
    case "text_set":
      return "Each greeting stays in this one ordered collection.";
    case "dialogue_sample":
      return "Keep each speaker and message together, in reading order.";
    case "image_set":
      return "Images sit in the order you put them in, and each may carry a name.";
    case "field_list":
      return "Each row is a short name and the value beside it.";
    case "link_list":
      return "Addresses have to start with http or https.";
    case "entry_table":
      return "Each entry is switched on by its own keys.";
    case "prompt_list":
      return "Fragments are sent in the order they sit in, under the headings you give them.";
    case "setting_group":
      return "The names are your app's own, and a setting you leave out stays out of the file.";
    case "color_set":
      return "These are the colours readers see first. Keep the app's names so the theme still knows where each colour belongs.";
    case "stylesheet_set":
      return "The main sheet, component sheets, and their fonts travel together with the theme.";
    case "variable_schema":
      return "Each variable is one thing a reader chooses before the preset runs.";
    case "script_list":
      return "Each script finds something and writes something else in its place.";
    case "record_list":
      return "Each Lumia keeps its identity, writing, and avatar together in Pack order.";
  }
}
