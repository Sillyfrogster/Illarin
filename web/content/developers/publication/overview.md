# Publication API

Publish posts on the Illarin blog from a script, a release job or another server-side tool.

## Before you start

You need an Illarin account with a verified email address and approval to publish for your app. Ask Illarin to approve that account. Approval sets which app, categories and announcement destinations you can use.

If you want to write in a browser, use [Your posts](/admin/blog). You only need an API token when you are connecting a tool.

1. Sign in to [your API page](/admin/blog/api).
2. Under your app, choose **New token** and name the tool that will use it.
3. Copy the secret when it appears. You cannot view it again.

Your API page also has examples using your app's actual IDs. The IDs on these public pages are fictional: replace them with yours before sending a request.

## Addresses and authentication

Send API requests to the main site. Published posts appear on the blog.

| Purpose | Address |
| --- | --- |
| API base URL | `https://illarin.com/api/v1/publication` |
| Editor and token management | `https://illarin.com/admin/blog` |
| Published posts | `https://blog.illarin.com/{slug}` |
| OpenAPI contract | [openapi.yaml](/openapi.yaml) |

Examples show the full request path, starting with `/api/v1/publication`. Add `https://illarin.com` in front of it. For a self-hosted instance, use its main site address instead. The blog hostname does not accept API requests.

Send your token in the `Authorization` header on every request:

```http
GET /api/v1/publication/token
Authorization: Bearer <token>
```

This checks the token and returns its permissions, not the secret itself. Find the IDs you need in the response:

| Value | Response field |
| --- | --- |
| Categories you may use | `grant.categories[].id` |
| Your default category | `grant.defaultCategory.id` |
| Your app | `grant.app.id` |
| Allowed announcement destinations | `grant.destinations[].id` |

The server gets your account, app and byline from the token. You cannot use it to publish as someone else. See the [complete token response](/developers/publication/requests#check-your-token).

## Create your first post

Follow these three steps in order. Each response supplies the post ID and the working-copy `version` needed by the next request. This example uses the Announcement category; use its ID from your own token response.

For each action, generate a different UUID for `Idempotency-Key`. If a request times out, retry it with the same key and unchanged body. The example keys below illustrate separate actions; generate your own. See [retries and conflicts](/developers/publication/requests).

### 1. Create a private draft

```http
POST /api/v1/publication/posts
Authorization: Bearer <token>
Content-Type: application/json
Idempotency-Key: 3f8c1d2e-9a7b-4c5d-8e6f-0a1b2c3d4e5f

{
  "categoryId": "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
  "title": "Scene exports are here"
}
```

The response is `201 Created` with the new post. Keep its `id` and `version`. Nothing is public yet.

### 2. Save the writing

Replace the post ID in this path with the `id` from step 1. Send its `version` in the body. Choose an unused `slug`, which becomes the last part of the public address.

```http
PUT /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48
Authorization: Bearer <token>
Content-Type: application/json
Idempotency-Key: 4d5e6f7a-8b9c-4d0e-8f1a-2b3c4d5e6f7a

{
  "version": 1,
  "categoryId": "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
  "title": "Scene exports are here",
  "summary": "Save a scene as one file and share it with another device.",
  "slug": "scene-exports-are-here",
  "document": {
    "version": 2,
    "content": [
      {
        "type": "paragraph",
        "content": [{ "type": "text", "text": "Open a scene and choose Export to save it as one file." }]
      }
    ]
  },
  "release": null,
  "header": null,
  "socialMediaId": null
}
```

The response is `200 OK` with the saved post and a new `version`. It is still private. The outer `version` tracks edits; `document.version` describes the JSON format and stays `2`.

Prefer Markdown for the body? Save the title, summary and other fields, then [import Markdown](/developers/publication/markdown). Importing also saves privately and returns a new working-copy version.

### 3. Publish the saved post

Use the same post ID and the `version` from your last save. Review your writing before sending this request: it makes the post public immediately.

```http
POST /api/v1/publication/posts/8c2f4a71-6e3d-4b95-a1d0-3f7e9c5b2a48/publish
Authorization: Bearer <token>
Content-Type: application/json
Idempotency-Key: 5b6c7d8e-9f0a-4b1c-8d2e-3f4a5b6c7d8e

{
  "version": 2,
  "destinationIds": []
}
```

The response is `200 OK` with `status: "published"`. Readers can now open `https://blog.illarin.com/scene-exports-are-here`.

The empty `destinationIds` list publishes without sending an announcement. The post still appears on the blog and in feeds. To announce it, choose IDs from your approval or omit `destinationIds` to use its defaults. Announcements happen after publication; a failed announcement does not take the post offline.

## What to use next

| Task | Reference |
| --- | --- |
| Edit a post, add pictures or restore an earlier draft | [Writing](/developers/publication/writing) |
| Schedule, update, withdraw or republish a post | [Publishing](/developers/publication/publishing) |
| Write the body as JSON | [Post document](/developers/publication/document) |
| Write the body as Markdown | [Markdown import](/developers/publication/markdown) |
| Handle retries, errors and concurrent edits | [Requests and errors](/developers/publication/requests) |
| Receive events on your own server | [Webhooks](/developers/publication/webhooks) |

## Terms used in the reference

| Term | Meaning |
| --- | --- |
| Post | One blog entry, with a permanent address after first publication. |
| Working copy | The private version you edit. Saving it does not change what readers see. |
| Revision | A saved snapshot created by a checkpoint, publication or schedule. Restoring one copies it into the working copy. |
| Post document | The structured JSON body containing the post's paragraphs, pictures and other blocks. |
| Approval (grant) | Permission for one account to publish for one app. The API calls this a `grant`. |
| Destination | A Discord channel or webhook that can receive an announcement. Illarin configures it; you choose from the destinations in your approval. |
| Publication event | A record that a post was published, updated or withdrawn. Webhook destinations receive these events. |

## Keep your token private

- Send it only to the main site's API over HTTPS, in the `Authorization` header.
- Keep it in your tool's secret storage. Never put it in a URL, browser code, log or committed file.
- Requests with a browser `Origin` header are refused. Use a server-side tool.
- Create a separate token for each tool. Revoke an exposed or lost token on your API page and create a replacement.
- A token stops working when it expires, is revoked, or its account or approval loses access. Published posts remain online.

## API scope and compatibility

This API manages posts under the token's approval. Catalog, account and administrator operations are outside its scope. The examples use ordinary HTTP; there is no SDK to install.

The API is versioned under `/v1/`. Compatible changes add fields while retaining existing names and meanings. Error `code` values are stable; error sentences may change. The [post document has its own version](/developers/publication/document#versions).
