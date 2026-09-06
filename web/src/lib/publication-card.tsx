import type { ReactElement } from "react";
import { MARK_BLADE, MARK_CORE } from "@/components/brand/BrandMark";
import { CARD_SIZE } from "@/lib/publication-metadata";

/** What the card draws: an optional plate on the right and an optional app under the title. */
export type CardSubject = {
  title: string;
  app: { name: string; mark: string | null } | null;
  plate: string | null;
};

const FIELD = "#050505";

const INK = "#f5f5f2";

const PLATE_WIDTH = 480;

/** How long a title runs before it needs a smaller size to stay inside the card. */
const TITLE_STEPS = [
  { upTo: 52, size: 74 },
  { upTo: 92, size: 58 },
  { upTo: Number.POSITIVE_INFINITY, size: 46 },
];

/** The size a title is set at, chosen so the longest one Illarin accepts still fits. */
export function cardTitleSize(title: string): number {
  const step = TITLE_STEPS.find(({ upTo }) => title.length <= upTo);
  return (step ?? TITLE_STEPS[TITLE_STEPS.length - 1]).size;
}

/** The Illarin mark as bytes a server-side renderer can draw without a stylesheet. */
export function markImage(fill: string): string {
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">` +
    `<path d="${MARK_BLADE}" fill="${fill}" fill-rule="evenodd"/>` +
    `<path d="${MARK_CORE}" fill="${fill}"/></svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

/** The card a link preview shows for one post. */
export function PostCard({ title, app, plate }: CardSubject): ReactElement {
  return (
    <div
      style={{
        backgroundColor: FIELD,
        color: INK,
        display: "flex",
        fontFamily: "Manrope",
        height: "100%",
        position: "relative",
        width: "100%",
      }}
    >
      {plate ? <Plate source={plate} /> : <CornerMark />}
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          height: "100%",
          padding: "62px 0 68px 76px",
          position: "relative",
          width: plate ? CARD_SIZE.width - PLATE_WIDTH : 940,
        }}
      >
        <div style={{ alignItems: "center", display: "flex", gap: 15 }}>
          <img alt="" height={28} src={markImage(INK)} width={28} />
          <span
            style={{
              fontSize: 19,
              fontWeight: 700,
              letterSpacing: "0.15em",
              opacity: 0.72,
            }}
          >
            ILLARIN BLOG
          </span>
        </div>

        <div style={{ alignItems: "center", display: "flex", flex: 1 }}>
          <div
            style={{
              display: "flex",
              fontFamily: "Bodoni Moda",
              fontSize: cardTitleSize(title),
              fontWeight: 500,
              letterSpacing: "-0.022em",
              lineHeight: 1.14,
              paddingRight: 28,
            }}
          >
            {title}
          </div>
        </div>

        {app ? (
          <div style={{ alignItems: "center", display: "flex", gap: 14 }}>
            {app.mark ? (
              <img alt="" height={30} src={app.mark} width={30} />
            ) : null}
            <span style={{ fontSize: 25, fontWeight: 500, opacity: 0.6 }}>
              {app.name}
            </span>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function Plate({ source }: { source: string }): ReactElement {
  return (
    <div
      style={{
        display: "flex",
        height: "100%",
        position: "absolute",
        right: 0,
        top: 0,
        width: PLATE_WIDTH,
      }}
    >
      <img
        alt=""
        height={CARD_SIZE.height}
        src={source}
        style={{ height: "100%", objectFit: "cover", width: "100%" }}
        width={PLATE_WIDTH}
      />
      <div
        style={{
          backgroundImage: `linear-gradient(to right, ${FIELD}, rgba(5, 5, 5, 0.35) 62%, rgba(5, 5, 5, 0))`,
          height: "100%",
          left: 0,
          position: "absolute",
          top: 0,
          width: 260,
        }}
      />
    </div>
  );
}

function CornerMark(): ReactElement {
  return (
    <img
      alt=""
      height={30}
      src={markImage(INK)}
      style={{ opacity: 0.5, position: "absolute", right: 76, top: 61 }}
      width={30}
    />
  );
}
