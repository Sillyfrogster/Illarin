import { notFound } from "next/navigation";
import { fetchProfile } from "@/lib/api/query";
import { renderLinkCard } from "@/lib/link-card";
import { readProfileAddress } from "@/lib/profile-address";
import { countWords } from "@/lib/profile-portfolio";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ profile: string }> },
): Promise<Response> {
  const handle = readProfileAddress(decodeURIComponent((await params).profile));
  if (!handle) notFound();
  const profile = await fetchProfile(handle);
  if (!profile) notFound();
  const name = profile.displayName || profile.handle;
  return renderLinkCard({
    title: name,
    image: profile.banner,
    avatar: profile.avatar,
    initials: name
      .split(/\s+/)
      .slice(0, 2)
      .map((word) => Array.from(word)[0])
      .join("")
      .toUpperCase(),
    byline: `@${profile.handle}`,
    description: profile.biography,
    footer: countWords(profile.works, profile.followers),
  });
}
