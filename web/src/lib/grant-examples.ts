import type { components } from "@/lib/api/schema";

type Grant = components["schemas"]["PublicationGrant"];

type CreatePostRequest = components["schemas"]["CreatePostRequest"];

type PublishPostRequest = components["schemas"]["PublishPostRequest"];

type PostReleaseEdit = components["schemas"]["PostReleaseEdit"];

export type GrantExample = {
  title: string;
  note: string;
  language: string;
  source: string;
};

const RELEASE_CATEGORY = "release";

const POST_ID = "{postId}";

const TOKEN = "Bearer <token>";

/** Requests filled with the ids this grant may use, and nothing anyone else's. */
export function grantExamples(grant: Grant): GrantExample[] {
  const examples = [credentialExample(), createExample(grant)];
  const release = releaseExample(grant);
  if (release) examples.push(release);
  examples.push(publishExample(grant));
  return examples;
}

function credentialExample(): GrantExample {
  return {
    title: "Check the token",
    note: "Returns the token and this approval, including every id in the table.",
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
    note: `categoryId is your default, ${grant.defaultCategory.label}. Any category id from the table below works.`,
    language: "http",
    source: request("POST", "/api/v1/publication/posts", body),
  };
}

function releaseExample(grant: Grant): GrantExample | null {
  if (!grant.categories.some((one) => one.slug === RELEASE_CATEGORY)) {
    return null;
  }
  const release: PostReleaseEdit = {
    appId: grant.app.id,
    version: "1.0.0",
    address: `${grant.app.home.replace(/\/$/, "")}/releases/1.0.0`,
  };
  return {
    title: "Release fields",
    note: `The release object a save needs when the category is Release. appId is ${grant.app.name}.`,
    language: "json",
    source: JSON.stringify({ release }, null, 2),
  };
}

function publishExample(grant: Grant): GrantExample {
  const chosen = grant.destinations
    .filter((one) => one.byDefault)
    .map((one) => one.id);
  const body: PublishPostRequest = {
    version: 1,
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
        ? `destinationIds announces to ${named.join(", ")}. Send [] to announce nowhere.`
        : "This approval has no destinations, so destinationIds is empty.",
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
    "Idempotency-Key: <unique per attempt>",
    "",
    JSON.stringify(body, null, 2),
  ].join("\n");
}
