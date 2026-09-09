import { ArrowUpRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import { Reveal } from "./LandingMotion";

export function InvitationChapter() {
  return (
    <section
      id="create"
      aria-labelledby="landing-create-title"
      className="relative isolate overflow-hidden bg-[#09080d] text-[#ffffff]"
    >
      <Image
        src="/landing/tide.webp"
        alt=""
        fill
        sizes="100vw"
        className="object-cover object-[60%_center]"
      />
      <div
        aria-hidden="true"
        className="absolute inset-0 bg-[linear-gradient(90deg,#09080deb_0%,#09080d99_45%,#09080d22_100%),linear-gradient(0deg,#09080dbb,transparent_60%)]"
      />
      <Shell className="relative py-24 sm:py-36 lg:py-44">
        <Reveal className="max-w-[760px]">
          <p className="mb-6 font-ui text-ui text-[#d3baff]">
            Now, it’s your turn.
          </p>
          <h2
            id="landing-create-title"
            className="font-display text-[clamp(3.3rem,6.8vw,7rem)] leading-[1.02] font-medium tracking-[-.045em]"
          >
            Make something
            <br />
            only you could.
          </h2>
          <p className="mt-7 max-w-[400px] text-lede text-[#ffffff]/85">
            That character in your head.
            <br />
            That world you keep coming back to.
            <br />
            There’s room for it here.
          </p>
          <div className="mt-9 flex flex-wrap gap-3">
            <Button
              asChild
              variant="primary"
              size="large"
              className="rounded-full px-6"
            >
              <Link href="/upload">
                Create on Illarin <ArrowUpRight aria-hidden="true" />
              </Link>
            </Button>
            <Button
              asChild
              variant="ghost"
              size="large"
              className="rounded-full text-[#ffffff] hover:bg-[#ffffff]/10 hover:text-[#ffffff]"
            >
              <Link href="/browse">Keep exploring</Link>
            </Button>
          </div>
          <p className="mt-10 max-w-[390px] text-meta leading-relaxed text-[#ffffff]/75">
            Start with a file or a new creation. Arrange your content, keep your
            draft private, and publish when you’re ready.
          </p>
        </Reveal>
      </Shell>
    </section>
  );
}
