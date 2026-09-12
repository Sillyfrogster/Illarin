# Requests and errors

Send requests safely, retry without repeating an action, and handle errors or edits made in another session.

## Conventions

- Send JSON request bodies with `Content-Type: application/json`. Picture uploads use `multipart/form-data` instead.
- Timestamps are RFC 3339. Send any offset; responses are UTC.
- IDs are UUIDs. Read them from responses; the sample IDs in this reference are fictional.
- Unknown keys inside a post document are refused. Unknown fields elsewhere in a request body are ignored.

## Check your token

`GET /api/v1/publication/token`

Returns `200` with token metadata and its publication approval. It never returns the token secret. Use the categories, app and destinations in this response when creating and publishing posts.

```json PublicationCredential
{
  "token": {
    "id": "1b2c3d4e-5f60-4718-a9b0-c1d2e3f4a5b6",
    "grantId": "2c3d4e5f-6071-4829-b0c1-d2e3f4a5b6c7",
    "name": "Release robot",
    "prefix": "k7m2p9qx",
    "createdAt": "2026-09-01T10:00:00Z",
    "expiresAt": null,
    "lastUsedAt": "2026-09-14T08:12:30Z",
    "revokedAt": null,
    "active": true
  },
  "grant": {
    "id": "2c3d4e5f-6071-4829-b0c1-d2e3f4a5b6c7",
    "holder": {
      "handle": "paperlantern",
      "displayName": "Paper Lantern",
      "avatar": null,
      "restricted": false
    },
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
    "categories": [
      {
        "id": "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
        "slug": "announcement",
        "label": "Announcement",
        "position": 1,
        "retired": false
      },
      {
        "id": "c7e1f3a5-9b2d-4c6e-8f1a-3d5b7c9e1f2a",
        "slug": "release",
        "label": "Release",
        "position": 2,
        "retired": false
      }
    ],
    "defaultCategory": {
      "id": "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
      "slug": "announcement",
      "label": "Announcement",
      "position": 1,
      "retired": false
    },
    "destinations": [
      {
        "id": "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9",
        "name": "Paper Lantern community",
        "kind": "discord",
        "state": "active",
        "events": ["publication.post.published.v1"],
        "role": "",
        "byDefault": true
      }
    ],
    "destinationsInherited": true,
    "grantedAt": "2026-08-20T15:30:00Z",
    "revokedAt": null,
    "active": true
  }
}
```


## Idempotency

An idempotency key identifies one intended action, such as creating one draft. Send it in the `Idempotency-Key` header on every request that changes something. Generate a new UUID for the next action, but reuse the same key and body when retrying an action whose result you did not receive.

```http
POST /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/revisions
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 4d5e6f7a-8b9c-4d0e-8f1a-2b3c4d5e6f7a

{ "version": 2 }
```

| Rule | Detail |
| --- | --- |
| Format | 8–200 characters. Use a UUID per intended action. |
| Scope | One token and one operation (method and path template). The same key on another operation or token is unrelated. |
| Same key, same body | Returns the original status and body. No work is done. Bodies are compared byte for byte. |
| Same key, different body | `409 idempotency_mismatch`. No work is done. |
| Same key, first request still running | `409 idempotency_in_progress`. Retry after a pause. |
| Lifetime | 24 hours. |
| Server errors | Responses of `500` and above are not stored. A retry runs the request again. |
| `GET` | Header ignored. |

## Version conflicts

Saving, checkpointing, publishing, scheduling, restoring, withdrawing, republishing, deleting and recovering a post require the working-copy `version` you last read. Replacing a schedule uses a `revisionId`; cancelling a schedule and uploading a picture do not take a working-copy version. If the working copy has changed since, the response is `409 stale_version` with the current version:

```json PostConflict
{
  "error": "This post was saved in another session. Copy any unsaved text, then reload to edit the latest version.",
  "code": "stale_version",
  "field": "version",
  "version": 7,
  "updatedAt": "2026-09-14T11:02:15Z"
}
```

Read the post again and compare it with your unsaved changes. Merge the changes you want to keep, then send the current version with a new idempotency key. Simply replacing the version number would overwrite the other edit. A retry with a key that already returned a conflict will return that same conflict.

`already_scheduled` and `schedule_running` use the same shape without `version`.

## Error shape

Publication API errors use this body. A proxy or a network failure may return a different response, so check the status and content type before parsing JSON:

```json PublicationError
{
  "error": "Your approval does not cover Article posts.",
  "code": "category_refused",
  "field": "categoryId"
}
```

| Field | Meaning |
| --- | --- |
| `code` | Stable. Branch on this. |
| `error` | Human-readable. May change. |
| `field` | Optional. The request field concerned, or a path into the document such as `document.content.3.content.0.marks.0.href`. |

Errors never name another account, approval or token.

## Error codes

| Code | Status | Meaning |
| --- | --- | --- |
| `unauthenticated` | 401 | Missing or unknown token, a browser `Origin` header, or a token used outside `/api/v1/publication/`. |
| `token_expired` | 401 | The token passed its expiry. |
| `token_revoked` | 401 | The token was revoked. |
| `grant_revoked` | 401 | The approval behind the token was revoked. |
| `forbidden` | 403 | Not allowed: another approval's post, a destination or role outside your approval, a slug change on a published post, or an admin-only action. |
| `not_found` | 404 | No such post, revision, picture, schedule or category. |
| `invalid` | 400, 413 | A field or the document is wrong. `field` names it. `413` for a picture or import over its size limit. |
| `category_refused` | 400 | The category exists but is outside your approval. |
| `stale_version` | 409 | The working copy changed. Body carries the current version. |
| `already_scheduled` | 409 | A schedule is already pending. |
| `schedule_running` | 409 | The schedule is publishing now. |
| `idempotency_mismatch` | 409 | The key was used with a different body. |
| `idempotency_in_progress` | 409 | The first request with this key has not finished. |
| `rate_limited` | 429 | Over the rate limit. `Retry-After` is set. |
| `server_error` | 500, 503 | Illarin could not complete the request. Retry with the same idempotency key. |

## Rate limits

Per token, per operation class, in one-minute windows that start with the first request:

| Class | Requests per minute |
| --- | --- |
| Read (`GET`) | 300 |
| Write (all other methods) | 60 |
| Picture upload | 20 |

Over the limit: `429 rate_limited` with `Retry-After` in seconds. Requests are not queued. Limits are per token; other tokens and the web editor are unaffected.

Retry `429`, `server_error` and network failures with exponential backoff and jitter, reusing the same idempotency key.
