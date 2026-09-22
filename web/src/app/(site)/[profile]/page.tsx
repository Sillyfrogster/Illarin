import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { cache } from "react";
import { fetchProfile, fetchWorks } from "@/lib/api/query";
import { buildBrowseHref, readBrowseFilters } from "@/lib/browse-url";
import { profilePath, readProfileAddress } from "@/lib/profile-address";
import {
  CARD_SIZE,
  pageMetadata,
  readableForMetadata,
  siteUrl,
} from "@/lib/site-metadata";
import { ProfilePage } from "./ProfilePage";

const loadProfile = cache(async (segment: string) => {
  const handle = readProfileAddress(decodeURIComponent(segment));
  return handle ? fetchProfile(handle, (await cookies()).toString()) : null;
});

export async function generateMetadata({
  params,
}: {
  params: Promise<{ profile: string }>;
}): Promise<Metadata> {
  const profile = await readableForMetadata(
    loadProfile((await params).profile),
  );
  if (!profile) return { title: "Not found" };

  const metadata = pageMetadata(
    profile.displayName
      ? `${profile.displayName} (@${profile.handle})`
      : `@${profile.handle}`,
    profile.biography ||
      `Characters, lorebooks, presets, themes and extensions published by ${profile.handle} on Illarin.`,
  );
  const canonical = profilePath(profile.handle);
  const card = `${canonical}/card.png`;
  return {
    ...metadata,
    alternates: { canonical },
    openGraph: {
      ...metadata.openGraph,
      url: canonical,
      images: [
        { url: card, alt: profile.displayName || profile.handle, ...CARD_SIZE },
      ],
    },
    twitter: { ...metadata.twitter, images: [card] },
  };
}

export default async function CreatorProfileListing({
  params,
  searchParams,
}: {
  params: Promise<{ profile: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { profile: encodedProfile } = await params;
  const requested = decodeURIComponent(encodedProfile);
  const [filters, profile, cookie] = await Promise.all([
    searchParams.then(readBrowseFilters),
    loadProfile(encodedProfile),
    cookies().then((value) => value.toString()),
  ]);
  if (!profile) notFound();

  const canonical = `@${profile.handle}`;
  if (requested !== canonical) {
    redirect(buildBrowseHref(filters, `/${canonical}`));
  }

  const initialPage = await fetchWorks(
    { ...filters, creator: profile.handle, limit: 24 },
    cookie,
  ).catch(() => null);

  return (
    <ProfilePage
      address={`${siteUrl}/${canonical}`}
      filters={filters}
      initial={profile}
      initialPage={initialPage}
    />
  );
}
