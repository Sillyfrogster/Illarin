"use client";

import {
  ArrowDown,
  ArrowUp,
  ImageUp,
  Package,
  Plus,
  RotateCcw,
  X,
} from "lucide-react";
import Image from "next/image";
import {
  type FormEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  configureApp,
  orderApps,
  readApps,
  updateApp,
  uploadAppMark,
} from "@/lib/api/publication";
import type { PublicationApp } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./PublicationSection.module.css";

export function PublicationApps() {
  const [apps, setApps] = useState<PublicationApp[] | null>(null);
  const [failure, setFailure] = useState("");
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");
  const [home, setHome] = useState("");
  const [busy, setBusy] = useState(false);
  const [marking, setMarking] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    const answer = await readApps();
    if (answer.error) {
      setFailure(answer.error);
      return;
    }
    setFailure("");
    setApps(answer.value?.apps ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const current = (apps ?? []).filter((app) => !app.retired);
  const retired = (apps ?? []).filter((app) => app.retired);

  async function add(event: FormEvent) {
    event.preventDefault();
    if (!slug.trim() || !name.trim() || !home.trim() || busy) return;
    setBusy(true);
    const answer = await configureApp(slug.trim(), name.trim(), home.trim());
    setBusy(false);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setSlug("");
    setName("");
    setHome("");
    setApps([...(apps ?? []), answer.value]);
  }

  async function reorder(index: number, step: number) {
    const target = index + step;
    if (!apps || target < 0 || target >= current.length) return;
    const answer = await orderApps(
      moved(
        apps.map((app) => app.id),
        apps.indexOf(current[index]),
        apps.indexOf(current[target]),
      ),
    );
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setApps(answer.value.apps);
  }

  async function setRetired(app: PublicationApp, retiredNow: boolean) {
    const answer = await updateApp(app.id, { retired: retiredNow });
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    replace(answer.value);
  }

  function chooseMark(app: PublicationApp) {
    setMarking(app.id);
    fileInput.current?.click();
  }

  async function sendMark(file: File | undefined) {
    if (!file || !marking) return;
    const answer = await uploadAppMark(marking, file);
    setMarking("");
    if (fileInput.current) fileInput.current.value = "";
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    replace(answer.value);
  }

  function replace(changed: PublicationApp) {
    setApps((known) =>
      (known ?? []).map((app) => (app.id === changed.id ? changed : app)),
    );
  }

  if (!apps) {
    return (
      <p className={styles.loading} aria-live="polite">
        {failure || "Reading the apps…"}
      </p>
    );
  }

  return (
    <div className={styles.region}>
      {failure ? (
        <p className={styles.failure} role="alert">
          {failure}
        </p>
      ) : null}

      {current.length > 0 ? (
        <ol className={styles.list}>
          {current.map((app, index) => (
            <li className={styles.row} key={app.id}>
              <span className={styles.mark}>
                {app.mark ? (
                  <Image
                    src={app.mark.url}
                    alt=""
                    width={40}
                    height={40}
                    unoptimized
                  />
                ) : (
                  <Package size={18} strokeWidth={1.6} aria-hidden="true" />
                )}
              </span>
              <span className={styles.identity}>
                <span className={styles.name}>{app.name}</span>
                <span className={styles.slug}>{app.slug}</span>
                <a
                  className={styles.home}
                  href={app.home}
                  rel="noreferrer noopener"
                  target="_blank"
                >
                  {app.home.replace(/^https:\/\//, "")}
                </a>
              </span>
              <span className={styles.rowActions}>
                <button
                  type="button"
                  onClick={() => reorder(index, -1)}
                  disabled={index === 0}
                  aria-label={`Move ${app.name} up`}
                >
                  <ArrowUp size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => reorder(index, 1)}
                  disabled={index === current.length - 1}
                  aria-label={`Move ${app.name} down`}
                >
                  <ArrowDown size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => chooseMark(app)}
                  aria-label={`Upload a mark for ${app.name}`}
                >
                  <ImageUp size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => setRetired(app, true)}
                  aria-label={`Retire ${app.name}`}
                >
                  <X size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
              </span>
            </li>
          ))}
        </ol>
      ) : (
        <p className={styles.empty}>No app is configured.</p>
      )}

      <form className={styles.add} onSubmit={add}>
        <h3>Configure an app</h3>
        <div className={styles.fields}>
          <label htmlFor="app-name">
            Name
            <input
              id="app-name"
              value={name}
              maxLength={48}
              placeholder="Lumiverse"
              onChange={(event) => setName(event.target.value)}
            />
          </label>
          <label htmlFor="app-slug">
            Slug
            <input
              id="app-slug"
              value={slug}
              maxLength={40}
              placeholder="lumiverse"
              onChange={(event) => setSlug(event.target.value)}
            />
          </label>
          <label htmlFor="app-home">
            Address
            <input
              id="app-home"
              value={home}
              maxLength={300}
              placeholder="https://lumiverse.app"
              onChange={(event) => setHome(event.target.value)}
            />
          </label>
        </div>
        <p className={styles.note}>
          Configuring an app approves nobody. Approve a person under
          Contributors.
        </p>
        <button
          type="submit"
          disabled={busy || !slug.trim() || !name.trim() || !home.trim()}
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          {busy ? "Configuring" : "Configure"}
        </button>
      </form>

      {retired.length > 0 ? (
        <div className={styles.retired}>
          <h3>Retired</h3>
          <ul>
            {retired.map((app) => (
              <li key={app.id}>
                {app.name}
                <button
                  type="button"
                  onClick={() => setRetired(app, false)}
                  aria-label={`Bring ${app.name} back`}
                >
                  <RotateCcw size={14} strokeWidth={1.8} aria-hidden="true" />
                  Bring back
                </button>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <input
        className="sr-only"
        ref={fileInput}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif"
        onChange={(event) => sendMark(event.target.files?.[0])}
      />
    </div>
  );
}
