import { ArrowRight } from "lucide-react";
import Image from "next/image";
import blogDark from "@/assets/art/full/illarin-blog-masthead-dark-v1.webp";
import blogLight from "@/assets/art/full/illarin-blog-masthead-light-v1.webp";
import { siteAddress } from "@/lib/site-address";

/** The plate the blog closes on, which is also the way back to the catalog it belongs to. */
export function BackToIllarin() {
  return (
    <a
      className="group relative mt-14 flex h-40 items-center overflow-hidden rounded-plate bg-deep sm:h-44"
      href={siteAddress("/browse")}
    >
      <Image
        alt=""
        className="absolute inset-0 size-full object-cover object-[80%_center] dark:hidden"
        priority={false}
        sizes="100vw"
        src={blogLight}
      />
      <Image
        alt=""
        className="absolute inset-0 hidden size-full object-cover object-[80%_center] dark:block"
        priority={false}
        sizes="100vw"
        src={blogDark}
      />
      <span
        aria-hidden="true"
        className="absolute inset-0 bg-gradient-to-r from-deep via-deep/85 to-transparent"
      />
      <span className="relative flex items-center gap-4 pl-6 font-display text-section font-medium text-ink sm:pl-10">
        Back to the collection
        <ArrowRight
          aria-hidden="true"
          className="size-5 transition-transform duration-300 group-hover:translate-x-1 motion-reduce:transition-none"
        />
      </span>
    </a>
  );
}
