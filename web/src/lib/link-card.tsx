import { ImageResponse } from "next/og";
import type { CSSProperties } from "react";
import { CARD_SIZE, mediaUrl } from "@/lib/site-metadata";

type CardSubject = {
  title: string;
  image?: CardImage | null;
  eyebrow?: string;
  byline: string;
  description?: string;
  footer?: string;
  avatar?: CardImage;
  initials?: string;
};

type CardImage = { url: string; width: number; height: number };

const assetUrl = `http://127.0.0.1:${process.env.PORT ?? "3000"}`;

let resources: ReturnType<typeof loadResources> | undefined;

function loadResources() {
  return Promise.all([
    Promise.all(
      (
        [
          { name: "Outfit", file: "Outfit-SemiBold.ttf", weight: 600 },
          { name: "DM Sans", file: "DMSans-Medium.ttf", weight: 500 },
        ] as const
      ).map(async (face) => ({
        name: face.name,
        data: await fetchAsset(`/fonts/${face.file}`).then((r) =>
          r.arrayBuffer(),
        ),
        weight: face.weight,
        style: "normal" as const,
      })),
    ),
    fetchAsset("/brand/illarin-horizontal-white.svg")
      .then((r) => r.text())
      .then((svg) => `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`),
    fetchAsset("/brand/link-card-fallback.png").then((r) =>
      imageDataUrl(r, "image/png"),
    ),
  ]);
}

async function fetchAsset(path: string): Promise<Response> {
  const response = await fetch(new URL(path, assetUrl));
  if (!response.ok) throw new Error(`Could not load card asset ${path}`);
  return response;
}

const clipped: CSSProperties = {
  display: "block",
  overflow: "hidden",
  textOverflow: "ellipsis",
  wordBreak: "break-word",
  flexShrink: 0,
};

export async function renderLinkCard(subject: CardSubject): Promise<Response> {
  if (!resources) {
    resources = loadResources().catch((error) => {
      resources = undefined;
      throw error;
    });
  }
  const [[fonts, logo, fallback], image, avatar] = await Promise.all([
    resources,
    drawable(subject.image?.url),
    drawable(subject.avatar?.url),
  ]);
  const supplied = image ?? avatar;
  const picture = image ? subject.image : subject.avatar;
  const portrait = supplied && picture && picture.width / picture.height < 1.3;
  const profile = subject.initials !== undefined;
  return new ImageResponse(
    <div
      style={{
        display: "flex",
        width: "100%",
        height: "100%",
        background: "#0a0a0a",
        color: "#fff",
        fontFamily: "DM Sans",
        position: "relative",
      }}
    >
      <img
        alt=""
        src={supplied ?? fallback}
        width={portrait ? 720 : 1200}
        height={630}
        style={{
          position: "absolute",
          right: 0,
          top: 0,
          width: portrait ? 720 : "100%",
          height: "100%",
          objectFit: portrait ? "contain" : "cover",
        }}
      />
      <div
        style={{
          position: "absolute",
          top: 0,
          left: 0,
          width: "100%",
          height: "100%",
          backgroundImage:
            "linear-gradient(90deg, rgba(6,6,8,0.97) 0%, rgba(6,6,8,0.90) 30%, rgba(6,6,8,0.55) 52%, rgba(6,6,8,0.04) 80%)",
        }}
      />
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          width: 700,
          height: "100%",
          padding: "44px 0 40px 48px",
        }}
      >
        <img alt="Illarin" src={logo} width={228} height={76} />
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            marginTop: profile ? 22 : 64,
            width: 620,
          }}
        >
          {profile ? (
            <div
              style={{
                display: "flex",
                width: 130,
                height: 130,
                borderRadius: "50%",
                border: "3px solid #b89aff",
                background: "#202022",
                overflow: "hidden",
                alignItems: "center",
                justifyContent: "center",
                marginBottom: 16,
                fontSize: 48,
                fontFamily: "Outfit",
              }}
            >
              {avatar ? (
                <img
                  alt=""
                  src={avatar}
                  width={124}
                  height={124}
                  style={{ objectFit: "cover" }}
                />
              ) : (
                subject.initials
              )}
            </div>
          ) : (
            <div style={{ fontSize: 25, color: "#b89aff", marginBottom: 14 }}>
              {subject.eyebrow}
            </div>
          )}
          <div
            style={{
              ...clipped,
              fontFamily: "Outfit",
              fontWeight: 600,
              fontSize: profile ? 44 : subject.title.length > 90 ? 46 : 52,
              lineHeight: 1.1,
              letterSpacing: "-0.02em",
              lineClamp: profile ? 2 : 3,
            }}
          >
            {subject.title}
          </div>
          <div
            style={{
              ...clipped,
              lineClamp: 1,
              fontSize: 26,
              color: "#b89aff",
              marginTop: 12,
            }}
          >
            {subject.byline}
          </div>
          {subject.description ? (
            <div
              style={{
                ...clipped,
                lineClamp: 2,
                fontSize: 25,
                lineHeight: 1.3,
                marginTop: 20,
                color: "#ededf0",
              }}
            >
              {subject.description}
            </div>
          ) : null}
        </div>
        {subject.footer ? (
          <div
            style={{
              ...clipped,
              lineClamp: 1,
              marginTop: "auto",
              paddingTop: 16,
              fontSize: 23,
              color: "#dedee3",
            }}
          >
            {subject.footer}
          </div>
        ) : null}
      </div>
      {!supplied ? (
        <div
          style={{
            position: "absolute",
            bottom: 25,
            right: 30,
            fontSize: 17,
            color: "#dedee3",
          }}
        >
          Illarin artwork
        </div>
      ) : null}
    </div>,
    {
      ...CARD_SIZE,
      fonts,
      headers: {
        "cache-control": "public, max-age=300",
        "content-type": "image/png",
      },
    },
  );
}

async function drawable(address?: string | null): Promise<string | null> {
  if (!address) return null;
  try {
    const url = new URL(address, mediaUrl);
    if (
      url.origin !== new URL(mediaUrl).origin ||
      !url.pathname.startsWith("/media/")
    )
      return null;
    const response = await fetch(url, {
      redirect: "error",
      signal: AbortSignal.timeout(5000),
    });
    const type = response.headers.get("content-type") ?? "";
    if (!response.ok || !type.startsWith("image/")) return null;
    return imageDataUrl(response, type);
  } catch {
    return null;
  }
}

async function imageDataUrl(response: Response, type: string): Promise<string> {
  const bytes = new Uint8Array(await response.arrayBuffer());
  let binary = "";
  for (let i = 0; i < bytes.length; i += 8192) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 8192));
  }
  return `data:${type};base64,${btoa(binary)}`;
}
