"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Shell } from "@/components/layout/Shell";
import { Trouble } from "@/components/ui/field";
import { Gate } from "@/components/ui/gate";
import { readDeletedPosts, readPosts } from "@/lib/api/posts";
import { readWorkspace } from "@/lib/api/publication";
import type { Post, PublicationWorkspace } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import {
  inStanding,
  nothingThere,
  STANDINGS,
  type Standing,
} from "@/lib/post-standing";
import { Approvals } from "./Approvals";
import { PostRow } from "./PostRow";
import { StandingRail } from "./StandingRail";
import { StartPost } from "./StartPost";

/** Everything this account has written for the blog, and what Illarin approved it to publish. */
export function PostDesk() {
  const { account } = useAuth();
  const [workspace, setWorkspace] = useState<PublicationWorkspace | null>(null);
  const [posts, setPosts] = useState<Post[] | null>(null);
  const [deleted, setDeleted] = useState<Post[]>([]);
  const [failure, setFailure] = useState("");
  const [standing, setStanding] = useState<Standing>("everything");

  const load = useCallback(async () => {
    const [open, written, waiting] = await Promise.all([
      readWorkspace(),
      readPosts(),
      readDeletedPosts(),
    ]);
    if (open.error || !open.value) {
      setFailure(open.error ?? "Your approvals could not be read.");
      return;
    }
    setFailure("");
    setWorkspace(open.value);
    setPosts(written.value?.posts ?? []);
    setDeleted(waiting.value?.posts ?? []);
  }, []);

  useEffect(() => {
    if (!account) return;
    void load();
  }, [account, load]);

  const counts = useMemo(() => tally(posts ?? [], deleted), [posts, deleted]);
  const shown = (standing === "deleted" ? deleted : (posts ?? [])).filter(
    (post) => inStanding(post, standing),
  );

  return (
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <header className="max-w-[52ch]">
        <p className="font-ui text-meta font-medium tracking-[0.14em] text-accent uppercase">
          Publication
        </p>
        <h1 className="mt-3 font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          Your posts
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">
          Everything you have written for the Illarin blog, and what Illarin
          approved you to publish.
        </p>
      </header>

      <div className="mt-10">
        <Inside
          account={account}
          counts={counts}
          failure={failure}
          onChoose={setStanding}
          onFailure={setFailure}
          shown={shown}
          standing={standing}
          waiting={posts === null}
          workspace={workspace}
        />
      </div>
    </Shell>
  );
}

function Inside({
  account,
  counts,
  failure,
  onChoose,
  onFailure,
  shown,
  standing,
  waiting,
  workspace,
}: {
  account: ReturnType<typeof useAuth>["account"];
  counts: Record<Standing, number>;
  failure: string;
  onChoose: (standing: Standing) => void;
  onFailure: (message: string) => void;
  shown: Post[];
  standing: Standing;
  waiting: boolean;
  workspace: PublicationWorkspace | null;
}) {
  if (account === undefined) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        Checking your account…
      </p>
    );
  }

  if (!account) {
    return (
      <Gate
        action="Sign in"
        heading="Sign in to see what you may publish"
        href="/sign-in"
        line="Illarin opens this page to the accounts it has approved."
      />
    );
  }

  if (!workspace || waiting) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        {failure || "Reading your approvals…"}
      </p>
    );
  }

  if (workspace.grants.length === 0 && !workspace.admin) {
    return (
      <Gate
        action="Back to Illarin"
        heading="Nobody has approved you to publish"
        href="/"
        line="Approval comes from Illarin's publication authority. An admin or a moderator role is not the same thing."
      />
    );
  }

  return (
    <div className="grid min-w-0 gap-section lg:grid-cols-[minmax(0,1fr)_minmax(0,22rem)] lg:gap-14">
      <section aria-labelledby="written" className="min-w-0">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <h2
            className="font-display text-section font-medium tracking-tight text-ink"
            id="written"
          >
            Written
          </h2>
          <StartPost onFailure={onFailure} workspace={workspace} />
        </div>

        {failure ? (
          <div className="mt-5">
            <Trouble>{failure}</Trouble>
          </div>
        ) : null}

        <div className="mt-5">
          <StandingRail chosen={standing} counts={counts} onChoose={onChoose} />
        </div>

        {shown.length === 0 ? (
          <p className="mt-8 max-w-[54ch] font-prose text-prose text-mute">
            {nothingThere(standing)}
          </p>
        ) : (
          <ul className="mt-4 -mx-4 flex list-none flex-col sm:-mx-5">
            {shown.map((post) => (
              <PostRow key={post.id} post={post} />
            ))}
          </ul>
        )}
      </section>

      <div className="min-w-0 lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:self-start">
        <Approvals workspace={workspace} />
      </div>
    </div>
  );
}

// What each standing holds, so the rail says so before it is opened.
function tally(posts: Post[], deleted: Post[]): Record<Standing, number> {
  const counted = {} as Record<Standing, number>;
  for (const standing of STANDINGS) {
    const from = standing === "deleted" ? deleted : posts;
    counted[standing] = from.filter((post) =>
      inStanding(post, standing),
    ).length;
  }
  return counted;
}
