"use client";

import { ArrowRight } from "lucide-react";
import Image from "next/image";
import type { ReactNode } from "react";
import { DefaultCover } from "@/components/media/DefaultCover";
import type { WorkDetail } from "@/lib/api/query";
import { TYPE_LABELS } from "@/lib/work-types";

/** PublishSubject shows the work being published and the state it moves to. */
export function PublishSubject({
  from,
  to,
  work,
}: {
  from: string;
  to: string;
  work: WorkDetail | null;
}) {
  const cover = work?.media.find((image) => image.isCover);
  return (
    <div className="flex items-center gap-4 md:flex-col md:items-stretch md:gap-5">
      <div className="relative aspect-5/6 w-16 shrink-0 overflow-hidden rounded-control bg-media shadow-cover md:w-full md:rounded-plate">
        {cover ? (
          <Image
            alt=""
            className="size-full object-contain"
            height={cover.height}
            src={cover.detailUrl}
            unoptimized
            width={cover.width}
          />
        ) : work ? (
          <DefaultCover compact type={work.type} />
        ) : null}
      </div>
      <div className="flex min-w-0 flex-col gap-3">
        <div className="min-w-0">
          <p className="truncate font-display text-ui font-medium text-ink md:whitespace-normal md:wrap-anywhere">
            {work?.name || " "}
          </p>
          <p className="truncate font-ui text-meta text-mute">
            {work ? `${TYPE_LABELS[work.type]} by ${work.creator}` : " "}
          </p>
        </div>
        <p className="flex flex-wrap items-center gap-2 font-ui text-meta">
          <span className="text-mute">{from}</span>
          <ArrowRight aria-hidden="true" className="size-3.5 text-mute" />
          <span className="rounded-full bg-accent-wash px-2.5 py-0.5 font-medium text-accent">
            {to}
          </span>
        </p>
      </div>
    </div>
  );
}

/** Hearer is one place a publication is announced, with its switch or the way to set it up. */
export function Hearer({
  children,
  icon,
  line,
  title,
}: {
  children: ReactNode;
  icon: ReactNode;
  line: string;
  title: string;
}) {
  return (
    <label className="flex min-h-14 cursor-pointer items-center gap-3 has-[a]:cursor-default">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-control bg-deep text-ink [&_svg]:size-4.5">
        {icon}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block font-ui text-ui font-medium text-ink">
          {title}
        </span>
        <span className="block font-ui text-meta text-mute">{line}</span>
      </span>
      {children}
    </label>
  );
}

/** HearerCheck is the switch at the end of a Hearer row. */
export function HearerCheck({
  checked,
  disabled,
  onChange,
}: {
  checked: boolean;
  disabled: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <input
      checked={checked}
      className="size-5 shrink-0 cursor-pointer accent-[var(--v-action)] disabled:cursor-default"
      disabled={disabled}
      onChange={(event) => onChange(event.target.checked)}
      type="checkbox"
    />
  );
}
