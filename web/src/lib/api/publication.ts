import type {
  AddedBlogIntegration,
  BlogAnnouncementAttempt,
  BlogAnnouncementAttemptState,
  BlogAnnouncementTry,
  BlogAnnouncementType,
  BlogIntegration,
  PublicationCategory,
  PublicationGrant,
  PublicationWorkspace,
  RotatedBlogSecret,
} from "@/lib/api/query";
import { ask } from "./request";

export type DiscordRepair = {
  requestId: string;
  action: "edit" | "delete" | "correction";
  messageId: string;
  text: string;
};

export type DiscordRepairResult = {
  action: string;
  messageId: string;
  state: "completed" | "refused" | "unconfirmed";
  detail: string;
};

export function repairDiscordAnnouncement(id: string, repair: DiscordRepair) {
  return ask<DiscordRepairResult>(
    "POST",
    `/blog/announcement-attempts/${id}/repair`,
    { body: repair },
  );
}

export function readCategories() {
  return ask<{ categories: PublicationCategory[] }>(
    "GET",
    "/publication/categories",
  );
}

export function updateCategory(
  id: string,
  change: { label?: string; retired?: boolean },
) {
  return ask<PublicationCategory>("PATCH", `/publication/categories/${id}`, {
    body: change,
  });
}

export function orderCategories(categoryIds: string[]) {
  return ask<{ categories: PublicationCategory[] }>(
    "PUT",
    "/publication/categories",
    { body: { categoryIds } },
  );
}

export function readGrants() {
  return ask<{ grants: PublicationGrant[] }>("GET", "/publication/grants");
}

export function approveContributor(approval: {
  handle: string;
  appId: string;
  categoryIds: string[];
  defaultCategoryId: string;
}) {
  return ask<PublicationGrant>("POST", "/publication/grants", {
    body: approval,
  });
}

export function updateGrant(
  id: string,
  change: { categoryIds?: string[]; defaultCategoryId?: string },
) {
  return ask<PublicationGrant>("PATCH", `/publication/grants/${id}`, {
    body: change,
  });
}

export function revokeGrant(id: string) {
  return ask<void>("DELETE", `/publication/grants/${id}`);
}

export function readIntegrations() {
  return ask<{ integrations: BlogIntegration[] }>("GET", "/blog/integrations");
}

export function addIntegration(endpoint: {
  name: string;
  address: string;
  announcements: BlogAnnouncementType[];
}) {
  return ask<AddedBlogIntegration>("POST", "/blog/integrations", {
    body: endpoint,
  });
}

export function updateIntegration(
  id: string,
  change: {
    name?: string;
    address?: string;
    announcements?: BlogAnnouncementType[];
  },
) {
  return ask<BlogIntegration>("PATCH", `/blog/integrations/${id}`, {
    body: change,
  });
}

export function addChannel(channel: {
  name: string;
  address: string;
  roleId: string;
  roleName: string;
}) {
  return ask<BlogIntegration>("POST", "/blog/channels", {
    body: channel,
  });
}

export function updateChannel(
  id: string,
  change: { name: string; address?: string; roleId: string; roleName: string },
) {
  return ask<BlogIntegration>("PATCH", `/blog/channels/${id}`, {
    body: change,
  });
}

export function verifyIntegration(id: string) {
  return ask<BlogIntegration>("POST", `/blog/integrations/${id}/verification`);
}

export function disableIntegration(id: string) {
  return ask<BlogIntegration>(
    "DELETE",
    `/blog/integrations/${id}/verification`,
  );
}

export function rotateIntegrationSecret(id: string) {
  return ask<RotatedBlogSecret>("POST", `/blog/integrations/${id}/secret`);
}

export function readDeliveries(state?: BlogAnnouncementAttemptState) {
  const narrowed = state ? `?state=${state}` : "";
  return ask<{ attempts: BlogAnnouncementAttempt[] }>(
    "GET",
    `/blog/announcement-attempts${narrowed}`,
  );
}

export function readAnnouncementTries(id: string) {
  return ask<{ tries: BlogAnnouncementTry[] }>(
    "GET",
    `/blog/announcement-attempts/${id}/tries`,
  );
}

export function replayAnnouncementAttempt(id: string) {
  return ask<BlogAnnouncementAttempt>(
    "POST",
    `/blog/announcement-attempts/${id}/replay`,
  );
}

export function removeIntegration(id: string) {
  return ask<void>("DELETE", `/blog/integrations/${id}`);
}

export function setGrantIntegrations(
  grantId: string,
  policy: {
    integrationIds: string[] | null;
    defaultIntegrationIds: string[];
  },
) {
  return ask<PublicationGrant>("PUT", `/blog/grants/${grantId}/integrations`, {
    body: policy,
  });
}

export function readWorkspace() {
  return ask<PublicationWorkspace>("GET", "/publication/workspace");
}
