import Image from "next/image";
import type { PostMedia } from "@/lib/api/query";

export type Header = { mediaId: string; alt: string; caption?: string };

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
    <figure className="mt-group">
      <span className="block overflow-hidden rounded-plate bg-deep">
        <Image
          alt={header.alt}
          className="mx-auto h-auto max-h-[30rem] w-full object-contain"
          height={picture.height}
          priority
          src={picture.url}
          unoptimized
          width={picture.width}
        />
      </span>
      {header.caption ? (
        <figcaption className="mt-3 max-w-[70ch] font-prose text-meta leading-6 text-mute">
          {header.caption}
        </figcaption>
      ) : null}
    </figure>
  );
}
