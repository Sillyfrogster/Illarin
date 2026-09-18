# Webhooks

Receive a signed notification on your server when a blog post is published, updated or withdrawn.

## Setup

You only need this page if you run a server that receives publication events. Ask Illarin to configure your HTTPS endpoint as an announcement destination; contributor tokens cannot create destinations. Store the signing secret provided during setup on your server. It is separate from your publication API token. You choose which destinations receive each publication; see [Announcement fields](/developers/publication/publishing#announcement-fields).

Before a destination is enabled, Illarin sends a verification request. Verify its signature, then respond with a `2xx` status and the challenge value to enable it.

| Request | Response |
| --- | --- |
| Signed `POST` with body `{ "id", "type": "publication.endpoint.verification.v1", "challenge", "sentAt" }` | `2xx` with body `<challenge>` or `{ "challenge": "<challenge>" }` |

The verification request is signed like every event. Verify it the same way.

## Events

| Event | Sent when |
| --- | --- |
| `publication.post.published.v1` | A post becomes public for the first time, or a withdrawn post is republished. |
| `publication.post.updated.v1` | An already-public post is published again with changes. |
| `publication.post.withdrawn.v1` | A post is withdrawn. Its address now returns `410`. |

Saves, checkpoints, schedules, deletions, recoveries and admin actions send nothing.

## Request

```http
POST /hooks/illarin
Content-Type: application/json
User-Agent: Illarin-Publication/1
webhook-id: 0a1b2c3d-4e5f-4061-8283-94a5b6c7d8e9
webhook-timestamp: 1789378200
webhook-signature: v1,K2h0dGdqQm5hRWx2NkpFVDk3RExoWmZzVHNvVE5GbGhHSkhtVjBoUDNzOD0=

{
  "id": "f1e2d3c4-b5a6-4978-8a9b-0c1d2e3f4a5b",
  "type": "publication.post.published.v1",
  "occurredAt": "2026-09-14T10:30:00Z",
  "note": "Paper Lantern 4.2 is out.",
  "post": {
    "id": "8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48",
    "revisionId": "a7b3c9d1-2e4f-4a6b-8c0d-5e1f2a3b4c7d",
    "title": "Paper Lantern 4.2",
    "summary": "Scene exports, a faster library and thirty fixes.",
    "category": { "slug": "release", "label": "Release" },
    "url": "https://blog.illarin.com/paper-lantern-4-2",
    "publishedAt": "2026-09-14T10:30:00Z",
    "updatedAt": null,
    "release": {
      "app": { "slug": "paper-lantern", "name": "Paper Lantern", "url": "https://blog.illarin.com/app/paper-lantern" },
      "version": "4.2.0",
      "url": "https://paperlantern.example/4.2"
    },
    "byline": {
      "handle": "paperlantern",
      "name": "Paper Lantern",
      "url": "https://illarin.com/@paperlantern",
      "app": { "slug": "paper-lantern", "name": "Paper Lantern", "url": "https://blog.illarin.com/app/paper-lantern" }
    }
  }
}
```

| Header | Meaning |
| --- | --- |
| `webhook-id` | Delivery id. Same on every attempt and replay of one event to one destination. Deduplicate on it. |
| `webhook-timestamp` | Unix seconds when this attempt was signed. New on every attempt. |
| `webhook-signature` | One or more space-separated signatures, each `v1,<base64>`. |

| Body field | Meaning |
| --- | --- |
| `id` | Event id. |
| `type` | Event name. |
| `occurredAt` | When the transition happened. Order events by this. |
| `note` | The publisher's note for this announcement, if any. |
| `post.id`, `post.revisionId` | The post and the exact revision. |
| `post.title`, `post.summary`, `post.category` | Public metadata. |
| `post.url` | Permanent address. Link readers here. |
| `post.socialImageUrl` | Present when the post uploaded its own sharing picture. |
| `post.publishedAt`, `post.updatedAt` | Public dates. |
| `post.release` | Present for release posts: app, version, release notes URL. |
| `post.byline` | Author identity as it stood at first publication. |

The body never contains the article text.

## Verify

```plain
signed = webhook-id + "." + webhook-timestamp + "." + body
webhook-signature = "v1," + base64(HMAC-SHA256(key, signed))
key = base64decode(secret without the "whsec_" prefix)
```

1. Read the raw body bytes. Do not re-serialize.
2. Reject if `webhook-timestamp` is more than 5 minutes from your clock.
3. Compute the expected signature. Compare against each value in `webhook-signature` with a constant-time comparison. Accept if any matches.
4. Then parse the body.

During secret rotation the header carries two signatures, new secret first. Accepting any match handles rotation.

```python
import base64, hashlib, hmac, time

def accepts(headers, body: bytes, secret: str) -> bool:
    sent = int(headers["webhook-timestamp"])
    if abs(time.time() - sent) > 300:
        return False
    key = base64.b64decode(secret.removeprefix("whsec_"))
    signed = f"{headers['webhook-id']}.{sent}.".encode() + body
    want = "v1," + base64.b64encode(hmac.new(key, signed, hashlib.sha256).digest()).decode()
    return any(hmac.compare_digest(want, given) for given in headers["webhook-signature"].split())
```

## Responding

Respond `2xx` as soon as the event is stored. Illarin waits 10 seconds and reads at most 8 KiB of the response.

| Response | Effect |
| --- | --- |
| `2xx` | Delivered. |
| `429` | Retried after `Retry-After`. |
| `408`, `425`, `5xx`, timeout, connection failure | Retried on the schedule below. |
| `410` | The destination is disabled. Nothing further is sent. |
| `3xx` | Not followed. Treated as refused. |
| Other `4xx` | Refused. No retries until an admin replays. |

## Retries

Your endpoint may receive the same event more than once. Retry delays after a failed attempt: 5 s, 5 min, 30 min, 2 h, 5 h, 10 h, 14 h, 20 h, 24 h, each with jitter. Ten attempts over about three days, then the delivery is `exhausted`. An admin replay starts a new attempt sequence under the same `webhook-id`.

Requirements for your endpoint:

- Deduplicate on `webhook-id`. Keep seen IDs for at least four days to cover automatic retries. An admin can replay older deliveries, so retain IDs longer if processing an event twice would cause harm.
- Expect the same event twice when your `2xx` is lost in transit.

## Ordering

No ordering is guaranteed between events or destinations. A retried `published` event can arrive after the `withdrawn` event that followed it. Order by `occurredAt` and treat `post.revisionId` as the identity of the edition. Discard an event older than one you already applied for the same post.

## Outbound policy

Illarin sends only to public `https` addresses on port 443, resolves the address on every attempt, follows no redirects, and sends `User-Agent: Illarin-Publication/1`. Destination addresses and secrets are never exposed through the API.
