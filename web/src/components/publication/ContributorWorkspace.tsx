"use client";

import { ArrowUpRight, Package } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { ConsoleGate } from "@/components/console/ConsoleGate";
import { ConsolePage } from "@/components/console/ConsolePage";
import { Section } from "@/components/console/Section";
import { readDeletedPosts, readPosts } from "@/lib/api/posts";
import { readWorkspace } from "@/lib/api/publication";
import type {
  Post,
  PublicationGrant,
  PublicationWorkspace,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import styles from "./ContributorWorkspace.module.css";
import { GrantTokens } from "./GrantTokens";
import { PostRows } from "./PostRows";

export function ContributorWorkspace() {
  const { account } = useAuth();
  const [open, setOpen] = useState<PublicationWorkspace | null>(null);
  const [posts, setPosts] = useState<Post[] | null>(null);
  const [deleted, setDeleted] = useState<Post[]>([]);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    const [workspace, written, waiting] = await Promise.all([
      readWorkspace(),
      readPosts(),
      readDeletedPosts(),
    ]);
    if (workspace.error || !workspace.value) {
      setFailure(workspace.error ?? "");
      return;
    }
    setFailure("");
    setOpen(workspace.value);
    setPosts(written.value?.posts ?? []);
    setDeleted(waiting.value?.posts ?? []);
  }, []);

  useEffect(() => {
    if (!account) return;
    void load();
  }, [account, load]);

  return (
    <ConsolePage
      eyebrow="Publication"
      heading="Your posts"
      hint="Everything you have written for the Illarin blog, and what Illarin approved you to publish."
    >
      <Inside
        account={account}
        deleted={deleted}
        failure={failure}
        onFailure={setFailure}
        open={open}
        posts={posts}
      />
    </ConsolePage>
  );
}

function Inside({
  account,
  open,
  posts,
  deleted,
  failure,
  onFailure,
}: {
  account: ReturnType<typeof useAuth>["account"];
  open: PublicationWorkspace | null;
  posts: Post[] | null;
  deleted: Post[];
  failure: string;
  onFailure: (message: string) => void;
}) {
  if (account === undefined) {
    return (
      <p className={rows.loading} aria-live="polite">
        Checking your account…
      </p>
    );
  }

  if (!account) {
    return (
      <ConsoleGate
        heading="Sign in to see what you may publish"
        line="Illarin opens this page to the accounts it has approved."
        href="/sign-in"
        action="Sign in"
      />
    );
  }

  if (!open || !posts) {
    return (
      <p className={rows.loading} aria-live="polite">
        {failure || "Reading your approvals…"}
      </p>
    );
  }

  if (open.grants.length === 0 && !open.admin) {
    return (
      <ConsoleGate
        heading="Nobody has approved you to publish"
        line="Approval comes from Illarin's publication authority. An admin or a moderator role is not the same thing."
        href="/"
        action="Back to Illarin"
      />
    );
  }

  return (
    <div className={styles.grants}>
      {failure ? (
        <p className={styles.failure} role="alert">
          {failure}
        </p>
      ) : null}
      <PostRows
        deleted={deleted}
        onFailure={onFailure}
        posts={posts}
        workspace={open}
      />
      {open.grants.map((grant) => (
        <Approval key={grant.id} grant={grant} />
      ))}
      {open.grants.length > 0 ? (
        <p className={styles.later}>
          Your{" "}
          <Link href={`/@${open.handle}`}>Verified App Contributor badge</Link>{" "}
          stays on your profile for as long as an approval stands.
        </p>
      ) : (
        <p className={styles.later}>
          You write as the Illarin Team. A post carries your name and your
          public positions.
        </p>
      )}
    </div>
  );
}

function Approval({ grant }: { grant: PublicationGrant }) {
  return (
    <Section
      lead={
        <span className={styles.mark}>
          {grant.app.mark ? (
            <Image
              src={grant.app.mark.url}
              alt=""
              width={40}
              height={40}
              unoptimized
            />
          ) : (
            <Package size={19} strokeWidth={1.6} aria-hidden="true" />
          )}
        </span>
      }
      title={grant.app.name}
      action={
        <a
          className={styles.home}
          href={grant.app.home}
          rel="noreferrer noopener"
          target="_blank"
        >
          {grant.app.home.replace(/^https:\/\//, "")}
          <ArrowUpRight size={14} strokeWidth={1.7} aria-hidden="true" />
        </a>
      }
    >
      <ol className={rows.list}>
        {grant.categories.map((category) => (
          <li className={rows.row} data-plain="true" key={category.id}>
            <span className={rows.name}>
              {category.label}{" "}
              <span className={rows.slug}>{category.slug}</span>
            </span>
            <span className={rows.detail} />
            <span className={rows.actions}>
              {category.id === grant.defaultCategory.id ? (
                <span className={styles.byDefault}>Default</span>
              ) : null}
            </span>
          </li>
        ))}
      </ol>
      <div className={styles.footnote}>
        <p className={styles.byline}>
          A post carries your name and {grant.app.name}, and Illarin stays the
          publisher. {grant.defaultCategory.label} is chosen unless you pick
          another.
        </p>
      </div>
      <GrantTokens grant={grant} />
    </Section>
  );
}
