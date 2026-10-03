import type { CSSProperties } from "react";
import type { BrowseWork } from "@/lib/api/query";
import { workDisplayName } from "@/lib/work-name";
import { workHref } from "@/lib/work-url";
import "./home.css";

const PER_ROW = 14;

/** CoverRail is two rows of real covers drifting in opposite directions under the hero, each one a link to its work. */
export function CoverRail({ works }: { works: BrowseWork[] }) {
  if (works.length < 4) return null;
  const half = Math.ceil(works.length / 2);
  const rows = [works.slice(0, half), works.slice(half)].map((row) =>
    fill(row.length ? row : works),
  );
  return (
    <section
      aria-label="Popular works"
      className="home-rail mt-group grid gap-4 overflow-hidden py-2"
    >
      {rows.map((row, index) => (
        <div
          className="home-track"
          data-reverse={index === 1 ? "" : undefined}
          key={index === 0 ? "forward" : "back"}
          style={{ "--run": `${80 + index * 20}s` } as CSSProperties}
        >
          {[false, true].map((copy) => (
            <div
              aria-hidden={copy || undefined}
              className="flex gap-4"
              inert={copy}
              key={copy ? "copy" : "row"}
            >
              {row.map((work, at) => (
                <a
                  aria-label={workDisplayName(work.name)}
                  className="group/cover block h-[clamp(9rem,17vh,12.5rem)] shrink-0 overflow-hidden rounded-art bg-inset ring-1 ring-ink/8"
                  href={workHref(work.id, work.name)}
                  key={`${work.id}-${at}`}
                >
                  {/* biome-ignore lint/performance/noImgElement: plain covers in a CSS marquee, sized by the rail */}
                  <img
                    alt=""
                    className="aspect-3/4 h-full object-cover object-top transition-transform duration-500 ease-(--ease-wipe) group-hover/cover:scale-105"
                    decoding="async"
                    loading="lazy"
                    src={work.cover?.url}
                  />
                </a>
              ))}
            </div>
          ))}
        </div>
      ))}
    </section>
  );
}

/** fill repeats a short row until it is long enough to loop without a visible gap. */
function fill(row: BrowseWork[]) {
  const out: BrowseWork[] = [];
  while (out.length < PER_ROW) out.push(...row);
  return out.slice(0, Math.max(PER_ROW, row.length));
}
