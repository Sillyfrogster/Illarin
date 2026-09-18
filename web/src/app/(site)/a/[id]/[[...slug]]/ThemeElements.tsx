import { CopyButton } from "@/components/ui/copy-button";
import type { ColorSetContent, StylesheetSetContent } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { themeColorName } from "@/lib/theme-colors";
import { CODE } from "./element-runs";

export function ThemePalette({
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

export function ThemeStyles({
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
          {(content.assets ?? []).map((work) => work.path).join(", ")}
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
