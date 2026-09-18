# Writing

Create and edit private drafts, upload pictures, and keep or restore saved revisions. Nothing on this page is visible to readers.

## Concurrency

Every post has a `version` that increases on each save. Requests that change a post must send the version they read. If the version is stale, the request fails with `409 stale_version` and the current version. Reload and retry. See [Version conflicts](/developers/publication/requests#version-conflicts).

## List posts

`GET /api/v1/publication/posts`

Lists posts under the token's approval, newest first.

| Query | Type | Meaning |
| --- | --- | --- |
| `deleted` | boolean | `true` lists deleted posts still inside their 30-day recovery window instead. |

Returns `200` with `{ "posts": [Post] }`.

## Get a post

`GET /api/v1/publication/posts/{id}`

Returns `200` with the `Post`, working copy included.

| Status | Code | When |
| --- | --- | --- |
| 403 | `forbidden` | The post belongs to another approval. |
| 404 | `not_found` | No such post. |

```json Post
{
  "id": "8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48",
  "status": "draft",
  "title": "Paper Lantern 4.2",
  "summary": "",
  "slug": "paper-lantern-4-2",
  "category": {
    "id": "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
    "slug": "announcement",
    "label": "Announcement",
    "position": 1,
    "retired": false
  },
  "document": { "version": 2, "content": [] },
  "documentVersion": 2,
  "release": null,
  "header": null,
  "socialMediaId": null,
  "publicRevisionId": null,
  "schedule": null,
  "withdrawal": null,
  "deletion": null,
  "media": [],
  "byline": null,
  "formerAddresses": [],
  "app": {
    "id": "5e9a1c3b-7d2f-4e86-b4a0-9c8d7e6f5a4b",
    "slug": "paper-lantern",
    "name": "Paper Lantern",
    "home": "https://paperlantern.example",
    "mark": null,
    "position": 1,
    "retired": false,
    "destinations": []
  },
  "grantId": "2c3d4e5f-6071-4829-b0c1-d2e3f4a5b6c7",
  "version": 1,
  "author": { "handle": "paperlantern" },
  "createdAt": "2026-09-14T09:00:00Z",
  "updatedAt": "2026-09-14T09:00:00Z"
}
```

| Field | Meaning |
| --- | --- |
| `status` | `draft` until first publication, `published` while a revision is public, `withdrawn` after withdrawal. |
| `version` | Working-copy version. Send it with edits and public transitions that require it. |
| `publicRevisionId` | The revision readers see. `null` until published. |
| `formerAddresses` | Slugs the post published under before. All still redirect to it. |
| `schedule`, `withdrawal`, `deletion` | Present while the post is in that state. |
| `media` | Every picture uploaded to the post. |

## Create a post

`POST /api/v1/publication/posts`

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `categoryId` | uuid | yes | A category from your approval. |
| `title` | string | yes | 1–160 characters. |
| `grantId` | uuid | no | Omit. The token supplies it. Another value is refused. |

Returns `201` with the `Post`. Send an [`Idempotency-Key`](/developers/publication/requests#idempotency).

```http
POST /api/v1/publication/posts
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 3f8c1d2e-9a7b-4c5d-8e6f-0a1b2c3d4e5f

{
  "categoryId": "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
  "title": "Paper Lantern 4.2"
}
```

## Save a post

`PUT /api/v1/publication/posts/{id}`

Replaces the whole working copy and keeps the post private until you publish. When editing a published post, readers keep seeing its last published revision. Send the complete document and metadata; omitted optional fields are cleared.

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `version` | integer | yes | The version you read. |
| `categoryId` | uuid | yes | A category from your approval. |
| `title` | string | yes | 1–160 characters. |
| `summary` | string | yes | 0–320 characters, plain text. Must be non-empty to publish. |
| `slug` | string | yes | Address under `https://blog.illarin.com/`. `a-z`, `0-9`, single hyphens, max 80. Locked after first publication. |
| `document` | object | yes | A [post document](/developers/publication/document). May be empty until publish. |
| `release` | object or null | no | Required when the category is `release`, refused otherwise. `{ "appId": uuid, "version": string, "address": string }`. `version` 1–40 characters. `address` optional, `https://`. |
| `header` | object or null | no | `{ "mediaId": uuid, "alt": string, "caption": string }`. `mediaId` must be a picture with purpose `header`. |
| `socialMediaId` | uuid or null | no | A picture with purpose `social`. Replaces the generated sharing card. |

Returns `200` with the `Post`. `version` is incremented.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | A field is wrong. `field` names it, or a path into the document such as `document.content.3.type`. |
| 400 | `category_refused` | The category is outside your approval. |
| 403 | `forbidden` | The slug of a published post was changed. |
| 409 | `stale_version` | `version` is old. |

```http
PUT /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 6d7e8f9a-0b1c-4d2e-8f3a-4b5c6d7e8f9a

{
  "version": 1,
  "categoryId": "c7e1f3a5-9b2d-4c6e-8f1a-3d5b7c9e1f2a",
  "title": "Paper Lantern 4.2",
  "summary": "Scene exports, a faster library and thirty fixes.",
  "slug": "paper-lantern-4-2",
  "document": {
    "version": 2,
    "content": [
      {
        "type": "paragraph",
        "content": [{ "type": "text", "text": "Scenes now export as one file." }]
      }
    ]
  },
  "release": {
    "appId": "5e9a1c3b-7d2f-4e86-b4a0-9c8d7e6f5a4b",
    "version": "4.2.0",
    "address": "https://paperlantern.example/4.2"
  },
  "header": null,
  "socialMediaId": null
}
```

## Upload a picture

`POST /api/v1/publication/posts/{id}/media`

Send `multipart/form-data`, with the `metadata` part first and the `file` part second. Let your HTTP client set the Content-Type boundary. Uploading alone does not place the picture in the article; save its ID in the document, header or social image field afterward.

| Part | Content | Rule |
| --- | --- | --- |
| `metadata` | JSON `{ "purpose": string }` | `document`, `header` or `social`. |
| `file` | image bytes | PNG, JPEG, WebP or GIF. Max 32 MiB. |

| Purpose | Used in |
| --- | --- |
| `document` | `image` and `gallery` blocks in the document. |
| `header` | The working copy's `header`. |
| `social` | The working copy's `socialMediaId`. |

Returns `201` with the `PostMedia`. Pictures are immutable. To replace one, upload another and reference the new id.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | Not a readable image, or the form is malformed. |
| 413 | `invalid` | Over 32 MiB. |
| 503 | `server_error` | Storage is full. Retry later. |

```bash
curl https://illarin.com/api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/media \
  -H 'Authorization: Bearer ip1.…' \
  -H 'Idempotency-Key: 7d2a9c4b-1e5f-4a8b-9c3d-6e0f1a2b3c4d' \
  -F 'metadata={"purpose":"document"};type=application/json' \
  -F 'file=@workspace.png'
```

```json PostMedia
{
  "id": "d4e8f1a2-9b3c-4d5e-8a6f-2b1c3d4e5f60",
  "postId": "8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48",
  "purpose": "document",
  "url": "/media/d4e8f1a2-9b3c-4d5e-8a6f-2b1c3d4e5f60/detail/1?expires=1789192185&signature=…",
  "thumbUrl": "/media/d4e8f1a2-9b3c-4d5e-8a6f-2b1c3d4e5f60/grid/1?expires=1789192185&signature=…",
  "width": 1600,
  "height": 900
}
```

`url` and `thumbUrl` are relative to `https://illarin.com` and signed for 15 minutes. Read the post again for fresh ones. A document may only reference pictures uploaded to the same post.

## Checkpoint

`POST /api/v1/publication/posts/{id}/revisions`

Saves the working copy as a revision. The working copy is unchanged. Readers see nothing.

| Field | Type | Required |
| --- | --- | --- |
| `version` | integer | yes |

Returns `201` with the `PostRevision`.

```json PostRevision
{
  "id": "a7b3c9d1-2e4f-4a6b-8c0d-5e1f2a3b4c7d",
  "number": 1,
  "title": "Paper Lantern 4.2",
  "summary": "Scene exports, a faster library and thirty fixes.",
  "slug": "paper-lantern-4-2",
  "category": {
    "id": "c7e1f3a5-9b2d-4c6e-8f1a-3d5b7c9e1f2a",
    "slug": "release",
    "label": "Release",
    "position": 2,
    "retired": false
  },
  "capturedFor": "checkpoint",
  "capturedBy": "paperlantern",
  "capturedAt": "2026-09-14T09:40:00Z",
  "public": false
}
```

`capturedFor` is `checkpoint`, `publication` or `schedule`. The revision response contains metadata only. The server retains its document for restoration, but does not include that document in this response.

## List revisions

`GET /api/v1/publication/posts/{id}/revisions`

Returns `200` with `{ "revisions": [PostRevision] }`, newest first.

## Restore a revision

`POST /api/v1/publication/posts/{id}/revisions/{revisionId}/restore`

Replaces the working copy with a saved revision. Any unsaved draft changes are replaced; existing revisions and the public post stay as they are. Publish the restored working copy when you want readers to see it.

| Field | Type | Required |
| --- | --- | --- |
| `version` | integer | yes |

Returns `200` with the `Post`. `version` is incremented.

## History

`GET /api/v1/publication/posts/{id}/history`

Returns `200` with `{ "actions": [PostAction] }`, newest first. Saves are not recorded. Checkpoints, publications, schedules, withdrawals, deletions and recoveries are.

```json PostActionList
{
  "actions": [
    {
      "id": "9d8c7b6a-5f4e-4d3c-8b2a-1f0e9d8c7b6a",
      "actor": "paperlantern",
      "credential": "token",
      "action": "post.checkpointed",
      "revision": 1,
      "at": "2026-09-14T09:40:00Z"
    },
    {
      "id": "8e7d6c5b-4a39-4281-9706-5e4d3c2b1a09",
      "actor": "paperlantern",
      "credential": "token",
      "action": "post.created",
      "revision": null,
      "at": "2026-09-14T09:00:00Z"
    }
  ]
}
```

## Delete a post

`POST /api/v1/publication/posts/{id}/delete`

Starts a 30-day recovery window. Contributors can delete a post only before its first publication. After that only an admin can, and only once it is withdrawn.

| Field | Type | Required |
| --- | --- | --- |
| `version` | integer | yes |

Returns `200` with the `Post`. `deletion.until` is the deadline.

## Recover a post

`POST /api/v1/publication/posts/{id}/recover`

Restores a deleted post before its deadline, with its working copy, revisions and pictures.

| Field | Type | Required |
| --- | --- | --- |
| `version` | integer | yes |

Returns `200` with the `Post`.
