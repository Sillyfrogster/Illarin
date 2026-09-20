import type { ReactElement } from "react";
import { MARK_PATH } from "@/components/brand/BrandMark";
import { CARD_SIZE } from "@/lib/blog-metadata";

export type CardSubject = {
  title: string;
  plate: string | null;
};

const FIELD = "#0a0a0a";

const INK = "#ffffff";

const PLATE_WIDTH = 480;

const TITLE_STEPS = [
  { upTo: 52, size: 74 },
  { upTo: 92, size: 58 },
  { upTo: Number.POSITIVE_INFINITY, size: 46 },
];

export function cardTitleSize(title: string): number {
  const step = TITLE_STEPS.find(({ upTo }) => title.length <= upTo);
  return (step ?? TITLE_STEPS[TITLE_STEPS.length - 1]).size;
}

export function markImage(fill: string): string {
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 144 144">` +
    `<g fill="${fill}" transform="translate(16 8)"><path d="${MARK_PATH}"/>` +
    `<path d="${MARK_PATH}" transform="translate(112 0) scale(-1 1)"/></g></svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

export function PostCard({
  title,
  plate,
  logo,
}: CardSubject & { logo: string }): ReactElement {
  return (
    <div
      style={{
        backgroundColor: FIELD,
        color: INK,
        display: "flex",
        fontFamily: "DM Sans",
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
          <img alt="Illarin" height={57} src={logo} width={170} />
          <span
            style={{
              fontSize: 27,
              fontWeight: 500,
              opacity: 0.72,
            }}
          >
            Blog
          </span>
        </div>

        <div style={{ alignItems: "center", display: "flex", flex: 1 }}>
          <div
            style={{
              display: "flex",
              fontFamily: "Outfit",
              fontSize: cardTitleSize(title),
              fontWeight: 600,
              letterSpacing: "-0.022em",
              lineHeight: 1.16,
              paddingRight: 28,
              wordBreak: "break-word",
            }}
          >
            {title}
          </div>
        </div>
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
          backgroundImage: `linear-gradient(to right, ${FIELD}, rgba(10, 10, 10, 0.35) 62%, rgba(10, 10, 10, 0))`,
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
      height={34}
      src={markImage("#b89aff")}
      style={{ opacity: 0.55, position: "absolute", right: 76, top: 62 }}
      width={34}
    />
  );
}
