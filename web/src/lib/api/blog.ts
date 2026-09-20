import type {
  AddedBlogIntegration,
  BlogAnnouncementAttempt,
  BlogAnnouncementAttemptState,
  BlogAnnouncementTry,
  BlogAnnouncementType,
  BlogCategory,
  BlogIntegration,
  BlogWorkspace,
  RotatedBlogSecret,
  WriterList,
  WriterResponse,
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
  return ask<{ categories: BlogCategory[] }>("GET", "/blog/categories");
}

export function updateCategory(
  id: string,
  change: { label?: string; retired?: boolean },
) {
  return ask<BlogCategory>("PATCH", `/blog/categories/${id}`, {
    body: change,
  });
}

export function orderCategories(categoryIds: string[]) {
  return ask<{ categories: BlogCategory[] }>("PUT", "/blog/categories", {
    body: { categoryIds },
  });
}

export function readWriters() {
  return ask<WriterList>("GET", "/blog/writers");
}

export function switchWriterOn(handle: string) {
  return ask<WriterResponse>("POST", "/blog/writers", { body: { handle } });
}

export function switchWriterOff(accountId: string) {
  return ask<void>("DELETE", `/blog/writers/${accountId}`);
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

export function readWorkspace() {
  return ask<BlogWorkspace>("GET", "/blog/workspace");
}
