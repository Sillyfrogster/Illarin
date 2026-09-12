"use client";

import { ArrowUpRight, ImageUp, Package, Plus } from "lucide-react";
import Image from "next/image";
import { useRef, useState } from "react";
import {
  Nothing,
  PanelHead,
  Past,
  PastRow,
  Row,
  RowMark,
  Rows,
  StartAction,
} from "@/components/register/RowParts";
import { Consequence, StepForm } from "@/components/register/StepParts";
import { Field, TextInput } from "@/components/ui/field";
import { Sortable, SortableItemHandle } from "@/components/ui/sortable";
import {
  configureApp,
  orderApps,
  setAppDestinations,
  updateApp,
  uploadAppMark,
} from "@/lib/api/publication";
import type { PublicationApp, PublicationDestination } from "@/lib/api/query";
import { nothingIn } from "@/lib/publication-register";
import { moved } from "@/lib/reorder";
import { DestinationChoice } from "./choices";

export function AppRows({
  apps,
  onFailure,
  onOpen,
  onOrdered,
  onSaved,
}: {
  apps: PublicationApp[];
  onFailure: (message: string) => void;
  onOpen: (app: PublicationApp | null) => void;
  onOrdered: (apps: PublicationApp[]) => void;
  onSaved: (saved: PublicationApp, added: boolean) => void;
}) {
  const current = apps.filter((one) => !one.retired);
  const retired = apps.filter((one) => one.retired);

  const [placing, setPlacing] = useState<PublicationApp[] | null>(null);
  const shown = placing ?? current;

  async function reorder(from: number, to: number) {
    const next = moved(current, from, to);
    setPlacing(next);
    const answer = await orderApps(
      next.map((one) => one.id).concat(retired.map((one) => one.id)),
    );
    setPlacing(null);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onOrdered(answer.value.apps);
  }

  async function bringBack(app: PublicationApp) {
    const answer = await updateApp(app.id, { retired: false });
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onSaved(answer.value, false);
  }

  return (
    <>
      <PanelHead
        action={
          <StartAction icon={Plus} onClick={() => onOpen(null)}>
            Add an app
          </StartAction>
        }
        id="register-heading"
        title="Apps"
      />

      {current.length === 0 ? (
        <Nothing>{nothingIn("apps", { apps })}</Nothing>
      ) : (
        <Sortable
          disabled={placing !== null}
          ids={shown.map((one) => one.id)}
          labels={(id) => shown.find((one) => one.id === id)?.name ?? "app"}
          onMove={(from, to) => void reorder(from, to)}
        >
          <Rows>
            {shown.map((app) => (
              <Row
                aside={
                  <SortableItemHandle
                    disabled={placing !== null || shown.length < 2}
                    label={`Move ${app.name}`}
                  />
                }
                facts={
                  <>
                    <span className="font-mono text-label">{app.slug}</span>
                    <span>{app.home.replace(/^https:\/\//, "")}</span>
                    <span>{announces(app)}</span>
                  </>
                }
                key={app.id}
                lead={
                  <RowMark>
                    {app.mark ? (
                      <Image
                        alt=""
                        className="size-11 object-contain"
                        height={44}
                        src={app.mark.url}
                        unoptimized
                        width={44}
                      />
                    ) : (
                      <Package
                        aria-hidden="true"
                        className="size-5"
                        strokeWidth={1.6}
                      />
                    )}
                  </RowMark>
                }
                onOpen={() => onOpen(app)}
                open={`Edit ${app.name}`}
                sortableId={app.id}
                title={app.name}
              />
            ))}
          </Rows>
        </Sortable>
      )}

      {retired.length > 0 ? (
        <Past summary={`${retired.length} retired`}>
          {retired.map((app) => (
            <PastRow
              action={
                <button
                  className="inline-flex min-h-11 items-center rounded-control px-3 font-ui text-meta font-medium text-accent outline-offset-3 hover:underline"
                  onClick={() => bringBack(app)}
                  type="button"
                >
                  Reactivate app
                </button>
              }
              key={app.id}
            >
              {app.name}
            </PastRow>
          ))}
        </Past>
      ) : null}
    </>
  );
}

function announces(app: PublicationApp): string {
  const many = app.destinations.length;
  if (many === 0) return "No announcement destinations";
  return many === 1
    ? `Announces to ${app.destinations[0].name}`
    : `Announces to ${many} destinations`;
}

export function AppStep({
  destinations,
  existing,
  onClose,
  onFailure,
  onSaved,
}: {
  destinations: PublicationDestination[];
  existing: PublicationApp | null;
  onClose: () => void;
  onFailure: (message: string) => void;
  onSaved: (saved: PublicationApp, added: boolean) => void;
}) {
  const [name, setName] = useState(existing?.name ?? "");
  const [slug, setSlug] = useState(existing?.slug ?? "");
  const [home, setHome] = useState(existing?.home ?? "");
  const [allowed, setAllowed] = useState<string[]>(
    existing ? existing.destinations.map((one) => one.id) : [],
  );
  const [defaults, setDefaults] = useState<string[]>(
    existing
      ? existing.destinations
          .filter((one) => one.byDefault)
          .map((one) => one.id)
      : [],
  );
  const [chosen, setChosen] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [busy, setBusy] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const shown = preview || existing?.mark?.url || "";
  const ready = Boolean(name.trim() && slug.trim() && home.trim());

  function take(file: File | undefined) {
    if (!file) return;
    setChosen(file);
    setPreview(URL.createObjectURL(file));
  }

  async function save() {
    setBusy(true);
    const written = existing
      ? await updateApp(existing.id, {
          home: home.trim(),
          name: name.trim(),
          slug: slug.trim(),
        })
      : await configureApp(slug.trim(), name.trim(), home.trim());
    if (written.error || !written.value) {
      setBusy(false);
      onFailure(written.error ?? "");
      return;
    }
    let saved = written.value;
    if (chosen) {
      const marked = await uploadAppMark(saved.id, chosen);
      if (marked.error || !marked.value) {
        setBusy(false);
        onSaved(saved, !existing);
        onFailure(marked.error ?? "");
        return;
      }
      saved = marked.value;
    }
    const policed = await setAppDestinations(saved.id, {
      defaultDestinationIds: defaults,
      destinationIds: allowed,
    });
    setBusy(false);
    if (policed.error || !policed.value) {
      onSaved(saved, !existing);
      onFailure(policed.error ?? "");
      return;
    }
    onSaved(policed.value, !existing);
    onClose();
  }

  async function retire() {
    if (!existing) return;
    setBusy(true);
    const answer = await updateApp(existing.id, { retired: true });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onSaved(answer.value, false);
    onClose();
  }

  return (
    <StepForm
      busy={busy}
      commit={existing ? "Save" : "Add"}
      onCommit={save}
      ready={ready}
      under={
        existing ? (
          <Consequence
            action="Retire"
            busy={busy}
            confirm="Retire app"
            onConfirm={retire}
          >
            Nobody can be approved for {existing.name} while it is retired.
            Approvals that already name it keep working, and everything
            published under it stays on the blog.
          </Consequence>
        ) : null
      }
    >
      <p className="max-w-[52ch] font-prose text-meta text-mute">
        Add a project to the blog. Approve contributors separately.
      </p>

      <Field htmlFor="app-name" label="Name">
        <TextInput
          id="app-name"
          maxLength={48}
          onChange={(event) => setName(event.target.value)}
          placeholder="Lumiverse"
          value={name}
        />
      </Field>

      <Field
        hint="Used in the app's blog address. Changing its name does not change this slug."
        htmlFor="app-slug"
        label="Slug"
      >
        <TextInput
          className="font-mono"
          id="app-slug"
          maxLength={40}
          onChange={(event) => setSlug(event.target.value)}
          placeholder="lumiverse"
          value={slug}
        />
      </Field>

      <Field htmlFor="app-home" label="Project URL">
        <TextInput
          id="app-home"
          maxLength={300}
          onChange={(event) => setHome(event.target.value)}
          placeholder="https://lumiverse.app"
          type="url"
          value={home}
        />
      </Field>

      <Field label="App logo">
        <div className="flex items-center gap-3">
          <span className="grid size-12 shrink-0 place-items-center overflow-hidden rounded-control bg-deep text-mute">
            {shown ? (
              <Image
                alt=""
                className="size-12 object-contain"
                height={48}
                src={shown}
                unoptimized
                width={48}
              />
            ) : (
              <Package
                aria-hidden="true"
                className="size-5"
                strokeWidth={1.6}
              />
            )}
          </span>
          <button
            className="inline-flex min-h-11 items-center gap-2 rounded-control bg-deep px-4 font-ui text-ui font-medium text-ink outline-offset-3 hover:bg-rule/45"
            onClick={() => fileInput.current?.click()}
            type="button"
          >
            <ImageUp aria-hidden="true" className="size-4" strokeWidth={1.8} />
            {shown ? "Replace" : "Upload logo"}
          </button>
          <input
            accept="image/png,image/jpeg,image/webp,image/gif"
            aria-label="Choose a mark for this app"
            className="sr-only"
            onChange={(event) => take(event.target.files?.[0])}
            ref={fileInput}
            type="file"
          />
        </div>
      </Field>

      <DestinationChoice
        allowed={allowed}
        defaults={defaults}
        destinations={destinations}
        legend="Announcement destinations"
        onAllowed={setAllowed}
        onDefaults={setDefaults}
      />

      {existing ? (
        <a
          className="inline-flex min-h-11 items-center gap-1.5 font-ui text-meta text-mute outline-offset-3 [overflow-wrap:anywhere] hover:text-ink"
          href={existing.home}
          rel="noreferrer noopener"
          target="_blank"
        >
          {existing.home.replace(/^https:\/\//, "")}
          <ArrowUpRight aria-hidden="true" className="size-3.5" />
        </a>
      ) : null}
    </StepForm>
  );
}
