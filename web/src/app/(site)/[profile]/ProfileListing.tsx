import { BrowseSurface } from "@/components/browse/BrowseSurface";
import type {
  BrowseFilters,
  BrowsePage,
  DeletedWork,
  Profile,
} from "@/lib/api/query";
import { DeletedWorks } from "./DeletedWorks";
import { ProfileBanner } from "./ProfileBanner";

export function ProfileListing({
  deletedWorks,
  filters,
  initialPage,
  profile,
}: {
  deletedWorks: DeletedWork[] | null;
  filters: BrowseFilters;
  initialPage: BrowsePage | null;
  profile: Profile;
}) {
  const isOwner = deletedWorks !== null;
  const name = profile.displayName || `@${profile.handle}`;

  return (
    <>
      <ProfileBanner
        deletedCount={deletedWorks?.length ?? null}
        isOwner={isOwner}
        profile={profile}
      />
      <BrowseSurface
        basePath={`/@${profile.handle}`}
        creator={profile.handle}
        filters={filters}
        heading={isOwner ? "Your work" : `Published by ${name}`}
        initialPage={initialPage}
        search={{
          label: `Search @${profile.handle}'s creations`,
          placeholder: "Search their work",
        }}
      />
      {deletedWorks !== null ? (
        <DeletedWorks initialItems={deletedWorks} />
      ) : null}
    </>
  );
}
