"use client";

import { Check, Link2, Share2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  hasNativeShare,
  type ShareState,
  shareReport,
} from "@/lib/article-share";

const REPORT_LINGERS = 4000;

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
      <Button onClick={copy} variant="ghost">
        {state === "copied" ? (
          <Check aria-hidden="true" className="text-accent" />
        ) : (
          <Link2 aria-hidden="true" />
        )}
        Copy link
      </Button>
      {sheet ? (
        <Button onClick={hand} variant="ghost">
          <Share2 aria-hidden="true" />
          Share
        </Button>
      ) : null}
      <output className="ml-3 block text-meta leading-5 text-mute text-pretty empty:hidden">
        {report.said}
      </output>
      {report.reveal ? (
        <Input
          aria-label="This article's link"
          className="ml-3 font-mono"
          readOnly
          ref={address}
          value={permalink}
        />
      ) : null}
    </div>
  );
}
