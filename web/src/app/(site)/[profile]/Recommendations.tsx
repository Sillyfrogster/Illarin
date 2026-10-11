"use client";

import { PinOff } from "lucide-react";
import Link from "next/link";
import { useId, useRef, useState } from "react";
import { BrowsePoster } from "@/components/browse/BrowsePoster";
import { GRID } from "@/components/browse/BrowseStates";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Tooltip } from "@/components/ui/tooltip";
import { recommendWork, removeRecommendation } from "@/lib/api/profile";
import type { NsfwPreference, Profile } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { workDisplayName } from "@/lib/work-name";
import { readWorkLink } from "@/lib/work-url";

const LIMIT = 4;

export function Recommendations({
  profile,
  preference,
  onChange,
}: {
  profile: Profile;
  preference: NsfwPreference;
  onChange: (profile: Profile) => void;
}) {
  const { account } = useAuth();
  const heading = useId();
  const input = useRef<HTMLInputElement>(null);
  const [link, setLink] = useState("");
  const [pending, setPending] = useState<string | null>(null);
  const [trouble, setTrouble] = useState("");
  const [said, setSaid] = useState("");
  const owner =
    profile.isOwner && account?.id === profile.id && account.emailVerified;
  const full = profile.recommendationCount >= LIMIT;
  const hidden = profile.recommendationCount - profile.recommendations.length;

  if (profile.restricted || (!owner && !profile.recommendations.length)) {
    return null;
  }

  async function change(workId: string, removing: boolean) {
    setPending(removing ? workId : "add");
    setTrouble("");
    setSaid("");
    const answer = await (removing
      ? removeRecommendation(workId)
      : recommendWork(workId));
    setPending(null);
    if (!answer.value) {
      setTrouble(
        answer.error ??
          "Illarin could not save your recommendations. Try again.",
      );
      return;
    }
    onChange(answer.value);
    if (!removing) setLink("");
    setSaid(removing ? "Recommendation removed." : "Recommendation added.");
    requestAnimationFrame(() => input.current?.focus());
  }

  return (
    <section aria-labelledby={heading} className="mb-14 min-w-0">
      <h2
        className="flex flex-wrap items-baseline gap-3 font-display text-section font-medium tracking-[-0.02em] text-ink"
        id={heading}
      >
        Recommendations
        {owner ? (
          <span className="font-ui text-meta font-medium tracking-normal text-mute tabular-nums">
            {profile.recommendationCount} of {LIMIT}
          </span>
        ) : null}
      </h2>
      {owner ? (
        <>
          <form
            className="mt-5"
            onSubmit={(event) => {
              event.preventDefault();
              if (pending || full) return;
              const id = readWorkLink(link, window.location.origin);
              if (!id) {
                setSaid("");
                setTrouble("Paste a link to a work on Illarin.");
                input.current?.focus();
                return;
              }
              void change(id, false);
            }}
          >
            <Field
              hint={
                full
                  ? "Remove a recommendation to add another."
                  : "Recommend up to four works by other creators on your profile."
              }
              label="Illarin work link"
            >
              <div className="flex flex-wrap gap-2">
                <Input
                  autoComplete="off"
                  className="basis-48 flex-1"
                  disabled={pending !== null || full}
                  inputMode="url"
                  onChange={(event) => setLink(event.target.value)}
                  placeholder="Paste a work link"
                  ref={input}
                  value={link}
                />
                <Button
                  disabled={pending !== null || full || !link.trim()}
                  loading={pending === "add"}
                  type="submit"
                >
                  Recommend
                </Button>
              </div>
            </Field>
          </form>
          {hidden > 0 ? (
            <p className="mt-3 font-ui text-meta text-mute">
              Some recommendations are hidden by your adult content setting.{" "}
              <Link
                className="text-accent hover:underline"
                href="/settings#browse"
              >
                Change adult content
              </Link>
            </p>
          ) : null}
          {trouble ? (
            <p className="mt-3 font-ui text-meta text-stop" role="alert">
              {trouble}
            </p>
          ) : null}
          <output className="mt-3 block font-ui text-meta text-mute empty:hidden">
            {said}
          </output>
        </>
      ) : null}
      {profile.recommendations.length ? (
        <ul className={`${GRID} mt-5`}>
          {profile.recommendations.map((work) => (
            <BrowsePoster
              action={
                owner ? (
                  <Tooltip content="Remove recommendation">
                    <Button
                      aria-label={`Remove recommendation for ${workDisplayName(work.name)}`}
                      className="bg-plane/90 backdrop-blur-sm"
                      disabled={pending !== null}
                      loading={pending === work.id}
                      onClick={() => void change(work.id, true)}
                      size="icon"
                    >
                      <PinOff aria-hidden="true" />
                    </Button>
                  </Tooltip>
                ) : undefined
              }
              key={work.id}
              preference={preference}
              work={work}
            />
          ))}
        </ul>
      ) : null}
    </section>
  );
}
