import { QueryClient } from "@tanstack/react-query";
import {
  acceptCandidateVersion,
  type Candidate,
  reportStaleWorkingCopy,
} from "@/lib/working-copy";
import { browserFetch } from "./browser-mutation";
import { api } from "./client";
import type { components, paths } from "./schema";

export type AssetDetail = components["schemas"]["AssetDetail"];
export type AssetImage = components["schemas"]["AssetImage"];
export type AssetBlock = components["schemas"]["AssetBlock"];
export type AssetElement = components["schemas"]["AssetElement"];
export type SaveAssetBlockRequest =
  components["schemas"]["SaveAssetBlockRequest"];
export type ArrangeAssetBlocksRequest =
  components["schemas"]["ArrangeAssetBlocksRequest"];
export type EntryTableContent = components["schemas"]["EntryTableContent"];
export type LorebookEntry = EntryTableContent["entries"][number];
export type RecordListContent = components["schemas"]["RecordListContent"];
export type LumiaRecord = RecordListContent["records"][number];
export type PromptListContent = components["schemas"]["PromptListContent"];
export type PromptGroup = PromptListContent["groups"][number];
export type PromptFragment = PromptListContent["fragments"][number];
export type SettingGroupContent = components["schemas"]["SettingGroupContent"];
export type PresetSetting = SettingGroupContent["settings"][number];
export type ColorSetContent = components["schemas"]["ColorSetContent"];
export type ThemeColorMode = ColorSetContent["modes"][number];
export type ThemeColor = ThemeColorMode["colors"][number];
export type StylesheetSetContent =
  components["schemas"]["StylesheetSetContent"];
export type ThemeStylesheet = StylesheetSetContent["stylesheets"][number];
export type ThemeFile = StylesheetSetContent["assets"][number];
export type VariableSchemaContent =
  components["schemas"]["VariableSchemaContent"];
export type PresetVariable = VariableSchemaContent["variables"][number];
export type ScriptListContent = components["schemas"]["ScriptListContent"];
export type RegexScript = ScriptListContent["scripts"][number];
export type TypedValue = components["schemas"]["TypedValue"];
export type AddableBlock = components["schemas"]["AddableBlock"];
export type ElementType = components["schemas"]["ElementType"];
export type AssetTag = components["schemas"]["AssetTag"];
export type ReadinessItem = components["schemas"]["ReadinessItem"];
export type PreservedNamespace = components["schemas"]["PreservedNamespace"];
export type ProtectionMismatch = components["schemas"]["ProtectionMismatch"];
export type RecordedVersion = components["schemas"]["RecordedVersion"];
export type VersionComparison = components["schemas"]["VersionComparison"];
export type VersionChangeGroup = components["schemas"]["VersionChangeGroup"];
export type VersionChange = components["schemas"]["VersionChange"];
export type IngestOperation = components["schemas"]["IngestOperation"];
export type ReplacementPreview = components["schemas"]["ReplacementPreview"];
export type ReplacementDecision =
  components["schemas"]["ReplacementAcceptance"]["unrepresentable"];
export type DownloadTarget = components["schemas"]["DownloadTarget"];
export type OriginalUpload = components["schemas"]["OriginalUpload"];
export type AssetInstance = components["schemas"]["AssetInstance"];
export type AssetInstanceList = components["schemas"]["AssetInstanceList"];
export type QueuedDelivery = components["schemas"]["QueuedDelivery"];
export type AssetUpdate = components["schemas"]["AssetUpdate"];
export type AssetUpdateRequest = components["schemas"]["AssetUpdateRequest"];
export type PromptCorrespondenceRequest =
  components["schemas"]["PromptCorrespondenceRequest"];
export type Profile = components["schemas"]["Profile"];
export type ProfileLink = components["schemas"]["ProfileLink"];
export type ProfileDistinction = components["schemas"]["ProfileDistinction"];
export type ProfileRestriction = components["schemas"]["ProfileRestriction"];
export type Distinction = components["schemas"]["Distinction"];
export type DistinctionForm = components["schemas"]["DistinctionForm"];
export type DistinctionAssignment =
  components["schemas"]["DistinctionAssignment"];
export type PublicationApp = components["schemas"]["PublicationApp"];
export type PublicationCategory = components["schemas"]["PublicationCategory"];
export type PublicationGrant = components["schemas"]["PublicationGrant"];
export type PublicationWorkspace =
  components["schemas"]["PublicationWorkspace"];
export type PublicationToken = components["schemas"]["PublicationToken"];
export type PublicationDestination =
  components["schemas"]["PublicationDestination"];
export type AddedPublicationDestination =
  components["schemas"]["AddedPublicationDestination"];
export type PublicationDestinationKind =
  components["schemas"]["PublicationDestinationKind"];
export type PublicationChannel = components["schemas"]["PublicationChannel"];
export type PublicationDestinationChoice =
  components["schemas"]["PublicationDestinationChoice"];
export type PublicationDestinationChoiceList =
  components["schemas"]["PublicationDestinationChoiceList"];
export type PostDelivery = components["schemas"]["PostDelivery"];
export type PostDeliveryAttempt = components["schemas"]["PostDeliveryAttempt"];
export type PostDeliveryState = components["schemas"]["PostDeliveryState"];
export type PublicationEvent = components["schemas"]["PublicationEvent"];
export type RotatedPublicationSecret =
  components["schemas"]["RotatedPublicationSecret"];
export type Post = components["schemas"]["Post"];
export type PublicPost = components["schemas"]["PublicPost"];
export type PostMedia = components["schemas"]["PostMedia"];
export type PostMediaPurpose = components["schemas"]["PostMediaPurpose"];
export type PostByline = components["schemas"]["PostByline"];
export type PostRelease = components["schemas"]["PostRelease"];
export type PostSummary = components["schemas"]["PostSummary"];
export type PostArchive = components["schemas"]["PostArchive"];
export type PostRevision = components["schemas"]["PostRevision"];
export type PostAction = components["schemas"]["PostAction"];
export type PostSchedule = components["schemas"]["PostSchedule"];
export type PostWithdrawal = components["schemas"]["PostWithdrawal"];
export type PostDeletion = components["schemas"]["PostDeletion"];
export type IssuedPublicationToken =
  components["schemas"]["IssuedPublicationToken"];
export type BrowseAsset = components["schemas"]["BrowseAsset"];
export type BrowsePage = components["schemas"]["AssetList"];
export type BrowseCursor = components["schemas"]["BrowseCursor"];
export type DeletedAsset = components["schemas"]["DeletedAsset"];
export type LegacyAsset = components["schemas"]["LegacyAsset"];
export type BrowseKind = BrowseAsset["kind"];
export type NsfwVisibility =
  components["schemas"]["NsfwVisibilityRequest"]["visibility"];

type AssetQuery = NonNullable<
  paths["/v1/assets"]["get"]["parameters"]["query"]
>;

export type BrowseFilters = Pick<
  AssetQuery,
  "kind" | "platform" | "q" | "facet"
>;

export type AssetListParams = BrowseFilters &
  Pick<AssetQuery, "creator" | "limit" | "before" | "beforeId" | "nsfw">;

/** A fresh client every call. A shared one would leak one visitor's cache into another's page. */
export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { staleTime: 30_000, gcTime: 10 * 60_000 },
    },
  });
}

export const assetKeys = {
  all: ["assets"] as const,
  list: (
    filters: BrowseFilters,
    visibility?: NsfwVisibility,
    creator?: string,
  ) => ["assets", "list", creator, filters, visibility] as const,
};

/** Every working-copy write refuses the same way, and a stale one tells the page. */
function writeRefusal(error: unknown, fallback: string): Error {
  reportStaleWorkingCopy(error);
  const detail = error as { error?: unknown } | undefined;
  return new Error(
    typeof detail?.error === "string"
      ? detail.error.replace(/^invalid block:\s*/i, "")
      : fallback,
  );
}

export async function fetchProfile(handle: string): Promise<Profile | null> {
  const { data, error } = await api.GET("/v1/profiles/{handle}", {
    params: { path: { handle } },
  });
  if (error || !data) return null;
  return data;
}

export async function fetchAssets(
  params: AssetListParams,
  cookie?: string,
): Promise<BrowsePage> {
  const { data, error } = await api.GET("/v1/assets", {
    params: { query: params },
    headers: cookie ? { cookie } : undefined,
  });
  if (error) throw new Error("Could not load the collection");
  return data;
}

export async function fetchLegacyAsset(
  author: string,
  name: string,
): Promise<LegacyAsset | null> {
  const { data, error } = await api.GET("/v1/legacy-assets/{author}/{name}", {
    params: { path: { author, name } },
  });
  if (error || !data) return null;
  return data;
}

export async function fetchLegacyProfile(
  discordId: string,
): Promise<Profile | null> {
  const { data, error } = await api.GET("/v1/legacy-profiles/{discordId}", {
    params: { path: { discordId } },
  });
  if (error || !data) return null;
  return data;
}

export async function fetchDeletedAssets(
  handle: string,
  cookie: string,
): Promise<DeletedAsset[] | null> {
  const { data, error } = await api.GET("/v1/profiles/{handle}/deleted", {
    params: { path: { handle } },
    headers: { cookie },
  });
  if (error || !data) return null;
  return data.items;
}

/** The API intentionally makes withheld, deleted, and missing assets identical. */
export async function fetchAsset(
  id: string,
  cookie?: string,
  workingCopy = false,
): Promise<AssetDetail | null> {
  const { data, error } = await api.GET("/v1/assets/{id}", {
    params: { path: { id }, query: { workingCopy } },
    headers: cookie ? { cookie } : undefined,
  });
  if (error || !data) return null;
  return data;
}

export type StartAssetApp = NonNullable<
  components["schemas"]["StartAssetRequest"]["app"]
>;

export async function startAsset(
  kind: string,
  app?: StartAssetApp,
): Promise<AssetDetail> {
  const { data, error } = await api.POST("/v1/assets", {
    body: app ? { kind, app } : { kind },
  });
  /** The upload variant returns an ingest operation, so require a page here. */
  if (error || !data || !("blocks" in data)) {
    throw new Error("Could not start the asset");
  }
  return data;
}

export async function saveNsfwVisibility(visibility: NsfwVisibility) {
  const { error } = await api.PUT("/v1/account/nsfw-visibility", {
    body: { visibility },
  });
  if (error) throw new Error("Could not save the content preference");
}

export async function saveAssetDiscovery(
  id: string,
  discovery: AssetDetail["discovery"],
) {
  const { error } = await api.PUT("/v1/assets/{id}/discovery", {
    params: { path: { id } },
    body: { discovery },
  });
  if (error) throw new Error("Could not save discovery");
}

/** Saves one builder block without changing any other block on the page. */
export async function saveAssetBlock(
  candidate: Candidate,
  assetId: string,
  blockId: string,
  block: SaveAssetBlockRequest,
): Promise<AssetBlock> {
  const { data, error, response } = await api.PUT(
    "/v1/assets/{id}/blocks/{blockId}",
    {
      params: {
        header: { "X-Working-Copy-Version": candidate.version },
        path: { id: assetId, blockId },
      },
      body: block,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The block could not be saved. Try again.");
  }
  return data;
}

/** Adds one block to the foot of the page, holding the element chosen for it. */
export async function addAssetBlock(
  candidate: Candidate,
  assetId: string,
  definition: string,
  elementType: ElementType,
): Promise<AssetBlock> {
  const { data, error, response } = await api.POST("/v1/assets/{id}/blocks", {
    params: {
      header: { "X-Working-Copy-Version": candidate.version },
      path: { id: assetId },
    },
    body: { definition, elementType },
  });
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The block could not be added. Try again.");
  }
  return data;
}

/** Media is stored first, then linked when its block is saved. */
export async function addAssetImage(
  candidate: Candidate,
  assetId: string,
  file: File,
  role: components["schemas"]["AddMediaRequest"]["role"],
): Promise<string> {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ role }));
  body.append("file", file, file.name);
  const response = await browserFetch(`/api/v1/assets/${assetId}/media`, {
    method: "POST",
    headers: { "X-Working-Copy-Version": String(candidate.version) },
    credentials: "same-origin",
    body,
  });
  if (!response.ok) {
    throw new Error(
      response.status === 413
        ? "That image is larger than Illarin accepts."
        : "The image could not be added. Try again.",
    );
  }
  acceptCandidateVersion(candidate, response);
  const added = (await response.json()) as { id?: unknown };
  if (typeof added.id !== "string") {
    throw new Error("The image could not be added. Try again.");
  }
  return added.id;
}

/** Saves the full page outline as one arrangement. */
export async function arrangeAssetBlocks(
  candidate: Candidate,
  assetId: string,
  arrangement: ArrangeAssetBlocksRequest,
): Promise<AssetBlock[]> {
  const { data, error, response } = await api.PUT("/v1/assets/{id}/blocks", {
    params: {
      header: { "X-Working-Copy-Version": candidate.version },
      path: { id: assetId },
    },
    body: arrangement,
  });
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The block order could not be saved. Try again.");
  }
  return data;
}

/** Removes one optional block and all of the elements it holds. */
export async function removeAssetBlock(
  candidate: Candidate,
  assetId: string,
  blockId: string,
) {
  const { error, response } = await api.DELETE(
    "/v1/assets/{id}/blocks/{blockId}",
    {
      params: {
        header: { "X-Working-Copy-Version": candidate.version },
        path: { id: assetId, blockId },
      },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The block could not be removed. Try again.");
  }
}

/** Moves a block's unpinned content, then removes the emptied block. */
export async function moveAssetBlockContent(
  candidate: Candidate,
  assetId: string,
  blockId: string,
  destinationBlockId: string,
): Promise<AssetBlock[]> {
  const { data, error, response } = await api.POST(
    "/v1/assets/{id}/blocks/{blockId}/move-and-remove",
    {
      params: {
        header: { "X-Working-Copy-Version": candidate.version },
        path: { id: assetId, blockId },
      },
      body: { destinationBlockId },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The content could not be moved. Try again.");
  }
  return data;
}

/** A null adult-content answer is allowed only while the asset is a draft. */
export async function saveAssetIdentity(
  candidate: Candidate,
  id: string,
  identity: { name: string; isNsfw: boolean | null },
) {
  const { error, response } = await api.PUT("/v1/assets/{id}/identity", {
    params: {
      header: { "X-Working-Copy-Version": candidate.version },
      path: { id },
    },
    body: identity,
  });
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The details could not be saved. Try again.");
  }
}

/**
 * Publishes a draft, or comes back with what publication is still waiting on.
 */
export async function publishAsset(
  candidate: Candidate,
  id: string,
): Promise<
  | { published: true }
  | { published: false; error: string; readiness?: ReadinessItem[] }
> {
  const { data, error, response } = await api.POST("/v1/assets/{id}/publish", {
    params: {
      header: { "X-Working-Copy-Version": candidate.version },
      path: { id },
    },
  });
  acceptCandidateVersion(candidate, response);
  if (data) return { published: true };
  const refusal = error as
    | { error?: unknown; readiness?: ReadinessItem[] }
    | undefined;
  return {
    published: false,
    error:
      typeof refusal?.error === "string"
        ? refusal.error
        : "The asset could not be published. Try again.",
    readiness: refusal?.readiness,
  };
}

/** The replacement this asset is still deciding about, or nothing where none waits. */
export async function fetchWaitingReplacement(
  id: string,
): Promise<IngestOperation | null> {
  const { data, error } = await api.GET("/v1/assets/{id}/revisions", {
    params: { path: { id } },
  });
  if (error || !data) return null;
  return data;
}

/** Hands over a replacement file. Nothing readers see changes until it is accepted. */
export async function uploadAssetReplacement(
  candidate: Candidate,
  id: string,
  file: File,
): Promise<IngestOperation> {
  const body = new FormData();
  body.append("file", file, file.name);
  const response = await browserFetch(`/api/v1/assets/${id}/revisions`, {
    method: "POST",
    headers: { "X-Working-Copy-Version": String(candidate.version) },
    credentials: "same-origin",
    body,
  });
  acceptCandidateVersion(candidate, response);
  const answer = (await response.json()) as IngestOperation & {
    error?: unknown;
    code?: unknown;
  };
  if (!response.ok) {
    throw writeRefusal(
      answer,
      response.status === 413
        ? "That file is larger than Illarin accepts."
        : "That file could not be accepted. Try again.",
    );
  }
  return answer;
}

/** Reads one ingest operation again while it is being processed. */
export async function readIngestOperation(
  url: string,
): Promise<IngestOperation> {
  const response = await fetch(`/api${url}`, {
    credentials: "same-origin",
    cache: "no-store",
  });
  if (!response.ok) throw new Error("Illarin could not read this upload yet.");
  return (await response.json()) as IngestOperation;
}

/** Applies a reviewed replacement to the working copy, with a choice for each thing the file cannot hold. */
export async function acceptAssetReplacement(
  candidate: Candidate,
  id: string,
  operationId: string,
  unrepresentable: ReplacementDecision,
): Promise<IngestOperation> {
  const { data, error, response } = await api.POST(
    "/v1/assets/{id}/revisions/{operationId}/accept",
    {
      params: {
        header: { "X-Working-Copy-Version": candidate.version },
        path: { id, operationId },
      },
      body: { unrepresentable },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "That file could not be applied. Try again.");
  }
  return data;
}

/** Discards a reviewed replacement and leaves the working copy as it was. */
export async function cancelAssetReplacement(id: string, operationId: string) {
  const { error } = await api.DELETE(
    "/v1/assets/{id}/revisions/{operationId}",
    { params: { path: { id, operationId } } },
  );
  if (error) throw new Error("That file could not be discarded. Try again.");
}

/** Publishes the reviewed working copy, or says what publication is waiting on. */
export async function publishAssetUpdate(
  candidate: Candidate,
  id: string,
  update: AssetUpdateRequest,
): Promise<
  | { published: true; update: AssetUpdate }
  | {
      published: false;
      error: string;
      code?: string;
      readiness?: ReadinessItem[];
    }
> {
  const { data, error, response } = await api.POST("/v1/assets/{id}/updates", {
    params: {
      header: { "X-Working-Copy-Version": candidate.version },
      path: { id },
    },
    body: update,
  });
  acceptCandidateVersion(candidate, response);
  if (data) return { published: true, update: data };
  reportStaleWorkingCopy(error);
  const refusal = error as
    | { error?: unknown; code?: unknown; readiness?: ReadinessItem[] }
    | undefined;
  return {
    published: false,
    error:
      typeof refusal?.error === "string"
        ? refusal.error
        : "The update could not be published. Try again.",
    code: typeof refusal?.code === "string" ? refusal.code : undefined,
    readiness: refusal?.readiness,
  };
}

/** Preserved namespaces belong to the source file and are owner-only. */
export async function fetchPreservedNamespaces(
  id: string,
): Promise<PreservedNamespace[]> {
  const { data } = await api.GET("/v1/assets/{id}/preserved", {
    params: { path: { id } },
  });
  return data ?? [];
}

/** The versions an asset has recorded, newest first. */
export async function fetchAssetUpdates(
  id: string,
  cookie?: string,
): Promise<RecordedVersion[]> {
  const { data } = await api.GET("/v1/assets/{id}/updates", {
    params: { path: { id } },
    headers: cookie ? { cookie } : undefined,
  });
  return data?.items ?? [];
}

/** What changed between two recorded versions, or why the reader cannot see it. */
export async function compareAssetVersions(
  id: string,
  from: number,
  to: number,
  cookie?: string,
): Promise<
  { compared: VersionComparison } | { compared: null; refusal: string }
> {
  const unreadable = {
    compared: null,
    refusal: "That comparison could not be read. Try again.",
  } as const;
  let data: VersionComparison | undefined;
  let response: Response;
  try {
    ({ data, response } = await api.GET("/v1/assets/{id}/updates/comparison", {
      params: { path: { id }, query: { from, to } },
      headers: cookie ? { cookie } : undefined,
    }));
  } catch {
    return unreadable;
  }
  if (data) return { compared: data };
  if (response.status === 409) {
    return {
      compared: null,
      refusal:
        "This is the first version Illarin recorded, so there is nothing before it to compare.",
    };
  }
  if (response.status === 404) {
    return {
      compared: null,
      refusal: "That version is not available to read.",
    };
  }
  return unreadable;
}

/** The recorded versions whose prompts no longer line up with the sealed ones. */
export async function fetchProtectionMismatches(
  id: string,
): Promise<ProtectionMismatch[]> {
  const { data } = await api.GET("/v1/assets/{id}/updates/protection", {
    params: { path: { id } },
  });
  return data?.items ?? [];
}

/** Says which recorded prompt each sealed prompt is, on one recorded version. */
export async function resolvePromptCorrespondence(
  id: string,
  number: number,
  matches: PromptCorrespondenceRequest["matches"],
) {
  const { error } = await api.PUT(
    "/v1/assets/{id}/updates/{number}/protection",
    {
      params: { path: { id, number } },
      body: { matches },
    },
  );
  if (error) {
    const refusal = error as { error?: unknown } | undefined;
    throw new Error(
      typeof refusal?.error === "string"
        ? refusal.error
        : "That version could not be settled. Try again.",
    );
  }
}

/** Deletes one namespace and everything under it, for good. */
export async function deletePreservedNamespace(
  candidate: Candidate,
  id: string,
  namespace: string,
) {
  const { error, response } = await api.DELETE(
    "/v1/assets/{id}/preserved/{namespace}",
    {
      params: {
        header: { "X-Working-Copy-Version": candidate.version },
        path: { id, namespace },
      },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "That data could not be deleted. Try again.");
  }
}

export async function fetchPublishedPost(
  slug: string,
): Promise<PublicPost | null> {
  const { data, error } = await api.GET("/v1/posts/{slug}", {
    params: { path: { slug } },
  });
  if (error || !data) return null;
  return data;
}

export async function fetchPostArchive(query: {
  page?: number;
  category?: string;
  app?: string;
}): Promise<PostArchive | null> {
  const { data, error, response } = await api.GET("/v1/posts", {
    params: { query },
  });
  if (response.status === 404) return null;
  if (error || !data)
    throw new Error("Could not read the publication archive.");
  return data;
}

export async function fetchPostCategories(): Promise<PublicationCategory[]> {
  const { data, error } = await api.GET("/v1/post-categories", {});
  if (error || !data) return [];
  return data.categories;
}

export async function fetchPostApps(): Promise<PublicationApp[]> {
  const { data, error } = await api.GET("/v1/post-apps", {});
  if (error || !data) return [];
  return data.apps;
}

export async function fetchProfileRestriction(
  handle: string,
): Promise<ProfileRestriction | null> {
  const { data, error } = await api.GET("/v1/profiles/{handle}/restriction", {
    params: { path: { handle } },
  });
  if (error || !data) return null;
  return data;
}

export async function restrictProfile(handle: string, reason: string) {
  const { data, error } = await api.PUT("/v1/profiles/{handle}/restriction", {
    params: { path: { handle } },
    body: { reason },
  });
  if (error || !data) throw new Error("Could not restrict the profile");
  return data;
}

export async function restoreProfile(handle: string) {
  const { error } = await api.DELETE("/v1/profiles/{handle}/restriction", {
    params: { path: { handle } },
  });
  if (error) throw new Error("Could not restore the profile");
}

export async function withholdAsset(id: string, reason: string) {
  const { error } = await api.PUT("/v1/assets/{id}/withhold", {
    params: { path: { id } },
    body: { reason },
  });
  if (error) throw new Error("Could not withhold the asset");
}

export async function deleteAsset(id: string) {
  const { error } = await api.DELETE("/v1/assets/{id}", {
    params: { path: { id } },
  });
  if (error) throw new Error("Could not delete the asset");
}

export async function restoreAsset(id: string) {
  const { error } = await api.POST("/v1/assets/{id}/restore", {
    params: { path: { id } },
  });
  if (error) throw new Error("Could not restore the asset");
}
