"use client";

import { ArrowUpRight } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { DocCode } from "@/components/docs/DocCode";
import { Shell } from "@/components/layout/Shell";
import { Gate } from "@/components/ui/gate";
import { readWorkspace } from "@/lib/api/publication";
import type { PublicationGrant, PublicationWorkspace } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { PUBLICATION_DOCS } from "@/lib/docs/collections";
import { grantExamples } from "@/lib/grant-examples";
import { GrantTokens } from "../GrantTokens";

export function ApiDesk() {
  const { account } = useAuth();
  const [workspace, setWorkspace] = useState<PublicationWorkspace | null>(null);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    const open = await readWorkspace();
    if (open.error || !open.value) {
      setFailure(open.error ?? "Your approvals could not be read.");
      return;
    }
    setFailure("");
    setWorkspace(open.value);
  }, []);

  useEffect(() => {
    if (!account) return;
    void load();
  }, [account, load]);

  return (
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <Link
        className="inline-flex min-h-11 items-center text-ui text-accent underline-offset-4 hover:underline"
        href="/admin/blog"
      >
        Your posts
      </Link>
      <header className="mt-4 max-w-[56ch]">
        <h1 className="font-display text-[2rem] leading-[1.15] font-semibold tracking-[-0.02em] text-balance">
          Publication API
        </h1>
        <p className="mt-3 font-prose text-prose text-mute">
          Create a token to publish blog posts from your own tools. The examples
          below use your app and permissions. Start with the{" "}
          <Link
            className="text-accent underline decoration-accent/40 underline-offset-[3px] hover:decoration-accent"
            href={PUBLICATION_DOCS.href}
          >
            Publication API guide
          </Link>
          .
        </p>
      </header>

      <div className="mt-10">
        <Inside account={account} failure={failure} workspace={workspace} />
      </div>
    </Shell>
  );
}

function Inside({
  account,
  failure,
  workspace,
}: {
  account: ReturnType<typeof useAuth>["account"];
  failure: string;
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
        heading="Sign in to manage API tokens"
        href="/sign-in?returnTo=%2Fadmin%2Fblog%2Fapi"
        line="Use the account approved to publish for your app."
      />
    );
  }

  if (!workspace) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        {failure || "Loading your approvals…"}
      </p>
    );
  }

  const active = workspace.grants.filter((grant) => grant.active);
  if (active.length === 0) {
    return (
      <Gate
        action="Read the reference"
        heading="No active publishing approval"
        href={PUBLICATION_DOCS.href}
        line="Ask Illarin to approve your account as a contributor for your app. Once approved, you can create a token here."
      />
    );
  }

  return (
    <div className="flex flex-col gap-14">
      {active.map((grant) => (
        <Approval grant={grant} key={grant.id} />
      ))}
    </div>
  );
}

function Approval({ grant }: { grant: PublicationGrant }) {
  return (
    <section
      aria-labelledby={`approval-${grant.id}`}
      className="grid min-w-0 gap-10 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)] lg:gap-14"
    >
      <div className="min-w-0">
        <h2
          className="font-display text-section font-semibold tracking-tight text-ink"
          id={`approval-${grant.id}`}
        >
          {grant.app.name}
        </h2>
        <a
          className="mt-1 inline-flex min-h-8 items-center gap-1.5 font-ui text-meta text-mute outline-offset-3 [overflow-wrap:anywhere] hover:text-ink"
          href={grant.app.home}
          rel="noreferrer noopener"
          target="_blank"
        >
          {grant.app.home.replace(/^https:\/\//, "")}
          <ArrowUpRight aria-hidden="true" className="size-3.5" />
        </a>
        <div className="mt-6">
          <GrantTokens grant={grant} />
        </div>
      </div>

      <div className="min-w-0 max-w-[46rem]">
        <h3 className="font-display text-ui font-semibold text-ink">
          IDs for your app
        </h3>
        <Ids grant={grant} />

        <h3 className="mt-10 font-display text-ui font-semibold text-ink">
          Create and publish a post for {grant.app.name}
        </h3>
        <p className="mt-1 font-prose text-meta text-mute">
          Send these requests to the main site, in order. Replace{" "}
          <code className="font-mono">&lt;token&gt;</code> with the secret you
          copied when creating a token. Replace the post ID, version and sample
          writing as each step explains. Generate a new idempotency key for each
          action and reuse it when retrying that action.
        </p>
        {grantExamples(grant).map((example) => (
          <div className="mt-6" key={example.title}>
            <h4 className="font-ui text-ui font-medium text-ink">
              {example.title}
            </h4>
            <p className="mt-1 font-prose text-meta text-mute">
              {example.note}
            </p>
            <DocCode
              label={example.title}
              language={example.language}
              source={example.source}
            />
          </div>
        ))}
      </div>
    </section>
  );
}

function Ids({ grant }: { grant: PublicationGrant }) {
  const rows = [
    ["App", grant.app.name, grant.app.id],
    ...grant.categories.map((category) => [
      category.id === grant.defaultCategory.id
        ? "Category, default"
        : "Category",
      category.label,
      category.id,
    ]),
    ...grant.destinations.map((destination) => [
      destination.byDefault ? "Destination, default" : "Destination",
      `${destination.name} (${destination.kind})`,
      destination.id,
    ]),
  ];
  return (
    <div className="mt-2 overflow-x-auto">
      <table className="w-full min-w-[28rem] border-collapse text-left">
        <thead>
          <tr className="border-b border-rule">
            {["Kind", "Name", "ID"].map((head) => (
              <th
                className="px-3 py-2.5 font-ui text-meta font-semibold text-ink"
                key={head}
                scope="col"
              >
                {head}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map(([kind, name, id]) => (
            <tr className="border-b border-rule/60" key={id}>
              <td className="px-3 py-2.5 font-prose text-meta text-mute">
                {kind}
              </td>
              <td className="px-3 py-2.5 font-prose text-meta text-ink">
                {name}
              </td>
              <td className="px-3 py-2.5 font-mono text-label text-ink">
                {id}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
