"use client";

import { Eye, EyeOff } from "lucide-react";
import Image from "next/image";
import { useEffect, useState } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import { Button } from "@/components/ui/button";
import { ImageZoom } from "@/components/ui/image-zoom";
import type { BrowseType, NsfwPreference, WorkImage } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import {
  readSessionPreference,
  readWorkReveal,
  writeWorkReveal,
} from "@/lib/nsfw-preference";
import { CoverControl } from "./CoverControl";

interface WorkMediaProps {
  id: string;
  media: WorkImage[];
  type: BrowseType;
  name: string;
  isNsfw: boolean | null;
  preference: NsfwPreference;
  coverInFile: boolean;
  writing: boolean;
}

/** The roles the header shows, leaving a gallery in the creator's own block. */
const COVER_ROLES = new Set(["avatar", "avatar_alt"]);

export function coverMedia(media: WorkImage[]): WorkImage[] {
  return media.filter((image) => COVER_ROLES.has(image.role));
}

function clearVariant(url: string) {
  return url.replace("_blurred/", "/");
}

export function WorkMedia({
  id,
  media,
  type,
  name,
  isNsfw,
  preference,
  coverInFile,
  writing,
}: WorkMediaProps) {
  const { account } = useAuth();
  const presentationMedia = coverMedia(media);
  const recorded = presentationMedia.findIndex((image) => image.isCover);
  const [chosen, setChosen] = useState<number | null>(null);
  const here = chosen ?? (recorded >= 0 ? recorded : null);
  const [failed, setFailed] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const [signedOutPreference, setSignedOutPreference] =
    useState<NsfwPreference>();
  const shown = here === null ? undefined : presentationMedia[here];
  const useFallback = failed || !shown;
  const showClear =
    preference === "shown" || signedOutPreference === "shown" || revealed;
  const canReveal = isNsfw === true && !showClear && !useFallback;

  useEffect(() => {
    if (account === undefined) return;
    if (account) {
      setSignedOutPreference(undefined);
      return;
    }
    setSignedOutPreference(readSessionPreference());
  }, [account]);

  useEffect(() => {
    if (!canReveal) return;
    setRevealed(readWorkReveal(id));
  }, [canReveal, id]);

  const source = shown
    ? showClear
      ? clearVariant(shown.detailUrl)
      : shown.detailUrl
    : "";

  return (
    <div className="w-full">
      <div className="relative">
        {useFallback ? (
          <div className="relative aspect-3/4 w-full overflow-hidden rounded-art">
            <DefaultCover type={type} />
          </div>
        ) : (
          <ImageZoom
            alt={name}
            className="rounded-art"
            detailSrc={source}
            revealOnHover
          >
            <Image
              alt={name}
              className="h-auto w-full rounded-art bg-media"
              height={shown.height}
              onError={() => setFailed(true)}
              priority
              sizes="(max-width: 767px) 88vw, 520px"
              src={source}
              unoptimized
              width={shown.width}
            />
          </ImageZoom>
        )}
        {isNsfw === true ? (
          <p className="absolute top-3 left-3 inline-flex items-center gap-1.5 rounded-control bg-media/85 px-2.5 py-1 text-label font-medium text-on-media">
            {showClear ? (
              <Eye aria-hidden="true" className="size-3.5" />
            ) : (
              <EyeOff aria-hidden="true" className="size-3.5" />
            )}
            {showClear ? "Adult" : "Adult · blurred"}
          </p>
        ) : null}
        {canReveal && !revealed ? (
          <Button
            className="absolute right-4 bottom-4 bg-field shadow-cover"
            onClick={() => {
              setRevealed(true);
              writeWorkReveal(id);
            }}
          >
            <Eye aria-hidden="true" />
            Show images
          </Button>
        ) : null}
      </div>

      {presentationMedia.length > 1 ? (
        <ul className="mt-2 grid list-none grid-cols-5 gap-1.5">
          {presentationMedia.map((image, index) => (
            <li key={image.id}>
              <button
                aria-current={index === here}
                aria-label={`Image ${index + 1} of ${presentationMedia.length}`}
                className={cn(
                  "block aspect-square w-full overflow-hidden rounded-chip bg-media outline-offset-3 transition-transform duration-160 motion-reduce:transition-none",
                  index === here
                    ? "inset-ring-2 inset-ring-accent"
                    : "opacity-70 hover:-translate-y-0.5 hover:opacity-100",
                )}
                onClick={() => {
                  setChosen(index);
                  setFailed(false);
                }}
                type="button"
              >
                <Image
                  alt=""
                  className="size-full object-cover"
                  height={image.height}
                  sizes="80px"
                  src={
                    showClear ? clearVariant(image.thumbUrl) : image.thumbUrl
                  }
                  unoptimized
                  width={image.width}
                />
              </button>
            </li>
          ))}
        </ul>
      ) : null}

      {writing ? (
        <CoverControl
          inFile={coverInFile}
          preview={useFallback ? undefined : source}
          workId={id}
        />
      ) : null}
    </div>
  );
}
