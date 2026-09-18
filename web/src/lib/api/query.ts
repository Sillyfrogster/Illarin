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
  WorkDetailsRequest,
  WorkElement,
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
  WorkDetailsRequest,
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
export type BrowseType = BrowseWork["type"];
export type NsfwPreference = NsfwPreferenceRequest["preference"];

export type BrowseFilters = Pick<
  ListWorksParams,
  "type" | "platform" | "q" | "facet"
>;

export type WorkListParams = BrowseFilters &
  Pick<ListWorksParams, "creator" | "limit" | "before" | "beforeId" | "nsfw">;

/** Creates an isolated cache for each server render. */
export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { staleTime: 30_000, gcTime: 10 * 60_000 },
    },
  });
}

export const workKeys = {
  all: ["works"] as const,
  list: (
    filters: BrowseFilters,
    preference?: NsfwPreference,
    creator?: string,
  ) => ["works", "list", creator, filters, preference] as const,
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

export async function fetchWorks(
  params: WorkListParams,
  cookie?: string,
  signal?: AbortSignal,
): Promise<BrowsePage> {
  const { data, error } = await api<WorkList>("GET", "/v1/works", {
    query: params,
    headers: cookie ? { cookie } : undefined,
    signal,
  });
  if (error || !data) throw new Error("Could not load works");
  return data;
}

export async function fetchDeletedWorks(
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

export async function fetchWork(
  id: string,
  cookie?: string,
  workingCopy = false,
): Promise<WorkDetail | null> {
  const { data, error } = await api<WorkDetail>("GET", `/v1/works/${id}`, {
    query: { workingCopy },
    headers: cookie ? { cookie } : undefined,
  });
  if (error || !data) return null;
  return data;
}

export type StartWorkApp = NonNullable<StartWorkRequest["app"]>;

export async function startWork(
  type: string,
  app?: StartWorkApp,
): Promise<WorkDetail> {
  const { data, error } = await api<WorkDetail>("POST", "/v1/works", {
    body: app ? { type, app } : { type },
  });
  if (error || !data || !("blocks" in data)) {
    throw new Error("Could not start the work");
  }
  return data;
}

export async function saveNsfwPreference(preference: NsfwPreference) {
  const { error } = await api<void>("PUT", "/v1/account/nsfw-preference", {
    body: { preference },
  });
  if (error) throw new Error("Could not save the content preference");
}

export async function saveWorkVisibility(
  id: string,
  visibility: WorkDetail["visibility"],
) {
  const { error } = await api<void>("PUT", `/v1/works/${id}/visibility`, {
    body: { visibility },
  });
  if (error) throw new Error("Could not save the visibility");
}

export async function saveWorkBlock(
  candidate: Candidate,
  workId: string,
  blockId: string,
  block: SaveWorkBlockRequest,
): Promise<WorkBlock> {
  const { data, error, response } = await api<WorkBlock>(
    "PUT",
    `/v1/works/${workId}/blocks/${blockId}`,
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

export async function addWorkBlock(
  candidate: Candidate,
  workId: string,
  definition: string,
  elementType: ElementType,
): Promise<WorkBlock> {
  const { data, error, response } = await api<WorkBlock>(
    "POST",
    `/v1/works/${workId}/blocks`,
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

export async function addWorkImage(
  candidate: Candidate,
  workId: string,
  file: File,
  role: AddMediaRequest["role"],
): Promise<string> {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ role }));
  body.append("file", file, file.name);
  const { data: added, response } = await api<{ id?: unknown }>(
    "POST",
    `/v1/works/${workId}/media`,
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

export async function arrangeWorkBlocks(
  candidate: Candidate,
  workId: string,
  arrangement: ArrangeWorkBlocksRequest,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "PUT",
    `/v1/works/${workId}/blocks`,
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
export async function fetchVault(workId: string): Promise<VaultPicture[]> {
  const { data, error } = await api<VaultPictureList>(
    "GET",
    `/v1/works/${workId}/vault`,
  );
  if (error || !data) {
    throw new Error("The waiting pictures could not be read. Try again.");
  }
  return data.pictures;
}

export async function placeVaultPicture(
  candidate: Candidate,
  workId: string,
  pictureId: string,
  mediaId?: string,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "POST",
    `/v1/works/${workId}/vault/${pictureId}/place`,
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
  workId: string,
  pictureId: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/works/${workId}/vault/${pictureId}`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The picture could not be discarded. Try again.");
  }
}

export async function removeWorkBlock(
  candidate: Candidate,
  workId: string,
  blockId: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/works/${workId}/blocks/${blockId}`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The block could not be removed. Try again.");
  }
}

export async function moveWorkBlockContent(
  candidate: Candidate,
  workId: string,
  blockId: string,
  destinationBlockId: string,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "POST",
    `/v1/works/${workId}/blocks/${blockId}/move-and-remove`,
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

export async function saveWorkDetails(
  candidate: Candidate,
  id: string,
  details: WorkDetailsRequest,
) {
  const { error, response } = await api<void>(
    "PUT",
    `/v1/works/${id}/details`,
    {
      headers: { "X-Working-Copy-Version": String(candidate.version) },
      body: details,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error) {
    throw writeRefusal(error, "The details could not be saved. Try again.");
  }
}

export async function publishWork(
  candidate: Candidate,
  id: string,
): Promise<
  | { published: true }
  | { published: false; error: string; readiness?: ReadinessItem[] }
> {
  const { data, error, response } = await api<WorkDetail>(
    "POST",
    `/v1/works/${id}/publish`,
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
        : "The work could not be published. Try again.",
    readiness: refusal?.readiness,
  };
}

export async function fetchWaitingReplacement(
  id: string,
): Promise<IngestOperation | null> {
  const { data, error } = await api<IngestOperation | null>(
    "GET",
    `/v1/works/${id}/revisions`,
  );
  if (error || !data) return null;
  return data;
}

export async function uploadWorkReplacement(
  candidate: Candidate,
  id: string,
  file: File,
): Promise<IngestOperation> {
  const body = new FormData();
  body.append("file", file, file.name);
  const { data, error, response } = await api<IngestOperation>(
    "POST",
    `/v1/works/${id}/revisions`,
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

export async function acceptWorkReplacement(
  candidate: Candidate,
  id: string,
  operationId: string,
  unrepresentable: ReplacementDecision,
  exposeProtected = false,
): Promise<IngestOperation> {
  const { data, error, response } = await api<IngestOperation>(
    "POST",
    `/v1/works/${id}/revisions/${operationId}/accept`,
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

export async function cancelWorkReplacement(id: string, operationId: string) {
  const { error } = await api<void>(
    "DELETE",
    `/v1/works/${id}/revisions/${operationId}`,
  );
  if (error) throw new Error("That file could not be discarded. Try again.");
}

export async function publishWorkUpdate(
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
    `/v1/works/${id}/updates`,
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
    `/v1/works/${id}/preserved`,
  );
  return data ?? [];
}

export async function fetchWorkUpdates(
  id: string,
): Promise<RecordedVersion[] | null> {
  try {
    const { data } = await api<RecordedVersionList>(
      "GET",
      `/v1/works/${id}/updates`,
    );
    return data?.items ?? null;
  } catch {
    return null;
  }
}

export async function restoreWorkVersion(
  candidate: Candidate,
  id: string,
  number: number,
) {
  const { error, response } = await api<void>(
    "POST",
    `/v1/works/${id}/updates/${number}/restore`,
    { headers: { "X-Working-Copy-Version": String(candidate.version) } },
  );
  acceptCandidateVersion(candidate, response);
  if (error) throw writeRefusal(error, "That version could not be restored.");
}

export async function correctWorkVersionNotes(
  id: string,
  number: number,
  correction: WorkVersionNotesRequest,
) {
  const { error } = await api<void>(
    "PATCH",
    `/v1/works/${id}/updates/${number}/notes`,
    { body: correction },
  );
  if (error) throw writeRefusal(error, "Those notes could not be corrected.");
}

export async function withdrawWorkVersion(
  id: string,
  number: number,
  explanation: string,
) {
  const { error } = await api<void>(
    "POST",
    `/v1/works/${id}/updates/${number}/withdraw`,
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
      `/v1/works/${id}/updates/${number}/downloads`,
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

export async function compareWorkVersions(
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
      `/v1/works/${id}/updates/comparison`,
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
    `/v1/works/${id}/updates/protection`,
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
    `/v1/works/${id}/updates/${number}/protection`,
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
    `/v1/works/${id}/preserved/${encodeURIComponent(namespace)}`,
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

export async function withholdWork(id: string, reason: string) {
  const { error } = await api<void>("PUT", `/v1/works/${id}/withhold`, {
    body: { reason },
  });
  if (error) throw new Error("Could not withhold the work");
}

export async function deleteWork(id: string) {
  const { error } = await api<void>("DELETE", `/v1/works/${id}`);
  if (error) throw new Error("Could not delete the work");
}

export async function restoreWork(id: string) {
  const { error } = await api<void>("POST", `/v1/works/${id}/restore`);
  if (error) throw new Error("Could not restore the work");
}
