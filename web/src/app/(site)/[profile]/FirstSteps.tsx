import { Check, ImageUp, PencilLine, Upload } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";

/** What an owner with nothing published yet can do to fill their page, each ticked off once done. */
export function FirstSteps({
  hasBanner,
  hasBiography,
  onEdit,
}: {
  hasBanner: boolean;
  hasBiography: boolean;
  onEdit: (focus: string) => void;
}) {
  return (
    <div className="rounded-plate bg-deep px-5 py-7 sm:px-8 sm:py-9">
      <h3 className="font-display text-title font-medium tracking-[-0.02em] text-ink">
        Nothing published yet
      </h3>
      <p className="mt-2 max-w-[52ch] font-prose text-prose text-mute">
        Readers who follow a link to you land here.
      </p>
      <ol className="m-0 mt-6 grid list-none gap-2 p-0">
        <Step
          action={
            <Button
              onClick={() => onEdit("profile-banner-pick")}
              size="compact"
              variant="outline"
            >
              <ImageUp aria-hidden="true" />
              Add banner
            </Button>
          }
          detail="Your page takes its color from it."
          done={hasBanner}
          label="Banner"
        />
        <Step
          action={
            <Button
              onClick={() => onEdit("profile-bio")}
              size="compact"
              variant="outline"
            >
              <PencilLine aria-hidden="true" />
              Write biography
            </Button>
          }
          detail="It sits under your name."
          done={hasBiography}
          label="Biography"
        />
        <Step
          action={
            <Button asChild size="compact" variant="primary">
              <Link href="/upload">
                <Upload aria-hidden="true" />
                Publish
              </Link>
            </Button>
          }
          detail="It shows up here and in Browse."
          done={false}
          label="First work"
        />
      </ol>
    </div>
  );
}

function Step({
  action,
  detail,
  done,
  label,
}: {
  action: ReactNode;
  detail: string;
  done: boolean;
  label: string;
}) {
  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-3 rounded-control bg-plane px-4 py-3.5">
      <span
        aria-hidden="true"
        className={cn(
          "grid size-7 shrink-0 place-items-center rounded-full",
          done ? "bg-accent text-on-accent" : "ring-1 ring-rule ring-inset",
        )}
      >
        {done ? <Check className="size-4" strokeWidth={2.4} /> : null}
      </span>
      <span className="min-w-0 flex-1 basis-48">
        <span
          className={cn(
            "block font-ui text-ui font-medium",
            done ? "text-mute" : "text-ink",
          )}
        >
          {label}
          {done ? <span className="sr-only"> (done)</span> : null}
        </span>
        <span className="block font-ui text-meta text-mute">{detail}</span>
      </span>
      {done ? null : action}
    </li>
  );
}
