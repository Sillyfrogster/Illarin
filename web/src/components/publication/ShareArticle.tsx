"use client";

import { Check, Link2, Share2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  hasNativeShare,
  type ShareState,
  shareReport,
} from "@/lib/article-share";
import styles from "./ArticleAside.module.css";

/** How long the reader is told the copy worked before the rail goes quiet again. */
const REPORT_LINGERS = 4000;

/** The two ways to hand this article to someone, neither of which asks a network to help. */
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
      requestAnimationFrame(() => address.current?.select());
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
    <div className={styles.share}>
      <p aria-hidden="true" className={styles.label}>
        Share
      </p>
      <div className={styles.actions}>
        <button className={styles.action} onClick={copy} type="button">
          {state === "copied" ? (
            <Check aria-hidden="true" size={15} strokeWidth={2.2} />
          ) : (
            <Link2 aria-hidden="true" size={15} strokeWidth={1.8} />
          )}
          Copy link
        </button>
        {sheet ? (
          <button className={styles.action} onClick={hand} type="button">
            <Share2 aria-hidden="true" size={15} strokeWidth={1.8} />
            Share
          </button>
        ) : null}
      </div>
      <output className={styles.said}>{report.said}</output>
      {report.reveal ? (
        <input
          aria-label="This article's link"
          className={styles.address}
          readOnly
          ref={address}
          value={permalink}
        />
      ) : null}
    </div>
  );
}
