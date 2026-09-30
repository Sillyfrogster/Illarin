"use client";

import { ImagePlus } from "lucide-react";
import Image from "next/image";
import { useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Sortable,
  SortableItem,
  SortableItemHandle,
} from "@/components/ui/sortable";
import { Switch } from "@/components/ui/switch";
import {
  addWorkImage,
  type WorkElement,
  type WorkImage,
} from "@/lib/api/query";
import { useDraftedChanges } from "@/lib/drafted-changes";
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
import { InlineItem, Note } from "./workspace/fields";

type ImageItem = {
  mediaId: string;
  name?: string;
  omitFromDownloads?: boolean;
};

export function ElementFields({
  workId,
  chosen,
  element,
  images,
  onChange,
  onChoose,
  onImageAdded,
  pending,
}: {
  workId: string;
  chosen: string | null;
  element: WorkElement;
  images: WorkImage[];
  onChange: (element: WorkElement) => void;
  onChoose: (key: string | null) => void;
  onImageAdded: () => void;
  pending: boolean;
}) {
  if (element.type === "image_set" && "images" in element.content) {
    return (
      <ImageEditor
        workId={workId}
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
        workId={workId}
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
  workId,
  images,
  isGallery,
  items,
  mediaRole,
  onAdded,
  onChange,
  pending,
}: {
  workId: string;
  images: WorkImage[];
  isGallery: boolean;
  items: ImageItem[];
  mediaRole: "expression" | "gallery";
  onAdded: () => void;
  onChange: (items: ImageItem[]) => void;
  pending: boolean;
}) {
  const candidate = useDraftedChanges();
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
      const mediaId = await addWorkImage(candidate, workId, chosen, mediaRole);
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
        <Note>No images yet</Note>
      ) : (
        <Sortable
          disabled={pending}
          ids={items.map((item) => item.mediaId)}
          labels={(id) => {
            const index = items.findIndex((item) => item.mediaId === id);
            return items[index]?.name?.trim() || `Image ${index + 1}`;
          }}
          onMove={(from, to) => onChange(moveItem(items, from, to))}
        >
          <ol className="m-0 flex list-none flex-col gap-7 p-0">
            {items.map((item, index) => {
              const stored = imagesById.get(item.mediaId);
              const source = previews[item.mediaId] ?? stored?.thumbUrl;
              const name = item.name?.trim() || `Image ${index + 1}`;
              return (
                <SortableItem
                  disabled={pending}
                  id={item.mediaId}
                  key={item.mediaId}
                >
                  <InlineItem
                    handle={
                      <SortableItemHandle
                        disabled={pending || items.length < 2}
                        label={`Move ${name}`}
                      />
                    }
                    name={name}
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
                          <Input
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
                            hint="Off leaves it out of downloads by default."
                            label="Include in downloads"
                            onCheckedChange={(included) =>
                              onChange(
                                replaceAt(items, index, {
                                  omitFromDownloads: included
                                    ? undefined
                                    : true,
                                }),
                              )
                            }
                            disabled={pending}
                          />
                        ) : null}
                      </div>
                    </div>
                  </InlineItem>
                </SortableItem>
              );
            })}
          </ol>
        </Sortable>
      )}
      {message ? (
        <p className="text-meta text-stop" role="alert">
          {message}
        </p>
      ) : null}
      <Button
        className="self-start"
        disabled={pending}
        loading={uploading}
        onClick={() => file.current?.click()}
      >
        {uploading ? null : <ImagePlus aria-hidden="true" />}
        {uploading ? "Adding…" : "Add image"}
      </Button>
      <input
        accept="image/*"
        hidden
        onChange={(event) => void upload(event.target.files?.[0] ?? null)}
        ref={file}
        type="file"
      />
    </div>
  );
}

export function elementHint(type: WorkElement["type"]): string {
  switch (type) {
    case "prose":
      return "Edit the text shown in this block.";
    case "text_set":
      return "Add and reorder greetings.";
    case "dialogue_sample":
      return "Keep each speaker and message together, in reading order.";
    case "image_set":
      return "Add images, then arrange them in display order.";
    case "field_list":
      return "Each row is a short name and the value beside it.";
    case "link_list":
      return "Addresses have to start with http or https.";
    case "entry_table":
      return "Add entries and choose the keys that activate them.";
    case "prompt_list":
      return "Add prompt fragments and set their order, grouping and placement.";
    case "setting_group":
      return "The names are your app's own, and a setting you leave out stays out of the file.";
    case "color_set":
      return "These are the colors readers see first. Keep the app's names so the theme still knows where each color belongs.";
    case "stylesheet_set":
      return "The main sheet, component sheets, and their fonts travel together with the theme.";
    case "variable_schema":
      return "Each variable is one thing a reader chooses before the preset runs.";
    case "script_list":
      return "Add scripts to find and replace matching text.";
    case "record_list":
      return "Each character keeps its writing and avatar together, in pack order.";
  }
}
