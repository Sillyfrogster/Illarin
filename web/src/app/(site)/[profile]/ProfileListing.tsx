import { CatalogSurface } from "@/components/catalog/CatalogSurface";
import type {
  BrowseFilters,
  BrowsePage,
  DeletedWork,
  Profile,
} from "@/lib/api/query";
import { DeletedAssets } from "./DeletedAssets";
import { ProfileBanner } from "./ProfileBanner";

export function ProfileListing({
  deletedAssets,
  filters,
  initialPage,
  profile,
}: {
  deletedAssets: DeletedWork[] | null;
  filters: BrowseFilters;
  initialPage: BrowsePage | null;
  profile: Profile;
}) {
  const isOwner = deletedAssets !== null;
  const name = profile.displayName || `@${profile.handle}`;

  return (
    <>
      <ProfileBanner
        deletedCount={deletedAssets?.length ?? null}
        isOwner={isOwner}
        profile={profile}
      />
      <CatalogSurface
        basePath={`/@${profile.handle}`}
        creator={profile.handle}
        filters={filters}
        heading={isOwner ? "Your creations" : `Published by ${name}`}
        initialPage={initialPage}
        search={{
          label: `Search @${profile.handle}'s creations`,
          placeholder: "Search their creations",
        }}
      />
      {deletedAssets !== null ? (
        <DeletedAssets initialItems={deletedAssets} />
      ) : null}
    </>
  );
}
