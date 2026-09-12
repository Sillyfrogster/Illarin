import type { components } from "@/lib/api/schema";
import type { PostDocument } from "@/lib/post-document";

type Grant = components["schemas"]["PublicationGrant"];

type CreatePostRequest = components["schemas"]["CreatePostRequest"];

type PublishPostRequest = components["schemas"]["PublishPostRequest"];

type SavePostRequest = components["schemas"]["SavePostRequest"];

export type GrantExample = {
  title: string;
  note: string;
  language: string;
  source: string;
};

const RELEASE_CATEGORY = "release";

const POST_ID = "{postId}";

const TOKEN = "Bearer <token>";

/** Builds a create, save and publish example using the contributor's approval. */
export function grantExamples(grant: Grant): GrantExample[] {
  return [
    credentialExample(),
    createExample(grant),
    saveExample(grant),
    publishExample(grant),
  ];
}

function credentialExample(): GrantExample {
  return {
    title: "Check the token",
    note: "Checks whether the token works and returns its permissions. It never returns the token secret.",
    language: "http",
    source: `GET /api/v1/publication/token\nAuthorization: ${TOKEN}`,
  };
}

function createExample(grant: Grant): GrantExample {
  const body: CreatePostRequest = {
    categoryId: grant.defaultCategory.id,
    title: `${grant.app.name} update`,
  };
  return {
    title: "Create a post",
    note: `Creates a private draft in ${grant.defaultCategory.label}. Copy the response's id into {postId} in the following requests.`,
    language: "http",
    source: request("POST", "/api/v1/publication/posts", body),
  };
}

function saveExample(grant: Grant): GrantExample {
  const body: Omit<SavePostRequest, "document"> & { document: PostDocument } = {
    version: 1,
    categoryId: grant.defaultCategory.id,
    title: `${grant.app.name} update`,
    summary: "A description of what changed and who it helps.",
    slug: `${grant.app.slug}-update`,
    document: {
      version: 2,
      content: [
        {
          type: "paragraph",
          content: [{ type: "text", text: "Write your update here." }],
        },
      ],
    },
    release:
      grant.defaultCategory.slug === RELEASE_CATEGORY
        ? { appId: grant.app.id, version: "1.0.0" }
        : null,
    header: null,
    socialMediaId: null,
  };
  return {
    title: "Save the post",
    note: "Replace the sample writing and choose an unused slug. Use the version returned when you created the draft. Saving keeps it private; retain the new version from this response for publishing.",
    language: "http",
    source: request("PUT", `/api/v1/publication/posts/${POST_ID}`, body),
  };
}

function publishExample(grant: Grant): GrantExample {
  const chosen = grant.destinations
    .filter((one) => one.byDefault)
    .map((one) => one.id);
  const body: PublishPostRequest = {
    version: 2,
    destinationIds: chosen,
    note: `${grant.app.name} update is out.`,
  };
  const named = grant.destinations
    .filter((one) => one.byDefault)
    .map((one) => one.name);
  return {
    title: "Publish now",
    note:
      named.length > 0
        ? `Makes the saved post public and announces it to ${named.join(", ")}. Use the version from your last save. Set destinationIds to [] to publish without an announcement.`
        : "Makes the saved post public without an announcement. Use the version from your last save. No destinations are selected by default.",
    language: "http",
    source: request(
      "POST",
      `/api/v1/publication/posts/${POST_ID}/publish`,
      body,
    ),
  };
}

function request(method: string, path: string, body: unknown): string {
  return [
    `${method} ${path}`,
    `Authorization: ${TOKEN}`,
    "Content-Type: application/json",
    "Idempotency-Key: <one UUID per action; reuse for retries>",
    "",
    JSON.stringify(body, null, 2),
  ].join("\n");
}
