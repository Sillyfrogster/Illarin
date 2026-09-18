import { QueryClient } from "@tanstack/react-query";
import { readRefusal } from "@/lib/answer";
import {
  acceptCandidateVersion,
  type Candidate,
  reportStaleWorkingCopy,
} from "@/lib/working-copy";
import { api } from "./client";
import type {
  AddableBlock,
  AddedPublicationDestination,
  AddMediaRequest,
  AppTarget,
  ArrangeWorkBlocksRequest,
  BrowseCursor,
  BrowseWork,
  ColorSetContent,
  DeletedWork,
  DeletedWorkList,
  DownloadTarget,
  ElementType,
  EntryTableContent,
  ExtensionDependency,
  IngestOperation,
  ListWorksParams,
  NsfwPreferenceRequest,
  OriginalUpload,
  Post,
  PostAction,
  PostArchive,
  PostByline,
  PostDeletion,
  PostDelivery,
  PostDeliveryAttempt,
  PostDeliveryState,
  PostMedia,
  PostMediaPurpose,
  PostRelease,
  PostRevision,
  PostSchedule,
  PostSummary,
  PostWithdrawal,
  PreservedNamespace,
  Profile,
  ProfileLink,
  ProfileRestriction,
  PromptCorrespondenceRequest,
  PromptListContent,
  ProtectionMismatch,
  ProtectionMismatchList,
  PublicationApp,
  PublicationAppList,
  PublicationCategory,
  PublicationCategoryList,
  PublicationChannel,
  PublicationDestination,
  PublicationDestinationChoice,
  PublicationDestinationChoiceList,
  PublicationDestinationType,
  PublicationEvent,
  PublicationGrant,
  PublicationWorkspace,
  PublicPost,
  QueuedDelivery,
  ReadinessItem,
  RecordedVersion,
  RecordedVersionDownloads,
  RecordedVersionList,
  RecordListContent,
  ReplacementAcceptance,
  ReplacementPreview,
  RotatedPublicationSecret,
  SaveWorkBlockRequest,
  ScriptListContent,
  SettingGroupContent,
  StartWorkRequest,
  StylesheetSetContent,
  TypedValue,
  VariableSchemaContent,
  VaultPicture,
  VaultPictureList,
  VersionChange,
  VersionChangeGroup,
  VersionComparison,
  WorkBlock,
  WorkDetail,
  WorkElement,
  WorkIdentityRequest,
  WorkImage,
  WorkInstance,
  WorkInstanceList,
  WorkList,
  WorkTag,
  WorkUpdate,
  WorkUpdateRequest,
  WorkVersionNotesRequest,
} from "./shapes";
export type {
  AddableBlock,
  AddedPublicationDestination,
  AppTarget,
  ArrangeWorkBlocksRequest,
  WorkBlock,
  WorkDetail,
  WorkElement,
  WorkIdentityRequest,
  WorkImage,
  WorkInstance,
  WorkInstanceList,
  WorkTag,
  WorkUpdate,
  WorkUpdateRequest,
  WorkVersionNotesRequest,
  BrowseWork,
  BrowseCursor,
  ColorSetContent,
  DeletedWork,
  DownloadTarget,
  ElementType,
  EntryTableContent,
  ExtensionDependency,
  IngestOperation,
  OriginalUpload,
  Post,
  PostAction,
  PostArchive,
  PostByline,
  PostDeletion,
  PostDelivery,
  PostDeliveryAttempt,
  PostDeliveryState,
  PostMedia,
  PostMediaPurpose,
  PostRelease,
  PostRevision,
  PostSchedule,
  PostSummary,
  PostWithdrawal,
  PreservedNamespace,
  Profile,
  ProfileLink,
  ProfileRestriction,
  PromptCorrespondenceRequest,
  PromptListContent,
  ProtectionMismatch,
  PublicPost,
  PublicationApp,
  PublicationCategory,
  PublicationChannel,
  PublicationDestination,
  PublicationDestinationChoice,
  PublicationDestinationChoiceList,
  PublicationDestinationType,
  PublicationEvent,
  PublicationGrant,
  PublicationWorkspace,
  QueuedDelivery,
  ReadinessItem,
  RecordListContent,
  RecordedVersion,
  RecordedVersionDownloads,
  ReplacementPreview,
  RotatedPublicationSecret,
  SaveWorkBlockRequest,
  ScriptListContent,
  SettingGroupContent,
  StylesheetSetContent,
  TypedValue,
  VariableSchemaContent,
  VaultPicture,
  VersionChange,
  VersionChangeGroup,
  VersionComparison,
};

export type LorebookEntry = EntryTableContent["entries"][number];
export type LumiaRecord = RecordListContent["records"][number];
export type PromptGroup = PromptListContent["groups"][number];
export type PromptFragment = PromptListContent["fragments"][number];
export type PresetSetting = SettingGroupContent["settings"][number];
export type ThemeColorMode = ColorSetContent["modes"][number];
export type ThemeColor = ThemeColorMode["colors"][number];
export type ThemeStylesheet = StylesheetSetContent["stylesheets"][number];
export type ThemeFile = StylesheetSetContent["assets"][number];
export type PresetVariable = VariableSchemaContent["variables"][number];
export type RegexScript = ScriptListContent["scripts"][number];
export type ReplacementDecision = ReplacementAcceptance["unrepresentable"];
export type BrowsePage = WorkList;
export type BrowseKind = BrowseWork["kind"];
export type NsfwVisibility = NsfwPreferenceRequest["visibility"];

export type BrowseFilters = Pick<
  ListWorksParams,
  "kind" | "platform" | "q" | "facet"
>;

export type AssetListParams = BrowseFilters &
  Pick<ListWorksParams, "creator" | "limit" | "before" | "beforeId" | "nsfw">;

/** Creates an isolated cache for each server render. */
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

export class SealedExposureError extends Error {
  constructor(
    message: string,
    readonly prompts: string[],
  ) {
    super(message);
  }
}

function writeRefusal(error: unknown, fallback: string): Error {
  reportStaleWorkingCopy(error);
  const detail = error as
    | { error?: unknown; code?: unknown; prompts?: unknown }
    | undefined;
  if (
    detail?.code === "sealed_exposure" &&
    Array.isArray(detail.prompts) &&
    detail.prompts.every((prompt) => typeof prompt === "string")
  ) {
    return new SealedExposureError(
      typeof detail.error === "string" ? detail.error : fallback,
      detail.prompts,
    );
  }
  return new Error(
    typeof detail?.error === "string"
      ? detail.error.replace(/^invalid block:\s*/i, "")
      : fallback,
  );
}

export async function fetchProfile(handle: string): Promise<Profile | null> {
  const { data, error } = await api<Profile>(
    "GET",
    `/v1/profiles/${encodeURIComponent(handle)}`,
  );
  if (error || !data) return null;
  return data;
}

export async function fetchAssets(
  params: AssetListParams,
  cookie?: string,
  signal?: AbortSignal,
): Promise<BrowsePage> {
  const { data, error } = await api<WorkList>("GET", "/v1/assets", {
    query: params,
    headers: cookie ? { cookie } : undefined,
    signal,
  });
  if (error || !data) throw new Error("Could not load the catalog");
  return data;
}

export async function fetchDeletedAssets(
  handle: string,
  cookie: string,
): Promise<DeletedWork[] | null> {
  const { data, error } = await api<DeletedWorkList>(
    "GET",
    `/v1/profiles/${encodeURIComponent(handle)}/deleted`,
    { headers: { cookie } },
  );
  if (error || !data) return null;
  return data.items;
}

export async function fetchAsset(
  id: string,
  cookie?: string,
  workingCopy = false,
): Promise<WorkDetail | null> {
  const { data, error } = await api<WorkDetail>("GET", `/v1/assets/${id}`, {
    query: { workingCopy },
    headers: cookie ? { cookie } : undefined,
  });
  if (error || !data) return null;
  return data;
}

export type StartAssetApp = NonNullable<StartWorkRequest["app"]>;

export async function startAsset(
  kind: string,
  app?: StartAssetApp,
): Promise<WorkDetail> {
  const { data, error } = await api<WorkDetail>("POST", "/v1/assets", {
    body: app ? { kind, app } : { kind },
  });
  if (error || !data || !("blocks" in data)) {
    throw new Error("Could not start the asset");
  }
  return data;
}

export async function saveNsfwVisibility(visibility: NsfwVisibility) {
  const { error } = await api<void>("PUT", "/v1/account/nsfw-visibility", {
    body: { visibility },
  });
  if (error) throw new Error("Could not save the content preference");
}

export async function saveAssetDiscovery(
  id: string,
  discovery: WorkDetail["discovery"],
) {
  const { error } = await api<void>("PUT", `/v1/assets/${id}/discovery`, {
    body: { discovery },
  });
  if (error) throw new Error("Could not save the catalog listing");
}

export async function saveAssetBlock(
  candidate: Candidate,
  assetId: string,
  blockId: string,
  block: SaveWorkBlockRequest,
): Promise<WorkBlock> {
  const { data, error, response } = await api<WorkBlock>(
    "PUT",
    `/v1/assets/${assetId}/blocks/${blockId}`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: block,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The block could not be saved. Try again.");
  }
  return data;
}

export async function addAssetBlock(
  candidate: Candidate,
  assetId: string,
  definition: string,
  elementType: ElementType,
): Promise<WorkBlock> {
  const { data, error, response } = await api<WorkBlock>(
    "POST",
    `/v1/assets/${assetId}/blocks`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: { definition, elementType },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The block could not be added. Try again.");
  }
  return data;
}

export async function addAssetImage(
  candidate: Candidate,
  assetId: string,
  file: File,
  role: AddMediaRequest["role"],
): Promise<string> {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ role }));
  body.append("file", file, file.name);
  const { data: added, response } = await api<{ id?: unknown }>(
    "POST",
    `/v1/assets/${assetId}/media`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body,
    },
  );
  if (!response.ok) {
    throw new Error(
      response.status === 413
        ? "That image is larger than Illarin accepts."
        : "The image could not be added. Try again.",
    );
  }
  acceptCandidateVersion(candidate, response);
  if (typeof added?.id !== "string") {
    throw new Error("The image could not be added. Try again.");
  }
  return added.id;
}

export async function arrangeAssetBlocks(
  candidate: Candidate,
  assetId: string,
  arrangement: ArrangeWorkBlocksRequest,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "PUT",
    `/v1/assets/${assetId}/blocks`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: arrangement,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The block order could not be saved. Try again.");
  }
  return data;
}

/** The pictures a README showed, waiting for the creator to place or let go. */
export async function fetchVault(assetId: string): Promise<VaultPicture[]> {
  const { data, error } = await api<VaultPictureList>(
    "GET",
    `/v1/assets/${assetId}/vault`,
  );
  if (error || !data) {
    throw new Error("The waiting pictures could not be read. Try again.");
  }
  return data.pictures;
}

export async function placeVaultPicture(
  candidate: Candidate,
  assetId: string,
  pictureId: string,
  mediaId?: string,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "POST",
    `/v1/assets/${assetId}/vault/${pictureId}/place`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: mediaId ? { mediaId } : undefined,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The picture could not be placed. Try again.");
  }
  return data;
}

export async function discardVaultPicture(
  candidate: Candidate,
  assetId: string,
  pictureId: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/assets/${assetId}/vault/${pictureId}`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The picture could not be discarded. Try again.");
  }
}

export async function removeAssetBlock(
  candidate: Candidate,
  assetId: string,
  blockId: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/assets/${assetId}/blocks/${blockId}`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The block could not be removed. Try again.");
  }
}

export async function moveAssetBlockContent(
  candidate: Candidate,
  assetId: string,
  blockId: string,
  destinationBlockId: string,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "POST",
    `/v1/assets/${assetId}/blocks/${blockId}/move-and-remove`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: { destinationBlockId },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The content could not be moved. Try again.");
  }
  return data;
}

export async function saveAssetIdentity(
  candidate: Candidate,
  id: string,
  identity: WorkIdentityRequest,
) {
  const { error, response } = await api<void>(
    "PUT",
    `/v1/assets/${id}/identity`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: identity,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The details could not be saved. Try again.");
  }
}

export async function publishAsset(
  candidate: Candidate,
  id: string,
): Promise<
  | { published: true }
  | { published: false; error: string; readiness?: ReadinessItem[] }
> {
  const { data, error, response } = await api<WorkDetail>(
    "POST",
    `/v1/assets/${id}/publish`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
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

export async function fetchWaitingReplacement(
  id: string,
): Promise<IngestOperation | null> {
  const { data, error } = await api<IngestOperation | null>(
    "GET",
    `/v1/assets/${id}/revisions`,
  );
  if (error || !data) return null;
  return data;
}

export async function uploadAssetReplacement(
  candidate: Candidate,
  id: string,
  file: File,
): Promise<IngestOperation> {
  const body = new FormData();
  body.append("file", file, file.name);
  const { data, error, response } = await api<IngestOperation>(
    "POST",
    `/v1/assets/${id}/revisions`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (!data) {
    throw writeRefusal(
      readRefusal(error),
      response.status === 413
        ? "That file is larger than Illarin accepts."
        : "That file could not be accepted. Try again.",
    );
  }
  return data;
}

export async function readIngestOperation(
  url: string,
): Promise<IngestOperation> {
  const { data } = await api<IngestOperation>("GET", url, {
    cache: "no-store",
  });
  if (!data) throw new Error("Illarin could not read this upload yet.");
  return data;
}

export async function acceptAssetReplacement(
  candidate: Candidate,
  id: string,
  operationId: string,
  unrepresentable: ReplacementDecision,
  exposeProtected = false,
): Promise<IngestOperation> {
  const { data, error, response } = await api<IngestOperation>(
    "POST",
    `/v1/assets/${id}/revisions/${operationId}/accept`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: { unrepresentable, exposeProtected },
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "That file could not be applied. Try again.");
  }
  return data;
}

export async function cancelAssetReplacement(id: string, operationId: string) {
  const { error } = await api<void>(
    "DELETE",
    `/v1/assets/${id}/revisions/${operationId}`,
  );
  if (error) throw new Error("That file could not be discarded. Try again.");
}

export async function publishAssetUpdate(
  candidate: Candidate,
  id: string,
  update: WorkUpdateRequest,
): Promise<
  | { published: true; update: WorkUpdate }
  | {
      published: false;
      error: string;
      code?: string;
      field?: string;
      readiness?: ReadinessItem[];
    }
> {
  const { data, error, response } = await api<WorkUpdate>(
    "POST",
    `/v1/assets/${id}/updates`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: update,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (data) return { published: true, update: data };
  reportStaleWorkingCopy(error);
  const refusal = error as
    | {
        error?: unknown;
        code?: unknown;
        field?: unknown;
        readiness?: ReadinessItem[];
      }
    | undefined;
  return {
    published: false,
    error:
      typeof refusal?.error === "string"
        ? refusal.error
        : "The update could not be published. Try again.",
    code: typeof refusal?.code === "string" ? refusal.code : undefined,
    field: typeof refusal?.field === "string" ? refusal.field : undefined,
    readiness: refusal?.readiness,
  };
}

export async function fetchPreservedNamespaces(
  id: string,
): Promise<PreservedNamespace[]> {
  const { data } = await api<PreservedNamespace[]>(
    "GET",
    `/v1/assets/${id}/preserved`,
  );
  return data ?? [];
}

export async function fetchAssetUpdates(
  id: string,
): Promise<RecordedVersion[] | null> {
  try {
    const { data } = await api<RecordedVersionList>(
      "GET",
      `/v1/assets/${id}/updates`,
    );
    return data?.items ?? null;
  } catch {
    return null;
  }
}

export async function restoreAssetVersion(
  candidate: Candidate,
  id: string,
  number: number,
) {
  const { error, response } = await api<void>(
    "POST",
    `/v1/assets/${id}/updates/${number}/restore`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) throw writeRefusal(error, "That version could not be restored.");
}

export async function correctAssetVersionNotes(
  id: string,
  number: number,
  correction: WorkVersionNotesRequest,
) {
  const { error } = await api<void>(
    "PATCH",
    `/v1/assets/${id}/updates/${number}/notes`,
    { body: correction },
  );
  if (error) throw writeRefusal(error, "Those notes could not be corrected.");
}

export async function withdrawAssetVersion(
  id: string,
  number: number,
  explanation: string,
) {
  const { error } = await api<void>(
    "POST",
    `/v1/assets/${id}/updates/${number}/withdraw`,
    { body: { explanation } },
  );
  if (error) throw writeRefusal(error, "That version could not be withdrawn.");
}

/** Loads the formats and media available for a historical download. */
export async function fetchRecordedVersionDownloads(
  id: string,
  number: number,
): Promise<
  | { offered: RecordedVersionDownloads }
  | { offered: null; refusal: string; retry: boolean }
> {
  const unreadable = {
    offered: null,
    refusal: "Illarin could not load this version's download options.",
    retry: true,
  } as const;
  let data: RecordedVersionDownloads | undefined;
  let response: Response;
  try {
    ({ data, response } = await api<RecordedVersionDownloads>(
      "GET",
      `/v1/assets/${id}/updates/${number}/downloads`,
    ));
  } catch {
    return unreadable;
  }
  if (data) return { offered: data };
  if (response.status === 404) {
    return {
      offered: null,
      refusal: "This version is not available to download.",
      retry: false,
    };
  }
  return unreadable;
}

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
    ({ data, response } = await api<VersionComparison>(
      "GET",
      `/v1/assets/${id}/updates/comparison`,
      { query: { from, to }, headers: cookie ? { cookie } : undefined },
    ));
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

export async function fetchProtectionMismatches(
  id: string,
): Promise<ProtectionMismatch[]> {
  const { data } = await api<ProtectionMismatchList>(
    "GET",
    `/v1/assets/${id}/updates/protection`,
  );
  return data?.items ?? [];
}

export async function resolvePromptCorrespondence(
  id: string,
  number: number,
  matches: PromptCorrespondenceRequest["matches"],
) {
  const { error } = await api<void>(
    "PUT",
    `/v1/assets/${id}/updates/${number}/protection`,
    { body: { matches } },
  );
  if (error) {
    const refusal = error as { error?: unknown } | undefined;
    throw new Error(
      typeof refusal?.error === "string"
        ? refusal.error
        : "The prompt matches could not be saved. Try again.",
    );
  }
}

export async function deletePreservedNamespace(
  candidate: Candidate,
  id: string,
  namespace: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/assets/${id}/preserved/${encodeURIComponent(namespace)}`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "That data could not be deleted. Try again.");
  }
}

export async function fetchPublishedPost(
  slug: string,
): Promise<PublicPost | null> {
  const { data, error } = await api<PublicPost>(
    "GET",
    `/v1/posts/${encodeURIComponent(slug)}`,
  );
  if (error || !data) return null;
  return data;
}

export async function fetchPostArchive(query: {
  page?: number;
  category?: string;
  app?: string;
}): Promise<PostArchive | null> {
  const { data, error, response } = await api<PostArchive>("GET", "/v1/posts", {
    query,
  });
  if (response.status === 404) return null;
  if (error || !data) throw new Error("Could not load blog posts.");
  return data;
}

export async function fetchPostCategories(): Promise<PublicationCategory[]> {
  const answer = await api<PublicationCategoryList>(
    "GET",
    "/v1/post-categories",
  ).catch(() => null);
  return answer?.data?.categories ?? [];
}

export async function fetchPostApps(): Promise<PublicationApp[]> {
  const { data, error } = await api<PublicationAppList>("GET", "/v1/post-apps");
  if (error || !data) return [];
  return data.apps;
}

export async function fetchProfileRestriction(
  handle: string,
): Promise<ProfileRestriction | null> {
  const { data, error } = await api<ProfileRestriction>(
    "GET",
    `/v1/profiles/${encodeURIComponent(handle)}/restriction`,
  );
  if (error || !data) return null;
  return data;
}

export async function restrictProfile(handle: string, reason: string) {
  const { data, error } = await api<ProfileRestriction>(
    "PUT",
    `/v1/profiles/${encodeURIComponent(handle)}/restriction`,
    { body: { reason } },
  );
  if (error || !data) throw new Error("Could not restrict the profile");
  return data;
}

export async function restoreProfile(handle: string) {
  const { error } = await api<void>(
    "DELETE",
    `/v1/profiles/${encodeURIComponent(handle)}/restriction`,
  );
  if (error) throw new Error("Could not restore the profile");
}

export async function withholdAsset(id: string, reason: string) {
  const { error } = await api<void>("PUT", `/v1/assets/${id}/withhold`, {
    body: { reason },
  });
  if (error) throw new Error("Could not withhold the asset");
}

export async function deleteAsset(id: string) {
  const { error } = await api<void>("DELETE", `/v1/assets/${id}`);
  if (error) throw new Error("Could not delete the asset");
}

export async function restoreAsset(id: string) {
  const { error } = await api<void>("POST", `/v1/assets/${id}/restore`);
  if (error) throw new Error("Could not restore the asset");
}
