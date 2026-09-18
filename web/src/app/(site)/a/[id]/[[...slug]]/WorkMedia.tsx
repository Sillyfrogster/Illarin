"use client";

import { motion, useReducedMotion } from "framer-motion";
import { Eye, EyeOff } from "lucide-react";
import Image from "next/image";
import { useEffect, useState } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
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
  typeLabel: string;
  name: string;
  isNsfw: boolean | null;
  preference: NsfwPreference;
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
  typeLabel,
  name,
  isNsfw,
  preference,
  writing,
}: WorkMediaProps) {
  const { account } = useAuth();
  const reduced = useReducedMotion();
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
    <div className="mx-auto w-full max-w-[350px]">
      <motion.div
        className="relative"
        initial={false}
        transition={{ type: "spring", stiffness: 180, damping: 22 }}
        whileHover={reduced || useFallback ? undefined : { rotate: -1, y: -5 }}
      >
        {useFallback ? (
          <div className="relative aspect-3/4 w-full overflow-hidden rounded-plate shadow-cover">
            <DefaultCover type={type} />
          </div>
        ) : (
          <ImageZoom
            alt={name}
            className="rounded-plate"
            detailSrc={source}
            revealOnHover
          >
            <Image
              alt={name}
              className="h-auto w-full rounded-plate bg-media shadow-cover"
              height={shown.height}
              onError={() => setFailed(true)}
              priority
              sizes="(max-width: 900px) 88vw, 350px"
              src={source}
              unoptimized
              width={shown.width}
            />
          </ImageZoom>
        )}
        {isNsfw === true ? (
          <p className="absolute top-3 left-3 inline-flex items-center gap-1.5 rounded-control bg-media/85 px-2.5 py-1 text-label font-medium tracking-wide text-on-media uppercase">
            {showClear ? (
              <Eye aria-hidden="true" className="size-3.5" />
            ) : (
              <EyeOff aria-hidden="true" className="size-3.5" />
            )}
            {showClear ? "Adult" : "Adult · blurred"}
          </p>
        ) : null}
        {canReveal && !revealed ? (
          <button
            className="absolute right-4 bottom-4 inline-flex min-h-11 items-center gap-2 rounded-control bg-field px-4 text-meta font-medium text-ink shadow-cover outline-offset-3"
            onClick={() => {
              setRevealed(true);
              writeWorkReveal(id);
            }}
            type="button"
          >
            <Eye aria-hidden="true" className="size-4" />
            Show images
          </button>
        ) : null}
      </motion.div>

      {presentationMedia.length > 1 ? (
        <ul className="mt-3 grid list-none grid-cols-5 gap-2">
          {presentationMedia.map((image, index) => (
            <li key={image.id}>
              <button
                aria-current={index === here}
                aria-label={`Picture ${index + 1} of ${presentationMedia.length}`}
                className={cn(
                  "block aspect-square w-full overflow-hidden rounded-control bg-media outline-offset-3 transition-transform duration-200 motion-reduce:transition-none",
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
          workId={id}
          hasCover={presentationMedia.length > 0}
          typeLabel={typeLabel}
        />
      ) : null}
    </div>
  );
}
