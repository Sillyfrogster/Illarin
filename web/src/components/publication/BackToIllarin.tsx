import { ArrowRight } from "lucide-react";
import Image from "next/image";
import doorway from "@/assets/art/brand/doorway-world.png";
import { siteAddress } from "@/lib/site-address";

export function BackToIllarin() {
  return (
    <a
      className="group relative mt-14 flex h-40 items-center overflow-hidden rounded-plate bg-deep sm:h-44"
      href={siteAddress("/browse")}
    >
      <Image
        alt=""
        className="absolute inset-0 size-full object-cover object-center"
        sizes="(min-width: 1216px) 1104px, 100vw"
        src={doorway}
      />
      <span
        aria-hidden="true"
        className="absolute inset-0 bg-gradient-to-r from-deep via-deep/85 to-transparent"
      />
      <span className="relative flex items-center gap-4 pl-6 font-display text-section font-medium text-ink sm:pl-10">
        Browse
        <ArrowRight
          aria-hidden="true"
          className="size-5 transition-transform duration-300 group-hover:translate-x-1 motion-reduce:transition-none"
        />
      </span>
    </a>
  );
}
