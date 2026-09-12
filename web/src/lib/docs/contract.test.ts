import { expect, test } from "bun:test";
import { type Contract, checkDoc, readHttpExample } from "./contract";
import { readDoc } from "./read-doc";

const CONTRACT: Contract = {
  paths: {
    "/v1/publication/posts": {
      post: {
        requestBody: {
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/CreatePostRequest" },
            },
          },
        },
        responses: {
          "201": {
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/Post" },
              },
            },
          },
        },
      },
    },
    "/v1/publication/posts/{id}/publish": {
      post: {
        requestBody: {
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/PublishPostRequest" },
            },
          },
        },
        responses: {},
      },
    },
  },
  components: {
    schemas: {
      CreatePostRequest: {
        type: "object",
        additionalProperties: false,
        required: ["categoryId", "title"],
        properties: {
          grantId: { type: "string", format: "uuid" },
          categoryId: { type: "string", format: "uuid" },
          title: { type: "string" },
        },
      },
      PublishPostRequest: {
        type: "object",
        additionalProperties: false,
        required: ["version"],
        properties: {
          version: { type: "integer" },
          destinationIds: {
            type: "array",
            nullable: true,
            items: { type: "string", format: "uuid" },
          },
          note: { type: "string" },
        },
      },
      Post: {
        type: "object",
        required: ["id", "status", "release"],
        properties: {
          id: { type: "string", format: "uuid" },
          status: { type: "string", enum: ["draft", "published"] },
          release: {
            anyOf: [
              { $ref: "#/components/schemas/PostRelease" },
              { type: "null" },
            ],
          },
        },
      },
      PostRelease: {
        type: "object",
        required: ["version"],
        properties: { version: { type: "string" } },
      },
    },
  },
};

const ID = "0f3c6f6e-2b1a-4c7d-9e8f-1a2b3c4d5e6f";

test("an http example splits into its request line, headers and body", () => {
  expect(
    readHttpExample(
      `POST /api/v1/publication/posts/${ID}/publish\nAuthorization: Bearer ip1.example\nIdempotency-Key: publish-1\n\n{"version": 4}`,
    ),
  ).toEqual({
    method: "post",
    path: `/v1/publication/posts/${ID}/publish`,
    headers: {
      authorization: "Bearer ip1.example",
      "idempotency-key": "publish-1",
    },
    body: { version: 4 },
  });
  expect(readHttpExample("GET /api/v1/publication/posts")).toEqual({
    method: "get",
    path: "/v1/publication/posts",
    headers: {},
    body: undefined,
  });
});

test("a doc whose examples match the contract has nothing to report", () => {
  const doc = readDoc(`# T

Lede.

\`\`\`http
POST /api/v1/publication/posts
Content-Type: application/json

{"categoryId": "${ID}", "title": "Hello"}
\`\`\`

\`\`\`json Post
{"id": "${ID}", "status": "draft", "release": null}
\`\`\`

\`\`\`json Post
{"id": "${ID}", "status": "published", "release": {"version": "1.2"}}
\`\`\`
`);
  expect(checkDoc(doc, CONTRACT)).toEqual([]);
});

test("a request off the contract is reported by what is wrong with it", () => {
  const doc = readDoc(`# T

Lede.

\`\`\`http
PUT /api/v1/publication/posts
\`\`\`

\`\`\`http
POST /api/v1/publication/posts/${ID}/publish

{"version": "4", "destinationIds": ["not-a-uuid"], "extra": true}
\`\`\`

\`\`\`http
POST /api/v1/publication/posts

{"title": "No category"}
\`\`\`

\`\`\`http
GET /api/v1/publication/nowhere
\`\`\`
`);
  expect(checkDoc(doc, CONTRACT)).toEqual([
    "PUT /v1/publication/posts: the contract has no such operation.",
    `POST /v1/publication/posts/${ID}/publish: body.version is not an integer.`,
    `POST /v1/publication/posts/${ID}/publish: body.destinationIds.0 is not a uuid.`,
    `POST /v1/publication/posts/${ID}/publish: body.extra is not in the contract.`,
    "POST /v1/publication/posts: body.categoryId is required.",
    "GET /v1/publication/nowhere: the contract has no such path.",
  ]);
});

test("an endpoint line is checked like a request without a body", () => {
  const doc = readDoc(`# T

Lede.

\`POST /api/v1/publication/posts\`

\`PATCH /api/v1/publication/posts/{id}/publish\`
`);
  expect(checkDoc(doc, CONTRACT)).toEqual([
    "PATCH /v1/publication/posts/{id}/publish: the contract has no such operation.",
  ]);
});

test("a json example names a schema the contract has and fits it", () => {
  const doc = readDoc(`# T

Lede.

\`\`\`json Nothing
{}
\`\`\`

\`\`\`json Post
{"id": "${ID}", "status": "gone", "release": {"version": 2}}
\`\`\`

\`\`\`json
{"untitled": true}
\`\`\`
`);
  expect(checkDoc(doc, CONTRACT)).toEqual([
    "json Nothing: the contract has no such schema.",
    "json Post: status is not one of draft, published.",
    "json Post: release fits none of its shapes.",
    "json: a json example names the schema it fits.",
  ]);
});

test("a request Illarin sends is checked against the webhook it carries", () => {
  const outbound: Contract = {
    ...CONTRACT,
    webhooks: {
      "publication.post.published.v1": {
        post: {
          requestBody: {
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["id", "type"],
                  properties: {
                    id: { type: "string", format: "uuid" },
                    type: { type: "string" },
                  },
                },
              },
            },
          },
        },
      },
    },
  };
  const doc = readDoc(`# T

Lede.

\`\`\`http
POST /hooks/illarin
webhook-id: ${ID}

{"id": "${ID}", "type": "publication.post.published.v1"}
\`\`\`

\`\`\`http
POST /hooks/illarin

{"type": "publication.post.deleted.v1"}
\`\`\`
`);
  expect(checkDoc(doc, outbound)).toEqual([
    "POST /hooks/illarin: the contract sends no such webhook.",
  ]);
});
