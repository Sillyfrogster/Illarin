"use client";

import { type CSSProperties, useState } from "react";
import { Shell } from "@/components/layout/Shell";
import {
  type ProfilePictureKind,
  removeProfilePicture,
  saveFeaturedWorks,
  saveProfile,
  saveProfilePicture,
} from "@/lib/api/profile";
import type {
  BrowseFilters,
  BrowsePage,
  BrowseWork,
  DeletedWork,
  Profile,
} from "@/lib/api/query";
import type { Answer } from "@/lib/api/request";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { toggleFeatured } from "@/lib/profile-portfolio";
import { Banner } from "./Banner";
import { DeletedWorks } from "./DeletedWorks";
import { FirstSteps } from "./FirstSteps";
import { type Editing, IdentityCard, type ProfileDraft } from "./IdentityCard";
import { RecentVersions } from "./RecentVersions";
import { RestrictedControl } from "./RestrictedControl";
import { Shelf } from "./Shelf";

function draftOf(profile: Profile): ProfileDraft {
  return {
    biography: profile.biography,
    contactEmail: profile.contactEmail,
    displayName: profile.displayName,
    links: profile.links,
  };
}

/** A creator's profile with its banner, card, shelf and recent versions */
export function ProfilePage({
  address,
  deletedWorks,
  filters,
  initial,
  initialPage,
}: {
  address: string;
  deletedWorks: DeletedWork[] | null;
  filters: BrowseFilters;
  initial: Profile;
  initialPage: BrowsePage | null;
}) {
  const { account } = useAuth();
  const [profile, setProfile] = useState(initial);
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [saving, setSaving] = useState(false);
  const [picturePending, setPicturePending] =
    useState<ProfilePictureKind | null>(null);
  const [failedField, setFailedField] = useState("");
  const [fieldTrouble, setFieldTrouble] = useState("");
  const [trouble, setTrouble] = useState("");
  const [said, setSaid] = useState("");
  const [featured, setFeatured] = useState(initial.featured);
  const [pinning, setPinning] = useState(false);
  const [pinTrouble, setPinTrouble] = useState("");

  const isOwner = profile.isOwner;
  const name = profile.displayName || `@${profile.handle}`;

  function hear(answer: Answer<Profile>, done: string): boolean {
    if (!answer.value) {
      const field = answer.refusal?.field ?? "";
      setFailedField(field);
      setFieldTrouble(field ? (answer.error ?? "") : "");
      setTrouble(
        field ? "Check the marked field and save again." : (answer.error ?? ""),
      );
      return false;
    }
    setProfile(answer.value);
    setFeatured(answer.value.featured);
    setFailedField("");
    setFieldTrouble("");
    setTrouble("");
    setSaid(done);
    return true;
  }

  function startEditing(focus?: string) {
    setSaid("");
    if (account && !account.emailVerified) {
      setTrouble("Verify your email before you edit your profile.");
      return;
    }
    setTrouble("");
    setDraft((current) => current ?? draftOf(profile));
    if (focus) {
      requestAnimationFrame(() => document.getElementById(focus)?.focus());
    }
  }

  async function save() {
    if (!draft) return;
    setSaving(true);
    const answer = await saveProfile(draft);
    setSaving(false);
    if (hear(answer, "Your profile is saved.")) setDraft(null);
  }

  async function changePicture(kind: ProfilePictureKind, file: File | null) {
    setPicturePending(kind);
    setSaid("");
    const answer = file
      ? await saveProfilePicture(kind, file)
      : await removeProfilePicture(kind);
    setPicturePending(null);
    hear(
      answer,
      kind === "banner"
        ? file
          ? "Your banner is updated."
          : "Your banner is removed."
        : file
          ? "Your picture is updated."
          : "Your picture is removed.",
    );
  }

  async function pin(work: BrowseWork) {
    const ids = toggleFeatured(
      featured.map((one) => one.id),
      work.id,
    );
    const previous = featured;
    setFeatured(ids.map((id) => previous.find((one) => one.id === id) ?? work));
    setPinning(true);
    setPinTrouble("");
    const answer = await saveFeaturedWorks(ids);
    setPinning(false);
    if (!answer.value) {
      setFeatured(previous);
      setPinTrouble(answer.error ?? "Featured work could not be saved.");
      return;
    }
    setProfile(answer.value);
    setFeatured(answer.value.featured);
  }

  const editing: Editing | null = draft
    ? {
        cancel: () => {
          setDraft(null);
          setFailedField("");
          setFieldTrouble("");
          setTrouble("");
        },
        change: setDraft,
        draft,
        failedField,
        fieldTrouble,
        onPickAvatar: (file) => void changePicture("avatar", file),
        onRemoveAvatar: () => void changePicture("avatar", null),
        picturePending: picturePending === "avatar",
        save: () => void save(),
        saving,
      }
    : null;

  return (
    <div
      className={cn(
        "bg-[linear-gradient(to_bottom,color-mix(in_oklab,var(--v-field)_91%,var(--tint)),var(--v-field)_40rem)] bg-no-repeat",
        !deletedWorks?.length && "pb-chapter",
      )}
      style={{ "--tint": profile.tint || "var(--v-accent)" } as CSSProperties}
    >
      <Banner
        banner={profile.restricted ? undefined : profile.banner}
        editing={editing !== null}
        onPick={(file) => void changePicture("banner", file)}
        onRemove={() => void changePicture("banner", null)}
        pending={picturePending === "banner"}
      />

      <Shell className="grid grid-cols-1 [grid-template-areas:'card'_'shelf'_'versions'] lg:grid-cols-[minmax(0,1fr)_minmax(0,23rem)] lg:grid-rows-[auto_1fr] lg:gap-x-12 lg:[grid-template-areas:'shelf_card'_'shelf_versions'] xl:gap-x-16">
        <div className="relative z-1 -mt-14 min-w-0 [grid-area:card] sm:-mt-20 lg:-mt-32">
          <IdentityCard
            address={address}
            editing={editing}
            isOwner={isOwner}
            onFollowChange={(following, followers) =>
              setProfile((current) => ({ ...current, followers, following }))
            }
            onStartEditing={() => startEditing()}
            profile={profile}
            said={said}
            trouble={trouble}
          />
        </div>

        <div className="min-w-0 pt-12 [grid-area:shelf] lg:pt-10">
          <Shelf
            featured={featured}
            filters={filters}
            firstSteps={
              isOwner ? (
                <FirstSteps
                  hasBanner={Boolean(profile.banner)}
                  hasBiography={Boolean(profile.biography)}
                  onEdit={startEditing}
                />
              ) : null
            }
            handle={profile.handle}
            initialPage={initialPage}
            isOwner={isOwner}
            name={name}
            pinning={
              isOwner && !profile.restricted
                ? {
                    featured: featured.map((one) => one.id),
                    pending: pinning,
                    toggle: (work) => void pin(work),
                    trouble: pinTrouble,
                  }
                : null
            }
            published={profile.works}
          />
        </div>

        <div className="min-w-0 pt-14 [grid-area:versions] empty:hidden lg:pt-12">
          <RecentVersions versions={profile.recentVersions} />
          <RestrictedControl
            handle={profile.handle}
            restricted={profile.restricted}
          />
        </div>
      </Shell>

      {deletedWorks !== null && deletedWorks.length > 0 ? (
        <div className="mt-chapter">
          <DeletedWorks initialItems={deletedWorks} />
        </div>
      ) : null}
    </div>
  );
}
