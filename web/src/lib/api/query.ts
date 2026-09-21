import { QueryClient } from "@tanstack/react-query";
import { readRefusal } from "@/lib/answer";
import {
  acceptCandidateVersion,
  type Candidate,
  reportStaleDraftedChanges,
} from "@/lib/drafted-changes";
import { api } from "./client";
import type {
  AddableBlock,
  AddedBlogIntegration,
  AddMediaRequest,
  AppFormat,
  AppList,
  AppName,
  ArrangeWorkBlocksRequest,
  BlogAnnouncementAttempt,
  BlogAnnouncementAttemptState,
  BlogAnnouncementTry,
  BlogAnnouncementType,
  BlogCategory,
  BlogCategoryList,
  BlogChannel,
  BlogIntegration,
  BlogIntegrationChoice,
  BlogIntegrationChoiceList,
  BlogIntegrationType,
  BlogWorkspace,
  BrowseCursor,
  BrowseWork,
  BuildChoices,
  ColorSetContent,
  DeletedWork,
  DeletedWorkList,
  DownloadFormat,
  ElementType,
  EntryTableContent,
  ExtensionDependency,
  FormatComparison,
  FoundImage,
  FoundImageList,
  ListWorksParams,
  NsfwPreferenceRequest,
  OriginalUpload,
  Post,
  PostAction,
  PostArchive,
  PostByline,
  PostDeletion,
  PostMedia,
  PostMediaPurpose,
  PostRevision,
  PostSchedule,
  PostSummary,
  PostUnpublishing,
  PreservedData,
  PrivatePromptMismatch,
  PrivatePromptMismatchList,
  Profile,
  ProfileLink,
  PromptCorrespondenceRequest,
  PromptListContent,
  PublicPost,
  QueuedSend,
  ReaderPreferences,
  ReadinessItem,
  RecentVersion,
  RecordedVersion,
  RecordedVersionDownloads,
  RecordedVersionList,
  RecordListContent,
  ReplacementAcceptance,
  ReplacementPreview,
  RestrictedProfile,
  RotatedBlogSecret,
  SaveWorkBlockRequest,
  ScriptListContent,
  SettingGroupContent,
  StylesheetSetContent,
  TypedValue,
  UploadOperation,
  VariableSchemaContent,
  VersionChange,
  VersionChangeGroup,
  VersionComparison,
  WorkBlock,
  WorkConnectedApp,
  WorkConnectedAppList,
  WorkDetail,
  WorkDetailsRequest,
  WorkElement,
  WorkImage,
  WorkList,
  WorkTag,
  WorkVersion,
  WorkVersionNotesRequest,
  WorkVersionRequest,
  WriterList,
  WriterResponse,
} from "./shapes";
export type {
  AddableBlock,
  AppName,
  ReaderPreferences,
  AddedBlogIntegration,
  AppFormat,
  ArrangeWorkBlocksRequest,
  WorkBlock,
  WorkDetail,
  WorkElement,
  WorkDetailsRequest,
  WorkImage,
  WorkConnectedApp,
  WorkConnectedAppList,
  WorkTag,
  WorkVersion,
  WorkVersionRequest,
  WorkVersionNotesRequest,
  BrowseWork,
  BrowseCursor,
  ColorSetContent,
  DeletedWork,
  DownloadFormat,
  ElementType,
  EntryTableContent,
  ExtensionDependency,
  FormatComparison,
  UploadOperation,
  OriginalUpload,
  Post,
  PostAction,
  PostArchive,
  PostByline,
  PostDeletion,
  BlogAnnouncementAttempt,
  BlogAnnouncementTry,
  BlogAnnouncementAttemptState,
  PostMedia,
  PostMediaPurpose,
  PostRevision,
  PostSchedule,
  PostSummary,
  PostUnpublishing,
  PreservedData,
  Profile,
  ProfileLink,
  RecentVersion,
  RestrictedProfile,
  PromptCorrespondenceRequest,
  PromptListContent,
  PrivatePromptMismatch,
  PublicPost,
  BlogCategory,
  BlogChannel,
  BlogIntegration,
  BlogIntegrationChoice,
  BlogIntegrationChoiceList,
  BlogIntegrationType,
  BlogAnnouncementType,
  BlogWorkspace,
  WriterList,
  WriterResponse,
  QueuedSend,
  ReadinessItem,
  RecordListContent,
  RecordedVersion,
  RecordedVersionDownloads,
  ReplacementPreview,
  RotatedBlogSecret,
  SaveWorkBlockRequest,
  ScriptListContent,
  SettingGroupContent,
  StylesheetSetContent,
  TypedValue,
  VariableSchemaContent,
  FoundImage,
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

export type BrowseFilters = Pick<ListWorksParams, "type" | "q" | "facet">;

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

export class PromptsMadePublicError extends Error {
  constructor(
    message: string,
    readonly prompts: string[],
  ) {
    super(message);
  }
}

function writeRefusal(error: unknown, fallback: string): Error {
  reportStaleDraftedChanges(error);
  const detail = error as
    | { error?: unknown; code?: unknown; prompts?: unknown }
    | undefined;
  if (
    detail?.code === "prompts_made_public" &&
    Array.isArray(detail.prompts) &&
    detail.prompts.every((prompt) => typeof prompt === "string")
  ) {
    return new PromptsMadePublicError(
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

export async function fetchProfile(
  handle: string,
  cookie?: string,
): Promise<Profile | null> {
  const { data, error } = await api<Profile>(
    "GET",
    `/v1/profiles/${encodeURIComponent(handle)}`,
    { headers: cookie ? { cookie } : undefined },
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
  draftedChanges = false,
): Promise<WorkDetail | null> {
  const { data, error } = await api<WorkDetail>("GET", `/v1/works/${id}`, {
    query: { draftedChanges },
    headers: cookie ? { cookie } : undefined,
  });
  if (error || !data) return null;
  return data;
}

/** fetchBuildChoices asks which types can be built from nothing and which apps each asks for. */
export async function fetchBuildChoices(): Promise<BuildChoices> {
  const { data, error } = await api<BuildChoices>("GET", "/v1/build-choices");
  if (error || !data) throw new Error("Could not load the types to build");
  return data;
}

export async function startWork(
  type: string,
  app?: string,
): Promise<WorkDetail> {
  const { data, error } = await api<WorkDetail>("POST", "/v1/works", {
    body: app ? { type, app } : { type },
  });
  if (error || !data || !("blocks" in data)) {
    throw new Error("Could not start the work");
  }
  return data;
}

/** fetchApps lists the apps a reader can pick as theirs. */
export async function fetchApps(): Promise<AppName[]> {
  const { data, error } = await api<AppList>("GET", "/v1/apps");
  if (error || !data) throw new Error("Could not load the apps");
  return data.apps;
}

export async function fetchPreferences(): Promise<ReaderPreferences> {
  const { data, error } = await api<ReaderPreferences>(
    "GET",
    "/v1/account/preferences",
  );
  if (error || !data) throw new Error("Could not load your preferences");
  return data;
}

/** saveAppPreference keeps the reader's app on their account, or in this browser when they are signed out. */
export async function saveAppPreference(app: string, signedIn: boolean) {
  if (!signedIn) {
    // biome-ignore lint/suspicious/noDocumentCookie: Safari has no Cookie Store API, and the server reads this cookie on the next request.
    document.cookie = `illarin_app=${encodeURIComponent(app)}; path=/; max-age=31536000; samesite=lax`;
    return;
  }
  const { response } = await api<void>("PUT", "/v1/account/app-preference", {
    body: { app },
  });
  if (!response.ok) throw new Error("Could not save your app");
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
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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
export async function fetchFoundImages(workId: string): Promise<FoundImage[]> {
  const { data, error } = await api<FoundImageList>(
    "GET",
    `/v1/works/${workId}/found-images`,
  );
  if (error || !data) {
    throw new Error("The waiting pictures could not be read. Try again.");
  }
  return data.pictures;
}

export async function placeFoundImage(
  candidate: Candidate,
  workId: string,
  pictureId: string,
  mediaId?: string,
): Promise<WorkBlock[]> {
  const { data, error, response } = await api<WorkBlock[]>(
    "POST",
    `/v1/works/${workId}/found-images/${pictureId}/place`,
    {
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
      body: mediaId ? { mediaId } : undefined,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (error || !data) {
    throw writeRefusal(error, "The picture could not be placed. Try again.");
  }
  return data;
}

export async function discardFoundImage(
  candidate: Candidate,
  workId: string,
  pictureId: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/works/${workId}/found-images/${pictureId}`,
    { headers: { "X-Drafted-Changes-Version": String(candidate.version) } },
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
    { headers: { "X-Drafted-Changes-Version": String(candidate.version) } },
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
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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
    { headers: { "X-Drafted-Changes-Version": String(candidate.version) } },
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
): Promise<UploadOperation | null> {
  const { data, error } = await api<UploadOperation | null>(
    "GET",
    `/v1/works/${id}/original-file`,
  );
  if (error || !data) return null;
  return data;
}

export async function uploadWorkReplacement(
  candidate: Candidate,
  id: string,
  file: File,
): Promise<UploadOperation> {
  const body = new FormData();
  body.append("file", file, file.name);
  const { data, error, response } = await api<UploadOperation>(
    "POST",
    `/v1/works/${id}/original-file`,
    {
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
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

export async function readUploadOperation(
  url: string,
): Promise<UploadOperation> {
  const { data } = await api<UploadOperation>("GET", url, {
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
  makePromptsPublic = false,
): Promise<UploadOperation> {
  const { data, error, response } = await api<UploadOperation>(
    "POST",
    `/v1/works/${id}/original-file/${operationId}/accept`,
    {
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
      body: { unrepresentable, makePromptsPublic },
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
    `/v1/works/${id}/original-file/${operationId}`,
  );
  if (error) throw new Error("That file could not be discarded. Try again.");
}

export async function publishWorkVersion(
  candidate: Candidate,
  id: string,
  version: WorkVersionRequest,
): Promise<
  | { published: true; version: WorkVersion }
  | {
      published: false;
      error: string;
      code?: string;
      field?: string;
      readiness?: ReadinessItem[];
    }
> {
  const { data, error, response } = await api<WorkVersion>(
    "POST",
    `/v1/works/${id}/versions`,
    {
      headers: { "X-Drafted-Changes-Version": String(candidate.version) },
      body: version,
    },
  );
  acceptCandidateVersion(candidate, response);
  if (data) return { published: true, version: data };
  reportStaleDraftedChanges(error);
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
        : "The version could not be published. Try again.",
    code: typeof refusal?.code === "string" ? refusal.code : undefined,
    field: typeof refusal?.field === "string" ? refusal.field : undefined,
    readiness: refusal?.readiness,
  };
}

export async function fetchPreservedData(id: string): Promise<PreservedData[]> {
  const { data } = await api<PreservedData[]>(
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
      `/v1/works/${id}/versions`,
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
    `/v1/works/${id}/versions/${number}/restore`,
    { headers: { "X-Drafted-Changes-Version": String(candidate.version) } },
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
    `/v1/works/${id}/versions/${number}/notes`,
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
    `/v1/works/${id}/versions/${number}/unpublish`,
    { body: { explanation } },
  );
  if (error) throw writeRefusal(error, "That version could not be withdrawn.");
}

/** fetchFormatTable reads the comparison of every format Illarin writes, field by field. */
export async function fetchFormatTable(): Promise<FormatComparison> {
  const { data, error } = await api<FormatComparison>("GET", "/v1/formats");
  if (error || !data) throw new Error("Could not load the format comparison");
  return data;
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
      `/v1/works/${id}/versions/${number}/downloads`,
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
      `/v1/works/${id}/versions/comparison`,
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

export async function fetchPrivatePromptMismatches(
  id: string,
): Promise<PrivatePromptMismatch[]> {
  const { data } = await api<PrivatePromptMismatchList>(
    "GET",
    `/v1/works/${id}/versions/private-prompts`,
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
    `/v1/works/${id}/versions/${number}/private-prompts`,
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

export async function deletePreservedData(
  candidate: Candidate,
  id: string,
  namespace: string,
) {
  const { error, response } = await api<void>(
    "DELETE",
    `/v1/works/${id}/preserved/${encodeURIComponent(namespace)}`,
    { headers: { "X-Drafted-Changes-Version": String(candidate.version) } },
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

export async function fetchPostCategories(): Promise<BlogCategory[]> {
  const answer = await api<BlogCategoryList>(
    "GET",
    "/v1/post-categories",
  ).catch(() => null);
  return answer?.data?.categories ?? [];
}

export async function fetchRestrictedProfile(
  handle: string,
): Promise<RestrictedProfile | null> {
  const { data, error } = await api<RestrictedProfile>(
    "GET",
    `/v1/profiles/${encodeURIComponent(handle)}/restricted`,
  );
  if (error || !data) return null;
  return data;
}

export async function restrictProfile(handle: string, reason: string) {
  const { data, error } = await api<RestrictedProfile>(
    "PUT",
    `/v1/profiles/${encodeURIComponent(handle)}/restricted`,
    { body: { reason } },
  );
  if (error || !data) throw new Error("Could not restrict the profile");
  return data;
}

export async function restoreProfile(handle: string) {
  const { error } = await api<void>(
    "DELETE",
    `/v1/profiles/${encodeURIComponent(handle)}/restricted`,
  );
  if (error) throw new Error("Could not restore the profile");
}

export async function takeDownWork(id: string, reason: string) {
  const { error } = await api<void>("PUT", `/v1/works/${id}/takedown`, {
    body: { reason },
  });
  if (error) throw new Error("Could not take down the work");
}

export async function deleteWork(id: string) {
  const { error } = await api<void>("DELETE", `/v1/works/${id}`);
  if (error) throw new Error("Could not delete the work");
}

export async function restoreWork(id: string) {
  const { error } = await api<void>("POST", `/v1/works/${id}/restore`);
  if (error) throw new Error("Could not restore the work");
}
