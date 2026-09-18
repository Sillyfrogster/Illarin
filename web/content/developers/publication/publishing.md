# Publishing

Make a saved post public now or later, take it offline, and check whether its announcements arrived.

## Requirements

A working copy must have a title, a non-empty summary, a slug, a document with text, and for the `release` category a `release` object. If anything is missing, publishing or scheduling fails with `400 invalid` and `field` names what is missing.

Publish and schedule capture the working copy `version` you name. Republish and replace-schedule name a `revisionId` the post already has. Use the working-copy version returned by your last read or save. A stale version fails with `409 stale_version`.

## Announcement fields

Publish, schedule and replace-schedule accept the same three optional fields:

| Field | Type | Rule |
| --- | --- | --- |
| `destinationIds` | uuid[] or null | Destinations to announce to. Omit for the defaults. `[]` announces nowhere. An id outside your approval fails with `403 forbidden`. |
| `roleDestinationIds` | uuid[] | Subset of `destinationIds` that should mention their approved role. A destination with no role fails with `403 forbidden`. |
| `note` | string | Max 500 characters. Sent with the announcement. Not part of the post. |

Withdraw and republish accept `destinationIds` and `note` only.

A destination only receives the events it subscribes to. Discord destinations receive the first publication only.

## List destinations

`GET /api/v1/publication/posts/{id}/destinations`

Returns `200` with `{ "destinations": [PublicationDestinationChoice] }`. `byDefault` marks the ones a publication starts with. No addresses or secrets are included.

## Publish now

`POST /api/v1/publication/posts/{id}/publish`

Saves a revision of the working copy and makes that revision public immediately. For an already-published post, this replaces the article readers see.

| Field | Type | Required |
| --- | --- | --- |
| `version` | integer | yes |
| `destinationIds`, `roleDestinationIds`, `note` | | no |

Returns `200` with the `Post`. `status` is `published`, `publicRevisionId` is set, `byline` is snapshotted. On first publication `publishedAt` is set. On later publications `updatedPublicAt` is set and the slug stays unchanged.

The post is live at `https://blog.illarin.com/{slug}` before the response returns. Announcements are sent afterwards.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | The working copy is incomplete, or the post is withdrawn. Use republish. |
| 403 | `forbidden` | A destination or role outside your approval. |
| 409 | `stale_version` | `version` is old. |

```http
POST /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/publish
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 5b6c7d8e-9f0a-4b1c-8d2e-3f4a5b6c7d8e

{
  "version": 6,
  "destinationIds": ["e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9"],
  "note": "Paper Lantern 4.2 is out."
}
```

## Schedule

`POST /api/v1/publication/posts/{id}/schedule`

Captures the working copy as a revision and publishes it at `at`. Later edits to the working copy do not change the scheduled revision.

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `version` | integer | yes | |
| `at` | date-time | yes | RFC 3339 with an offset. Must be in the future. |
| `destinationIds`, `roleDestinationIds`, `note` | | no | |

Returns `201` with the `Post`. `schedule` is set.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | Incomplete working copy, `at` in the past, or the post is withdrawn. |
| 409 | `already_scheduled` | The post already has a pending schedule. Replace it instead. |
| 409 | `stale_version` | `version` is old. |

```http
POST /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/schedule
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 9c0d1e2f-3a4b-4c5d-8e6f-7a8b9c0d1e2f

{
  "version": 6,
  "at": "2026-09-16T14:00:00+02:00",
  "destinationIds": ["e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9"]
}
```

```json PostSchedule
{
  "id": "c2d4e6f8-1a3b-4c5d-9e7f-8a9b0c1d2e3f",
  "revisionId": "a7b3c9d1-2e4f-4a6b-8c0d-5e1f2a3b4c7d",
  "revisionNumber": 2,
  "at": "2026-09-16T12:00:00Z",
  "state": "pending",
  "createdBy": "paperlantern",
  "createdAt": "2026-09-14T10:00:00Z"
}
```

| State | Meaning |
| --- | --- |
| `pending` | Waiting for `at`. |
| `publishing` | In progress. Cannot be replaced or cancelled. |
| `published` | Done. |
| `cancelled` | Cancelled by a caller. |
| `stopped` | Stopped by Illarin. `stoppedBecause` says why, for example a revoked approval. |

## Replace a schedule

`PUT /api/v1/publication/posts/{id}/schedule`

Chooses a different saved revision or publication time for the pending schedule. To schedule your latest edits, first create a checkpoint and use its revision ID.

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `revisionId` | uuid | yes | A revision the post already has. |
| `at` | date-time | yes | RFC 3339 with an offset. Must be in the future. |
| `destinationIds`, `roleDestinationIds`, `note` | | no | |

Returns `200` with the `Post`.

| Status | Code | When |
| --- | --- | --- |
| 404 | `not_found` | No pending schedule, or no such revision. |
| 409 | `schedule_running` | The schedule is already publishing. |

## Cancel a schedule

`DELETE /api/v1/publication/posts/{id}/schedule`

No body. Returns `200` with the `Post`. The captured revision stays in history.

| Status | Code | When |
| --- | --- | --- |
| 404 | `not_found` | No pending schedule. |
| 409 | `schedule_running` | The schedule is already publishing. |

## Withdraw

`POST /api/v1/publication/posts/{id}/withdraw`

Takes a published post out of public view. The address returns `410 Gone` with a withdrawal notice. Revisions and dates are unchanged. A pending schedule is cancelled. An announcement already sent to Discord is not edited or removed.

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `version` | integer | yes | |
| `reason` | string | yes | Private audit reason, visible to people authorized to manage the post. Never shown to readers. |
| `explanation` | string | no | Public. Shown on the withdrawn address. Must differ from `reason`. |
| `destinationIds`, `note` | | no | |

Returns `200` with the `Post`. `status` is `withdrawn`.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | The post is not published, or `explanation` equals `reason`. |
| 409 | `stale_version` | `version` is old. |
| 409 | `schedule_running` | A schedule is publishing right now. |

```http
POST /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/withdraw
Authorization: Bearer ip1.…
Content-Type: application/json
Idempotency-Key: 1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b

{
  "version": 7,
  "reason": "The download link pointed at the wrong build.",
  "explanation": "This post is being corrected and will return shortly.",
  "destinationIds": []
}
```

## Republish

`POST /api/v1/publication/posts/{id}/republish`

Puts a withdrawn post back at the same address, showing the named revision. Publish and schedule are refused while a post is withdrawn; this is the only way back.

| Field | Type | Required | Rule |
| --- | --- | --- | --- |
| `version` | integer | yes | |
| `revisionId` | uuid | yes | A revision the post already has. |
| `destinationIds`, `note` | | no | |

Returns `200` with the `Post`. If `revisionId` is the revision that was public before, `updatedPublicAt` is unchanged. Otherwise it is set. `publishedAt` never changes.

| Status | Code | When |
| --- | --- | --- |
| 400 | `invalid` | The post is not withdrawn. |
| 404 | `not_found` | No such revision. |
| 409 | `stale_version` | `version` is old. |

## Deliveries

`GET /api/v1/publication/posts/{id}/deliveries`

Returns `200` with `{ "deliveries": [PostDelivery] }`: one entry per event per destination, with the last attempt. No addresses, secrets or bodies are included.

```json PostDeliveryList
{
  "deliveries": [
    {
      "id": "0a1b2c3d-4e5f-4061-8283-94a5b6c7d8e9",
      "eventId": "f1e2d3c4-b5a6-4978-8a9b-0c1d2e3f4a5b",
      "eventType": "publication.post.published.v1",
      "postId": "8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48",
      "postTitle": "Paper Lantern 4.2",
      "revisionId": "a7b3c9d1-2e4f-4a6b-8c0d-5e1f2a3b4c7d",
      "destination": "Paper Lantern community",
      "kind": "discord",
      "messageId": "1416788201234567890",
      "removed": false,
      "state": "delivered",
      "settledReason": "arrived",
      "run": 1,
      "attempts": 1,
      "occurredAt": "2026-09-14T10:30:00Z",
      "dueAt": "2026-09-14T10:30:00Z",
      "settledAt": "2026-09-14T10:30:02Z",
      "last": {
        "run": 1,
        "number": 1,
        "outcome": "delivered",
        "status": 200,
        "detail": "Discord accepted the announcement.",
        "tookMs": 412,
        "attemptedAt": "2026-09-14T10:30:02Z"
      }
    }
  ]
}
```

| State | Meaning |
| --- | --- |
| `pending` | Waiting for the next attempt. |
| `sending` | An attempt is in progress. |
| `delivered` | The destination accepted it. |
| `failed` | Stopped. `settledReason` says why. |
| `unconfirmed` | Discord accepted it without returning a message id. Never retried, because a retry could post twice. |

| Settled reason | Meaning |
| --- | --- |
| `arrived` | Delivered. |
| `exhausted` | Every retry failed. |
| `refused` | The endpoint answered a `4xx` other than `410` or `429`. |
| `gone` | The endpoint answered `410`. It receives nothing further. |
| `removed`, `disabled`, `moved` | The destination was changed by Illarin before delivery. |
| `unconfirmed` | See the `unconfirmed` state. |

Delivery state never affects whether a post is public. Contact Illarin about a failed or unconfirmed announcement. Contributor tokens cannot replay or repair deliveries.
