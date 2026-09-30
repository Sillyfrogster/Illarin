"use client";

import { ArrowUpRight, GitBranch, RotateCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CheckboxRow } from "@/components/ui/checkbox";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { InputCopy } from "@/components/ui/input-copy";
import { Item, ItemGroup } from "@/components/ui/item";
import { RadioGroup } from "@/components/ui/radio-group";
import { api } from "@/lib/api/client";

type ReleaseSource = {
  repository: string;
  attachment: string | null;
  includePrereleases: boolean;
  verified: boolean;
  proof?: string;
  lastError: string | null;
  imports: {
    id: number;
    tag: string;
    status: "queued" | "held" | "failed" | "published";
    failure: string | null;
    versionNumber: number | null;
  }[];
};

function failure(error: unknown): string {
  const detail = error as { error?: unknown } | null;
  return typeof detail?.error === "string"
    ? detail.error
    : "Illarin could not update GitHub releases. Try again.";
}

export function GitHubReleases({ workId }: { workId: string }) {
  const path = `/v1/works/${workId}/github-releases`;
  const [source, setSource] = useState<ReleaseSource | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [repository, setRepository] = useState("");
  const [attachment, setAttachment] = useState("");
  const [useAttachment, setUseAttachment] = useState(false);
  const [includePrereleases, setIncludePrereleases] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  const refresh = useCallback(async () => {
    const result = await api<ReleaseSource | null>("GET", path, {
      cache: "no-store",
    });
    if (result.error) throw new Error(failure(result.error));
    setSource(result.data ?? null);
    setLoading(false);
  }, [path]);

  useEffect(() => {
    void refresh().catch((error) => {
      setLoading(false);
      setMessage(
        error instanceof Error
          ? error.message
          : "Illarin could not load GitHub releases.",
      );
    });
  }, [refresh]);

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setMessage("");
    try {
      await action();
      await refresh();
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "Illarin could not update GitHub releases.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function change(
    method: "PUT" | "POST" | "DELETE",
    suffix = "",
    body?: unknown,
  ) {
    const result = await api<ReleaseSource | undefined>(method, path + suffix, {
      body,
    });
    if (result.error) throw new Error(failure(result.error));
  }

  if (loading) return null;

  return (
    <section
      className="mx-auto mt-chapter w-full max-w-5xl px-[var(--gutter)]"
      id="github-releases"
      aria-labelledby="github-releases-title"
    >
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="mb-2 flex items-center gap-2 text-meta text-mute">
            <GitBranch aria-hidden="true" className="size-4" />
            Extension updates
          </p>
          <h2
            id="github-releases-title"
            className="font-display text-title font-medium text-ink"
          >
            GitHub releases
          </h2>
        </div>
        {source && !editing ? (
          <Button
            disabled={busy}
            onClick={() => void run(refresh)}
            size="compact"
            type="button"
            variant="secondary"
          >
            <RotateCw aria-hidden="true" /> Refresh status
          </Button>
        ) : null}
      </div>
      {message ? (
        <Alert className="mb-5" tone="stop">
          {message}
        </Alert>
      ) : null}

      {!source || editing ? (
        <Card>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              void run(async () => {
                await change("PUT", "", {
                  repository,
                  attachment: useAttachment ? attachment : null,
                  includePrereleases,
                });
                setEditing(false);
              });
            }}
          >
            <CardHeader>
              <CardTitle>Connect a repository</CardTitle>
              <CardDescription>
                New releases become extension versions
              </CardDescription>
            </CardHeader>
            <div className="grid gap-8 px-4 pt-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)] lg:gap-10">
              <div className="flex flex-col gap-5">
                <Field label="Repository URL">
                  <Input
                    onChange={(event) => setRepository(event.target.value)}
                    placeholder="https://github.com/owner/repository"
                    required
                    type="url"
                    value={repository}
                  />
                </Field>
                <p className="max-w-sm text-meta text-mute">
                  You need to prove the repository is yours before Illarin
                  imports its releases.
                </p>
              </div>
              <div className="flex flex-col gap-5">
                <fieldset>
                  <legend className="mb-2 text-meta font-medium text-ink">
                    File to import
                  </legend>
                  <RadioGroup
                    onValueChange={(next) =>
                      setUseAttachment(next === "attachment")
                    }
                    options={[
                      {
                        value: "archive",
                        label: "Source archive",
                        hint: "GitHub's release archive",
                      },
                      {
                        value: "attachment",
                        label: "Named file",
                        hint: "Same attachment on each release",
                      },
                    ]}
                    value={useAttachment ? "attachment" : "archive"}
                  />
                  {useAttachment ? (
                    <Input
                      aria-label="Attachment file name"
                      className="mt-3"
                      onChange={(event) => setAttachment(event.target.value)}
                      placeholder="extension.zip"
                      required
                      value={attachment}
                    />
                  ) : null}
                </fieldset>
                <CheckboxRow
                  checked={includePrereleases}
                  label="Include prereleases"
                  onCheckedChange={setIncludePrereleases}
                />
              </div>
            </div>
            <CardFooter className="justify-end gap-2 pt-4">
              {source ? (
                <Button
                  disabled={busy}
                  onClick={() => setEditing(false)}
                  type="button"
                  variant="ghost"
                >
                  Cancel
                </Button>
              ) : null}
              <Button disabled={busy} type="submit" variant="primary">
                Connect repository
              </Button>
            </CardFooter>
          </form>
        </Card>
      ) : (
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]">
          <Card>
            <CardHeader>
              <CardTitle>Repository</CardTitle>
              <CardAction
                className={`text-meta font-medium ${source.verified ? "text-accent" : "text-mute"}`}
              >
                {source.verified ? "Connected" : "Waiting for proof"}
              </CardAction>
            </CardHeader>
            <div className="flex flex-col gap-5 px-4 pt-4">
              <a
                className="group flex w-fit max-w-full items-start gap-2 text-section font-medium text-ink hover:text-accent"
                href={`https://github.com/${source.repository}`}
                rel="noreferrer"
                target="_blank"
              >
                <span className="wrap-anywhere">{source.repository}</span>
                <ArrowUpRight
                  aria-hidden="true"
                  className="mt-1 size-4 shrink-0 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
                />
                <span className="sr-only">(opens in a new tab)</span>
              </a>
              <dl className="grid gap-4 border-t border-rule pt-5 text-meta sm:grid-cols-2">
                <div>
                  <dt className="text-mute">File</dt>
                  <dd className="mt-1 font-medium text-ink">
                    {source.attachment ?? "Source archive"}
                  </dd>
                </div>
                <div>
                  <dt className="text-mute">Releases</dt>
                  <dd className="mt-1 font-medium text-ink">
                    {source.includePrereleases
                      ? "Stable and prerelease"
                      : "Stable only"}
                  </dd>
                </div>
              </dl>
            </div>
            <CardFooter className="gap-1">
              <Button
                disabled={busy}
                onClick={() => {
                  setRepository(`https://github.com/${source.repository}`);
                  setAttachment(source.attachment ?? "");
                  setUseAttachment(source.attachment !== null);
                  setIncludePrereleases(source.includePrereleases);
                  setEditing(true);
                }}
                size="compact"
                type="button"
                variant="ghost"
              >
                Change source
              </Button>
              <Button
                disabled={busy}
                onClick={() => void run(() => change("DELETE"))}
                size="compact"
                type="button"
                variant="ghost"
              >
                Disconnect
              </Button>
            </CardFooter>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>
                {source.verified ? "Release history" : "Prove ownership"}
              </CardTitle>
              <CardDescription>
                {source.verified
                  ? "Checks hourly"
                  : "One file in your repository"}
              </CardDescription>
            </CardHeader>
            <div className="px-4 pt-4">
              {!source.verified ? (
                <div className="flex flex-col gap-4">
                  <p className="text-ui text-ink">
                    Add{" "}
                    <code className="font-mono text-meta">.illarin-proof</code>{" "}
                    to the repository root with this code inside:
                  </p>
                  <InputCopy value={source.proof ?? ""} />
                  <Button
                    className="self-start"
                    disabled={busy}
                    onClick={() => void run(() => change("POST", "/verify"))}
                    type="button"
                    variant="primary"
                  >
                    Verify repository
                  </Button>
                </div>
              ) : source.imports.length > 0 ? (
                <ItemGroup
                  as="ol"
                  className="-mx-4 rounded-none bg-transparent"
                >
                  {source.imports.map((item) => (
                    <Item
                      className="flex-row flex-wrap items-start justify-between gap-3"
                      key={item.id}
                    >
                      <div className="min-w-0">
                        <p className="break-all text-ui font-medium text-ink">
                          {item.tag}
                        </p>
                        <p
                          className={`mt-0.5 text-meta ${item.status === "failed" ? "text-stop" : "text-mute"}`}
                        >
                          {item.status === "published"
                            ? `Published as version ${item.versionNumber}`
                            : item.status === "held"
                              ? "Waiting for your draft changes"
                              : item.status === "queued"
                                ? "Import queued"
                                : "Import failed"}
                        </p>
                        {item.failure ? (
                          <p className="mt-2 text-meta text-stop">
                            {item.failure}
                          </p>
                        ) : null}
                      </div>
                      {item.status === "held" || item.status === "failed" ? (
                        <Button
                          disabled={busy}
                          onClick={() =>
                            void run(() => change("POST", `/${item.id}/retry`))
                          }
                          size="compact"
                          type="button"
                          variant="secondary"
                        >
                          {item.status === "held" ? "Resume" : "Retry"}
                        </Button>
                      ) : null}
                    </Item>
                  ))}
                </ItemGroup>
              ) : (
                <p className="text-ui text-mute">
                  No releases imported yet. Illarin checks GitHub each hour.
                </p>
              )}
              {source.lastError ? (
                <p
                  className="mt-4 rounded-control bg-stop-wash px-3 py-2 text-meta text-stop"
                  role="alert"
                >
                  {source.lastError}
                </p>
              ) : null}
            </div>
          </Card>
        </div>
      )}
    </section>
  );
}
