import Image from "next/image";
import type { PostMedia } from "@/lib/api/query";
import styles from "./ArticleHeader.module.css";

export type Header = { mediaId: string; alt: string; caption?: string };

/** The picture an article opens with. */
export function ArticleHeader({
  header,
  media,
}: {
  header?: Header | null;
  media: PostMedia[];
}) {
  const picture = header && media.find((one) => one.id === header.mediaId);
  if (!header || !picture) return null;
  return (
    <figure className={styles.header}>
      <Image
        alt={header.alt}
        className={styles.picture}
        height={picture.height}
        src={picture.url}
        unoptimized
        width={picture.width}
      />
      {header.caption ? (
        <figcaption className={styles.caption}>{header.caption}</figcaption>
      ) : null}
    </figure>
  );
}
