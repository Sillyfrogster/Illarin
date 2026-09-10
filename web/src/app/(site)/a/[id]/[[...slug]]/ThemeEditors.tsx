"use client";

import { FilePlus2 } from "lucide-react";
import { useRef, useState } from "react";
import type {
  ColorSetContent,
  StylesheetSetContent,
  ThemeColor,
  ThemeColorMode,
  ThemeFile,
  ThemeStylesheet,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { pickerColor } from "@/lib/theme-colors";
import { moveItem, replaceAt, without } from "./workspace/collection";
import {
  AddAction,
  Field,
  FieldGroup,
  ItemMoveActions,
  Note,
  RemoveAction,
  Switch,
  TextAreaField,
  TextField,
} from "./workspace/fields";

export function ColorSetEditor({
  content,
  onChange,
  pending,
}: {
  content: ColorSetContent;
  onChange: (content: ColorSetContent) => void;
  pending: boolean;
}) {
  function changeMode(index: number, mode: ThemeColorMode) {
    onChange({ modes: replaceAt(content.modes, index, mode) });
  }

  return (
    <div className="flex flex-col gap-8">
      {content.modes.map((mode, modeIndex) => (
        <FieldGroup
          key={mode.name || modeIndex}
          legend={mode.name?.trim() || "Default mode"}
        >
          <Field
            hint="what the app calls this set of colours"
            label="Mode name"
          >
            <TextField
              disabled={pending}
              onChange={(event) =>
                changeMode(modeIndex, {
                  ...mode,
                  name: event.target.value || undefined,
                })
              }
              placeholder="Default"
              value={mode.name ?? ""}
            />
          </Field>
          <ColorRows
            colors={mode.colors}
            onChange={(colors) => changeMode(modeIndex, { ...mode, colors })}
            pending={pending}
          />
          {content.modes.length > 1 ? (
            <RemoveAction
              disabled={pending}
              onClick={() =>
                onChange({ modes: without(content.modes, modeIndex) })
              }
            >
              Remove mode
            </RemoveAction>
          ) : null}
        </FieldGroup>
      ))}
      <AddAction
        disabled={pending}
        onClick={() =>
          onChange({
            modes: [
              ...content.modes,
              { colors: [], name: `Mode ${content.modes.length + 1}` },
            ],
          })
        }
      >
        Add mode
      </AddAction>
    </div>
  );
}

function ColorRows({
  colors,
  onChange,
  pending,
}: {
  colors: ThemeColor[];
  onChange: (colors: ThemeColor[]) => void;
  pending: boolean;
}) {
  return (
    <div className="flex flex-col gap-3">
      {colors.length === 0 ? (
        <Note>This mode names no colours yet.</Note>
      ) : null}
      {colors.map((color, index) => (
        <div className="flex flex-wrap items-end gap-2" key={color.id ?? index}>
          <label
            className="relative grid size-11 shrink-0 place-items-center overflow-hidden rounded-control focus-within:outline focus-within:outline-2 focus-within:outline-accent focus-within:outline-offset-2"
            style={{ backgroundColor: color.value }}
          >
            <span className="sr-only">
              Choose {color.name || `colour ${index + 1}`}
            </span>
            <input
              className="size-11 cursor-pointer opacity-0"
              disabled={pending}
              onChange={(event) =>
                onChange(
                  replaceAt(colors, index, { value: event.target.value }),
                )
              }
              type="color"
              value={pickerColor(color.value)}
            />
          </label>
          <TextField
            aria-label={`Name for colour ${index + 1}`}
            className="min-w-28 flex-1"
            disabled={pending}
            onChange={(event) =>
              onChange(replaceAt(colors, index, { name: event.target.value }))
            }
            placeholder="Name"
            value={color.name}
          />
          <TextField
            aria-label={`Value for ${color.name || `colour ${index + 1}`}`}
            className="min-w-36 flex-1 font-mono"
            disabled={pending}
            onChange={(event) =>
              onChange(replaceAt(colors, index, { value: event.target.value }))
            }
            placeholder="#7c5cff"
            value={color.value}
          />
          <RemoveAction
            disabled={pending}
            label={`Remove ${color.name || `colour ${index + 1}`}`}
            onClick={() => onChange(without(colors, index))}
          />
        </div>
      ))}
      <AddAction
        disabled={pending}
        onClick={() =>
          onChange([
            ...colors,
            { id: crypto.randomUUID(), name: "colour", value: "#7c5cff" },
          ])
        }
      >
        Add colour
      </AddAction>
    </div>
  );
}

export function StylesheetSetEditor({
  content,
  onChange,
  pending,
}: {
  content: StylesheetSetContent;
  onChange: (content: StylesheetSetContent) => void;
  pending: boolean;
}) {
  const [message, setMessage] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);
  const sheets = content.stylesheets ?? [];
  const files = content.assets ?? [];

  async function addFiles(chosen: FileList | null) {
    if (!chosen?.length) return;
    setMessage("");
    try {
      const attached = await Promise.all(Array.from(chosen, themeFile));
      onChange({ ...content, assets: [...files, ...attached] });
    } catch {
      setMessage("Those files could not be attached. Try them again.");
    } finally {
      if (fileInput.current) fileInput.current.value = "";
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <Field label="Main stylesheet">
        <TextAreaField
          className="font-mono text-meta"
          disabled={pending}
          onChange={(event) =>
            onChange({ ...content, global: event.target.value })
          }
          rows={14}
          spellCheck={false}
          value={content.global}
        />
      </Field>

      <FieldGroup legend="Component stylesheets">
        {sheets.length === 0 ? (
          <Note>
            The main sheet carries the theme until you split part of it out.
          </Note>
        ) : null}
        {sheets.map((sheet, index) => (
          <ComponentSheet
            key={sheet.id ?? index}
            moves={{
              onEarlier: () =>
                onChange({
                  ...content,
                  stylesheets: moveItem(sheets, index, index - 1),
                }),
              onLater: () =>
                onChange({
                  ...content,
                  stylesheets: moveItem(sheets, index, index + 1),
                }),
              position: index,
              total: sheets.length,
            }}
            onChange={(changes) =>
              onChange({
                ...content,
                stylesheets: replaceAt(sheets, index, changes),
              })
            }
            onRemove={() =>
              onChange({ ...content, stylesheets: without(sheets, index) })
            }
            pending={pending}
            position={index}
            sheet={sheet}
          />
        ))}
        <AddAction
          disabled={pending}
          onClick={() =>
            onChange({
              ...content,
              stylesheets: [
                ...sheets,
                {
                  css: "",
                  enabled: true,
                  id: crypto.randomUUID(),
                  name: `Component ${sheets.length + 1}`,
                },
              ],
            })
          }
        >
          Add component stylesheet
        </AddAction>
      </FieldGroup>

      <FieldGroup legend="Theme files">
        <Note>
          Fonts and other files referenced by the CSS stay inside the theme
          bundle.
        </Note>
        {files.length > 0 ? (
          <ul className="flex flex-col gap-2">
            {files.map((file, index) => (
              <li
                className="flex flex-wrap items-center gap-x-3 gap-y-1"
                key={file.id ?? index}
              >
                <span className="min-w-0 flex-1 text-ui text-ink wrap-anywhere">
                  {file.path}
                </span>
                <span className="text-label text-mute">
                  {file.mediaType || "Attached file"}
                </span>
                <RemoveAction
                  disabled={pending}
                  label={`Remove ${file.path}`}
                  onClick={() =>
                    onChange({ ...content, assets: without(files, index) })
                  }
                />
              </li>
            ))}
          </ul>
        ) : (
          <Note>No files attached.</Note>
        )}
        <label className="inline-flex min-h-11 cursor-pointer items-center gap-2 self-start rounded-control bg-deep px-4 text-meta font-medium text-ink hover:bg-rule/45 has-disabled:opacity-45">
          <FilePlus2 aria-hidden="true" size={16} />
          Attach files
          <input
            accept="font/*,.woff,.woff2,.ttf,.otf"
            className="sr-only"
            disabled={pending}
            multiple
            onChange={(event) => void addFiles(event.target.files)}
            ref={fileInput}
            type="file"
          />
        </label>
        {message ? (
          <p className="text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
      </FieldGroup>
    </div>
  );
}

function ComponentSheet({
  moves,
  onChange,
  onRemove,
  pending,
  position,
  sheet,
}: {
  moves: Parameters<typeof ItemMoveActions>[0]["moves"];
  onChange: (changes: Partial<ThemeStylesheet>) => void;
  onRemove: () => void;
  pending: boolean;
  position: number;
  sheet: ThemeStylesheet;
}) {
  return (
    <div className={cn("flex flex-col gap-3", !sheet.enabled && "opacity-70")}>
      <Field label={`Component ${position + 1}`}>
        <TextField
          disabled={pending}
          onChange={(event) => onChange({ name: event.target.value })}
          value={sheet.name}
        />
      </Field>
      <Field label={`${sheet.name || `Component ${position + 1}`} CSS`}>
        <TextAreaField
          className="font-mono text-meta"
          disabled={pending}
          onChange={(event) => onChange({ css: event.target.value })}
          rows={9}
          spellCheck={false}
          value={sheet.css}
        />
      </Field>
      <Switch
        checked={sheet.enabled}
        hint="A sheet left out stays in the theme and reaches no reader."
        label="Included"
        onChange={(enabled) => onChange({ enabled })}
        pending={pending}
      />
      <div className="flex flex-wrap items-center gap-1">
        <ItemMoveActions moves={moves} pending={pending} />
        <RemoveAction disabled={pending} onClick={onRemove}>
          Remove sheet
        </RemoveAction>
      </div>
    </div>
  );
}

async function themeFile(file: File): Promise<ThemeFile> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  let binary = "";
  for (let start = 0; start < bytes.length; start += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(start, start + 0x8000));
  }
  return {
    data: btoa(binary),
    id: crypto.randomUUID(),
    mediaType: file.type || undefined,
    path: `assets/${file.name.replaceAll("\\", "-")}`,
  };
}
