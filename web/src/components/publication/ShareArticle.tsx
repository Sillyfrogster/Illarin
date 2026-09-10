"use client";

import { Check, Link2, Share2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  hasNativeShare,
  type ShareState,
  shareReport,
} from "@/lib/article-share";

const REPORT_LINGERS = 4000;

const ACTION =
  "inline-flex min-h-11 items-center gap-2 rounded-control px-3 text-ui text-mute transition-colors hover:bg-deep hover:text-ink";

export function ShareArticle({
  permalink,
  title,
}: {
  permalink: string;
  title: string;
}) {
  const [state, setState] = useState<ShareState>("ready");
  const [sheet, setSheet] = useState(false);
  const address = useRef<HTMLInputElement>(null);

  useEffect(() => setSheet(hasNativeShare(navigator)), []);

  useEffect(() => {
    if (state === "refused") address.current?.select();
    if (state !== "copied") return;
    const timer = setTimeout(() => setState("ready"), REPORT_LINGERS);
    return () => clearTimeout(timer);
  }, [state]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(permalink);
      setState("copied");
    } catch {
      setState("refused");
    }
  }

  async function hand() {
    try {
      await navigator.share({ title, url: permalink });
    } catch (trouble) {
      if (trouble instanceof DOMException && trouble.name === "AbortError") {
        return;
      }
      await copy();
    }
  }

  const report = shareReport(state);
  return (
    <div className="-ml-3 grid justify-items-start gap-1">
      <button className={ACTION} onClick={copy} type="button">
        {state === "copied" ? (
          <Check aria-hidden="true" className="size-4 text-accent" />
        ) : (
          <Link2 aria-hidden="true" className="size-4" />
        )}
        Copy link
      </button>
      {sheet ? (
        <button className={ACTION} onClick={hand} type="button">
          <Share2 aria-hidden="true" className="size-4" />
          Share
        </button>
      ) : null}
      <output className="ml-3 block text-meta leading-5 text-mute text-pretty empty:hidden">
        {report.said}
      </output>
      {report.reveal ? (
        <input
          aria-label="This article's link"
          className="ml-3 h-10 w-full rounded-control bg-deep px-3 font-mono text-[13px] text-ink"
          readOnly
          ref={address}
          value={permalink}
        />
      ) : null}
    </div>
  );
}
