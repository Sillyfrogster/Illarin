# App integration

This guide is for developers adding Illarin support to an app. All API paths are relative to the exact Illarin base URL and include the public `/api` prefix. If the API behaves differently, report the mismatch.

The current protocol connects one installation of an app, rotates its
credentials, records its capabilities, lets its owner send works to it, mirrors
the app's library, and lets its owner revoke it.

## What one installation must keep

Keep one record per Illarin account and installation:

- Illarin base URL. Never assume one global host.
- Connected app ID.
- Granted permissions as last read. The owner can change them at any time.
- Current access token and its expiry.
- Current refresh token.
- The capabilities last sent to Illarin.

Store refresh tokens in the operating system credential store where one exists.
On a headless system, use a file readable only by the service account or an
equivalent secret store. Do not put tokens in logs, URLs, crash reports, update
checks, analytics, or exported app settings.
Keep this record across app upgrades. Migrate its storage in place if the app's
settings format changes; do not reconnect or replace the installation ID merely
because the app version changed.

Illarin updates capabilities without changing the installation ID, credentials or grants. A reported Lumiverse upgrade disconnect has no confirmed incident cause. A client refresh failure can also clear saved local connection state; do not describe an upgrade as proof that Illarin revoked the connection. Keep a sanitized record of the first failed request and whether the saved credential was present when investigating one.

Every installation connects independently. Do not ship a shared credential and do
not copy one when cloning an app profile, container, or virtual machine.
Clear Illarin credentials when an installation identity is cloned.

Treat the exact base URL as part of the security identity. Send a code or token
only to the server that issued it, and never merge records because two servers,
apps, or installations use the same display name.

## Security model

Illarin authenticates the account owner and the installation they approve. It
does not authenticate or endorse the software vendor. There is no client ID,
client secret, registration, allowlist, or trusted app badge.

The app and installation names on the approval screen are self-asserted and
marked unverified. Never use either name as an identity. Illarin does not
accept app-supplied HTML, CSS, JavaScript, logos, update URLs, or remote
images through this protocol.

Use HTTPS for every request except the final same-device loopback callback. Treat
all codes and tokens as opaque. Generate security values with a cryptographic
random-number generator.

This profile does not make a compromised installation safe. Access and refresh
tokens are bearer secrets, so malware that can read the app's credential
store can act as that installation until the token expires or the owner revokes
it. Manual code matching reduces cross-device consent phishing but cannot protect
an owner who approves an unsolicited matching request. Keep those limits visible
in the app instead of describing a connection as absolute proof of app
identity.

## Implementation order

1. Define one durable local installation record.
2. Choose the narrowest permissions and declare supported formats.
3. Implement browser authorization with S256 PKCE.
4. Add the manual device fallback only if the app can run where loopback
   is impossible.
5. Serialize refreshes so only one refresh is in flight per installation.
6. Add capability updates after app upgrades.
7. Refresh once after an access-endpoint `401`; treat a refresh-endpoint `401`
   as terminal and offer to connect again.
8. Run one send wait at a time and acknowledge only what you installed.
9. Report your library incrementally, with an occasional full snapshot.
10. Run the conformance checklist at the end of this guide.

## Describe the installation

Both authorization paths start with the same capabilities:

```json
{
  "appName": "Paper Lantern",
  "name": "studio workstation",
  "appVersion": "4.2.0",
  "protocolVersion": 1,
  "capabilities": ["org.example.paperlantern:media-sidecars"],
  "acceptedFormats": ["example_bundle_v2", "example_bundle_v1"],
  "permissions": ["work:receive"]
}
```

`appName` is the app's name and `name` is this installation's. The limits are
part of the wire contract:

- `appName` and `name` are printable text from 1 to 64 characters after
  trimming.
- `appVersion` is optional and at most 64 characters.
- `protocolVersion` is currently exactly `1`.
- `capabilities` and `acceptedFormats` are required arrays, even when empty.
- Each array has at most 32 unique values, each at most 64 characters.
- A capability has a namespace and name, such as
  `org.example.paperlantern:media-sidecars`.
- A format is a lowercase module ID such as `example_bundle_v2`.
- The whole JSON request body may not exceed 4 KiB. Unknown JSON fields are
  rejected.

Protocol version 1 changes only by adding optional request fields and new
response fields. An installation written before a field was added can leave it
out, and every installation must ignore response fields it does not recognise.

Use a stable reverse-domain namespace for capabilities you own. A capability is
only a claim about interoperability. It does not grant a permission, make an
unknown server feature available, or cause Illarin to run app code.

`acceptedFormats` is ordered from most to least preferred. It tells a send
which Illarin formats your app can read. The `example_*` values in this guide
are placeholders, not registered formats. Declare only IDs backed by readers
your app actually ships. Illarin exposes the formats currently offered for a
work in that work's `downloads[].format` values; there is no global format
catalog in this protocol version. The list can grow, and an unknown ID grants
nothing or selects no server-side writer.

The only permissions are:

- `work:receive`: collect works the owner sends to this installation.
- `library:sync`: report this installation's library.

`permissions` in the request is what the installation asks for. It may be empty
and it grants nothing. The owner sees both permissions unchecked, chooses any,
all or none, and can change the choice later in account settings without
connecting again. Read what was granted from `connectedApp.permissions` in the
credentials, `GET /api/v1/connected-apps/me` or a refresh; never assume the
requested set. A call that needs a permission the owner has not granted, or has
since removed, answers `403`, including with an access token issued before the
change. Handle that by telling the owner what is off, not by retrying.

**Change on 22 September 2026:** approving no longer grants the requested set.
An installation connected after this date holds only what its owner checked.

Ask only for permissions the installation will use. A capabilities update
cannot add or change permissions.

## Same-device browser authorization

This is the normal path for a desktop app. The app opens the system browser and
listens temporarily on loopback.

### 1. Prepare the callback and PKCE values

Bind a listener to `127.0.0.1` or `::1` on an available port. Pick one callback
path and keep the URI byte-for-byte for the exchange. Illarin accepts only:

```text
http://127.0.0.1:<port>/<path>
http://[::1]:<port>/<path>
```

It rejects `localhost`, alternate textual forms of the IP addresses, LAN and
public hosts, missing ports, user information, query strings, and fragments.

Generate:

- A PKCE verifier containing 43 to 128 unreserved characters.
- A state value containing 32 to 128 unreserved characters.
- The S256 challenge:

```text
BASE64URL-NO-PADDING(SHA256(ASCII(code_verifier)))
```

Keep the verifier and state in memory or protected temporary storage. Never put
the verifier in the browser URL.

### 2. Start authorization

```http
POST /api/v1/connect/authorizations
Content-Type: application/json

{
  "appName": "Paper Lantern",
  "name": "studio workstation",
  "appVersion": "4.2.0",
  "protocolVersion": 1,
  "capabilities": [],
  "acceptedFormats": ["example_bundle_v2"],
  "permissions": ["work:receive"],
  "redirectUri": "http://127.0.0.1:49152/illarin/callback",
  "state": "<random-state>",
  "codeChallenge": "<S256-challenge>",
  "codeChallengeMethod": "S256"
}
```

The response contains `authorizationUrl`, a short `userCode` such as
`BCDF-2345`, and `expiresAt`. The request expires in five minutes. Show the
`userCode` in the app, then open `authorizationUrl` in the system browser. Do
not fetch it in an embedded web view and do not log it; the URL contains a
one-use request secret.

The owner must type the `userCode` on that page before Illarin shows the
request's permissions or lets them approve it. The URL never carries the code,
and there is no way to prefill it. Keep the code on screen until the callback
arrives.

**Change on 22 September 2026:** browser authorization now needs the app to show
`userCode`. An app that opens `authorizationUrl` without showing it cannot
complete a new connection, including through the old `/api/v1/link/` paths.
Existing connections and their credentials are unaffected.

### 3. Validate the loopback callback

After approval, the browser requests the exact callback with `code` and `state`
query parameters. After denial it sends `error=access_denied` and `state`.

Before accepting a code:

1. Require the callback path you opened.
2. Compare state with the original value using a constant-time comparison.
3. Reject missing, repeated, or unexpected parameters.
4. Stop the loopback listener after one terminal callback or timeout.

### 4. Exchange the code

```http
POST /api/v1/connect/token
Content-Type: application/json

{
  "authorizationCode": "<code-from-the-callback>",
  "codeVerifier": "<original-verifier>",
  "redirectUri": "http://127.0.0.1:49152/illarin/callback"
}
```

The authorization code is one-use. The verifier, exact redirect URI, expiry,
approval, and redemption state must all match. On any failure, discard the local
authorization state and start again. Never retry a successful code exchange.

## Headless device fallback

Use this path only when the installation cannot receive loopback, such as a
remote terminal or headless server. A desktop app must use PKCE instead.

This registration-free flow adopts RFC 8628's manual-code, expiry, polling,
`slow_down`, denial, and consent-phishing protections. It is not a drop-in OAuth
Device Authorization Grant: it has no client ID or `grant_type`, and its HTTP
statuses and response bodies are the ones this guide describes.

### 1. Start the device request

```http
POST /api/v1/connect/requests
Content-Type: application/json

{
  "appName": "Paper Lantern",
  "name": "render box",
  "protocolVersion": 1,
  "capabilities": [],
  "acceptedFormats": ["example_bundle_v2"],
  "permissions": ["work:receive"]
}
```

The response contains a private `deviceCode`, a short `userCode`,
`verificationUrl`, `expiresAt`, and `interval`. The request expires in ten
minutes.

Show the URL and code separately. There is deliberately no complete prefilled
verification URL. Tell the owner to type the code, confirm that the approval
screen shows the same code, and decline any request they did not start.

### 2. Poll with finite requests

```http
POST /api/v1/connect/poll
Content-Type: application/json

{"deviceCode":"<private-device-code>"}
```

Use this state machine:

| Response | Meaning | Next action |
| --- | --- | --- |
| `200`, `status: pending` | No decision yet | Wait at least the current interval |
| `200`, `status: connected` | Connection complete | Persist the returned token pair |
| `400`, `access_denied` | Owner declined | Stop |
| `400`, `expired_token` | Request expired | Stop and offer to restart |
| `404` | Unknown or already used code | Stop |
| `429`, `slow_down` | Polling was too fast | Use `Retry-After`; the larger interval remains in force |
| Other `429` | Source rate limit | Use `Retry-After` |
| Network failure | Outcome unknown | Back off exponentially without polling before the interval |

Authorization polling is a sequence of ordinary requests. Do not hold one open,
use a WebSocket, or try to receive credentials through a callback.

## Store and rotate credentials

A successful device poll or browser exchange returns:

```json
{
  "accessToken": "ia1.…",
  "accessTokenExpiresAt": "2026-08-22T18:30:00Z",
  "refreshToken": "ir1.…",
  "connectedApp": {
    "id": "…",
    "permissions": ["work:receive"]
  }
}
```

Treat the token strings as opaque. An access token lasts 15 minutes. Send it in
the authorization header, never a query parameter:

```http
Authorization: Bearer ia1.…
```

Refresh shortly before access expiry, allowing for clock skew:

```http
POST /api/v1/connect/refresh
Content-Type: application/json

{"refreshToken":"ir1.…"}
```

Only one worker may refresh an installation at a time. On success, durably
replace the old refresh token before releasing the new access token to other
workers. The old refresh token is spent when the server commits, even if the
response is lost. If the outcome is unknown, do not blindly retry the old token;
stop the installation and ask the owner to connect again.
Coordinate refresh across every worker sharing the saved record, including old
and new app versions during an upgrade.

Replay of a replaced refresh token retained in Illarin's 90-day detection window
revokes the whole connected app and all its access tokens. An older replacement
is still rejected, but Illarin no longer keeps enough information to attribute
it to a connected app. A refresh family also expires after 90 days without authenticated
use. After a `401` from an ordinary access endpoint, serialize one refresh and
retry the original request once. A `401` from the refresh endpoint is terminal:
stop polling, sends, and sync, remove local credentials, and offer to connect
again. Never create a refresh-and-retry loop.

Secret-bearing responses use `Cache-Control: no-store`. A conforming app must
apply the same policy to its own HTTP cache and diagnostic output.

## Update capabilities after an app release

An installation can replace its capabilities without connecting again:

```http
PUT /api/v1/connected-apps/me
Authorization: Bearer ia1.…
Content-Type: application/json

{
  "appVersion": "4.3.0",
  "protocolVersion": 1,
  "capabilities": ["org.example.paperlantern:media-sidecars"],
  "acceptedFormats": ["example_bundle_v2", "example_bundle_v1"]
}
```

Send the complete replacement, not a patch. Names and granted permissions cannot
be changed here. If an upgrade needs a permission the owner has not granted, ask
the owner to turn it on in their Illarin account settings; the connection and its
credentials stay as they are.

## Add support for a new app or format

Connecting has no switch statement over apps. An app may use its own app name,
installation name, namespaced capabilities, and ordered format list without
pretending to be another product.

There are two different extension jobs:

1. **The app already reads an Illarin format.** Declare those existing format
   IDs in preference order. No Illarin-specific branding or registration is
   needed.
2. **The app needs a new file format.** Add a format module to Illarin.
   Capabilities alone cannot upload a writer or make unknown bytes safe.

An Illarin format-module contribution should:

- Choose a stable lowercase module ID that is also the format ID.
- Declare its type, read/write directions, recognition rules, role support,
  content limits, preservation namespace, and tested source formats.
- Implement the writer used for sends and, when uploads use the format, a
  reader with fail-closed recognition.
- Preserve unknown data under the module's namespace instead of silently
  deleting it.
- Register through that type's `Modules()` list; the server builds one registry
  from those lists.
- Add capability, recognition, round-trip, cross-origin, size-limit, and corpus
  tests. Do not derive fixtures from production data.

Capabilities follow the same rule: the app may declare a namespaced value
freely, but Illarin must explicitly implement any behavior that consumes it.
Unknown values remain inert. This seam gives app developers room to add support
without granting remote code or remote branding control.

## Collect sends

An owner presses send on a work's page and Illarin queues the work for one of
their connected apps. Illarin never calls out, so collection is a pull. It
needs the `work:receive` permission.

```http
POST /api/v1/sends/collect
Authorization: Bearer ia1.…
Content-Type: application/json

{"acknowledge":["<send-id>","<send-id>"]}
```

Illarin holds the request for 25 to 30 seconds. It answers `200` as soon as
there is work or a takedown notice, and `204` when the wait ends with neither.
Send `"acknowledge": []` when there is nothing to confirm; the field is required.

This is a durable queue read, not authorization polling, and the two never share
a request. Illarin checks the credential, the `work:receive` permission and the
work's own visibility again at the moment it is released, so a work
withdrawn after it was queued never arrives.

A `200` carries one entry per released send, and the
[takedown notices](#takedown-notices) waiting for this installation:

```json
{
  "sends": [
    {
      "id": "…",
      "workId": "…",
      "versionNumber": 4,
      "type": "character",
      "name": "…",
      "format": "example_bundle_v2",
      "label": "Example bundle",
      "queuedAt": "2026-08-23T18:30:00Z",
      "leaseExpiresAt": "2026-08-23T18:45:00Z",
      "files": [
        {"type": "export", "url": "https://…/send/…/export?expires=…&signature=…"},
        {"type": "picture", "url": "https://…/media/…", "mediaId": "…",
         "role": "expression", "isCover": false}
      ]
    }
  ],
  "takedowns": []
}
```

`format` is the first format in your declared order that Illarin can write for
that work, or `raw` for the creator's own uploaded file when nothing else fits.
The addresses are short-lived and signed: fetch them with ordinary `GET`s, which
makes a large file retryable rather than an all-or-nothing read. Every image the
work holds is listed, so a format that cannot carry one can still be installed
with it; a format that embeds an image hands you those bytes twice.

Rules for a conforming client:

- A send arrives at least once. Deduplicate on `id`, install idempotently, and
  acknowledge only after the work is durably stored.
- An unacknowledged send comes back when its lease runs out. After a few
  unacknowledged takes Illarin stops offering it, so acknowledge what you install.
- Open one wait at a time. A second request supersedes the first, which then
  answers `204`; two workers waiting for the same installation simply take turns.
- After a failure, back off exponentially with jitter and honour `Retry-After`.
  `429` is a rate limit and `503` means Illarin is holding as many waits as it will.
- Store `versionNumber` against `workId`. A larger one later means a newer
  version was published; fetch it again even if the bytes turn out the same.

An acknowledged send stays on record as delivered for a week, so the owner sees
on the work's page that it arrived.

### Install extensions

An extension is sent only to a connected app that declares the capability of
the app it is written for: `chat.lumiverse:extension-install` for a Spindle
extension, `app.sillytavern:extension-install` for a SillyTavern one. Without it the
page offers a download only, and a send queued before the capability was
withdrawn stops as `unsupported`. The main file is the developer's archive exactly
as uploaded, with `type` set to `extension`; accept the matching format id
(`extension_spindle` or `extension_sillytavern`) so `format` names it rather
than `raw`. The archive holds the manifest at its root or inside the one folder
that wraps everything else, as a repository download does.

The install rules the capability commits you to are in the
[extension install checklist](#extension-install-checklist): install a first send
disabled and ask for its permissions before first run, keep an update enabled and
ask only about permissions the new manifest adds, refuse a send that would
replace an extension installed from another source, and show the owner a
[takedown notice](#takedown-notices). If the owner granted `library:sync`, report
the extension in your library once it is installed. An owner may decline library
sharing and still receive and install an extension.

## Report your library

`library:sync` mirrors the app's library so the site can show its owner what is
installed and what has moved on since. It is outbound only: Illarin never writes
this and never sends it back to another installation.

```http
POST /api/v1/library/sync
Authorization: Bearer ia1.…
Content-Type: application/json

{
  "snapshot": false,
  "appVersion": "4.3.0",
  "entries": [
    {"workId": "…", "versionNumber": 4},
    {"workId": "…", "versionNumber": 1}
  ],
  "removed": ["…"]
}
```

`appVersion` is the version of the app this installation runs,
as printable text of at most 64 characters. Send it with every report, because
each report replaces the one before. An extension page lists the versions of its
app it is installed on, counting only installations that declare that app's
`extension-install` capability. A version appears only once five of them report
it, so no single installation can be picked out.

An installation written before this field leaves it out, and the protocol stays
at version 1 for it: the report is accepted, and the installation counts under
the `appVersion` in its capabilities, or under no version if they have none.

Set `snapshot` to `true` to replace the whole mirror for this installation;
anything absent is removed, so a snapshot may not also carry `removed`. Leave it
`false` to add, update and remove only what you name. At most 2000 entries and
2000 removals per request, and at most 256 KiB of body.

Leave `versionNumber` out when an installation predates the number. Illarin
records the work's current version number rather than calling the install out of
date: an installation that cannot say which version it holds has not told us it
is behind.

Report immutable work ids and never addresses, in both directions, so a creator
renaming something cannot break your state. Send incremental reports as things
change and a full snapshot occasionally, so a missed update repairs itself.
The response counts what was recorded:

```json
{"accepted": 142, "removed": 3, "ignored": 1, "takedowns": []}
```

`ignored` counts entries naming a work Illarin cannot offer, such as one that
has since been deleted.

### Takedown notices

When Illarin takes down an extension this installation reports installed, the
next library report or send wait carries a notice naming it, whichever comes
first:

```json
{"accepted": 0, "removed": 0, "ignored": 0,
 "takedowns": [{"workId": "…", "name": "Quiet Toolbox",
                "takenDownAt": "2026-09-14T06:00:00Z"}]}
```

Each takedown is carried once, so keep the notice when it arrives and show the
owner which extension it names. Whether to switch the extension off is the
owner's call. Illarin never contacts the installation to tell it: the notice
only rides on a request the installation makes. A send of that extension
still waiting to be collected stops as `withdrawn`.

An extension taken down again after its takedown was lifted carries a new notice.
Only an extension carries one, because every other type is content an app reads
rather than code it runs.

## Revocation and multiple connected apps

The account settings page lists and revokes connected apps independently.
Revoking one invalidates both of its credential classes immediately, wipes its
capabilities and the app version it reported, and deletes its pending sends,
its library mirror and any takedown notice it has not yet collected. Another
connected app on the same account keeps all of them.

Your app should provide a local disconnect action too. Until a public remote
revocation endpoint is specified, local disconnect removes local credentials
and tells the owner to revoke the matching connected app in Illarin settings.
Match it by the server-issued connected app ID and the displayed app and
installation names, not by token prefix alone.

## Conformance checklist

Before calling an integration complete, verify all of these:

- Browser authorization uses the system browser, S256 PKCE, random state, and an
  exact literal loopback callback, and shows the `userCode` the owner types.
- The callback rejects a wrong state, wrong path, missing code, denial, timeout,
  and duplicate callback.
- Device fallback is manual, shows no prefilled link, obeys the persistent
  `slow_down` interval, and stops on every terminal response.
- Required arrays are sent even when empty; capabilities stay below every bound.
- The app requests only the permissions it uses and reads what was granted
  from the credentials, which may be fewer or none.
- A `403` for a missing permission is shown to the owner and not retried.
- Credentials are isolated per installation, stored securely, redacted from
  logs, and never placed in a URL.
- Refresh is serialized and the replacement is committed atomically.
- An access-endpoint `401` causes at most one refresh and retry; a refresh-endpoint
  `401` stops every background worker and removes unusable credentials.
- Updating capabilities cannot change names or permissions.
- Two installations of the same app can connect, refresh, update, and
  disconnect without sharing state.
- Unknown capabilities and formats produce no privileged behavior.
- Response fields the installation does not recognise are ignored.
- One send wait is open at a time, `204` is handled, files are fetched
  as ordinary retryable `GET`s, and sends are acknowledged only after they
  are durably installed.
- Send ids are deduplicated, so the same send arriving twice installs once.
- Library reports name immutable work ids, stay inside every bound, carry the
  app's `appVersion`, and a snapshot carries no removals.
- An installation that declares an `extension-install` capability passes every
  item of the [extension install checklist](#extension-install-checklist).
- All tests use synthetic accounts, names, codes, and works.

### Extension install checklist

Declaring:

- The app declares `chat.lumiverse:extension-install` for Spindle extensions or
  `app.sillytavern:extension-install` for SillyTavern ones, and only in a release
  that passes every item here.
- Its accepted formats include the matching format id, `extension_spindle` or
  `extension_sillytavern`.
- Installing needs `work:receive`. Library sharing stays optional: with
  `library:sync` declined, the app still installs what it receives and does not
  ask again on every send.

Installing:

- The manifest is read from the top of the archive, or from inside the folder
  that holds every file.
- The contents of the folder that holds the manifest are what gets installed.
- Every extension installed from a send is recorded against its `workId`.
- A first send installs disabled.
- A first install asks the owner to approve every permission its manifest lists
  before it first runs.
- An update stays enabled if it was, with every grant it had.
- An update asks the owner only about permissions the new manifest adds.
- An update keeps the extension's stored data, whether the owner approves the
  added permissions or not.
- A send that would replace an extension from another source, whether a
  clone, a copy made by hand or a different Illarin work, is refused and leaves
  the installed extension untouched.
- A refusal shows the owner both extensions and where each came from.
- A send is acknowledged only once it is installed, and a refused send never
  is.

Reporting:

- Each extension is reported in the library when it is installed, updated or
  removed.
- Every library report carries the app's `appVersion`.
- A takedown notice from a library report or a send wait is stored and shown
  to the owner, naming the extension.
- A taken-down extension stays as it is until the owner decides what to do with it.

## Compatibility names

Illarin says work where it said asset, type where it said kind, and follow where
it said watch. A work's version number is now the one number that says a newer
version exists: it replaces the content generation, and it moves with every
published version, even one that left the file alone. A send lists its `files`
where it listed `artifacts`.

Connecting says app where it said platform or application, connected app where
it said linked instance, connection request where it said link request,
permission where it said scope, capabilities where it said declaration, format
where it said export target, and send where it said delivery. On the wire:

| Old name | New name |
| --- | --- |
| `/api/v1/link/requests`, `/poll`, `/authorizations`, `/token`, `/refresh` | the same ends under `/api/v1/connect/` |
| `/api/v1/instances`, `/api/v1/instances/me` | `/api/v1/connected-apps`, `/api/v1/connected-apps/me` |
| `/api/v1/deliveries/collect` | `/api/v1/sends/collect` |
| `/delivery/<id>/export` | `/send/<id>/export` |
| `applicationName`, `instanceName`, `applicationVersion` | `appName`, `name`, `appVersion` |
| `acceptedTargets` | `acceptedFormats` |
| `scopes` | `permissions` |
| the `asset:receive` scope | the `work:receive` permission |
| `instance` in a credentials response, and its `linkedAt` | `connectedApp`, and its `connectedAt` |
| poll `status: linked` | poll `status: connected` |
| `deliveries` in a collect response | `sends` |
| the `X-Illarin-Export-Target` header on a file | `X-Illarin-Format` |
| `withheld` in a collect response or a library report result, and each notice's `withheldAt` | `takedowns`, and `takenDownAt` |

For now the old names still answer beside the new ones: every `/api/v1/assets`
path still works at its `/api/v1/works` address, and every old path above still
works. A request may use either name for a field or a permission, and a
response carries both. A send carries `assetId`, `kind`, `contentGeneration`
and `artifacts` next to `workId`, `type`, `versionNumber` and `files`, and a
library report may name a work by `assetId` and its version by
`contentGeneration`. The old poll path still answers `status: linked`.

The old names stop answering on 2026-11-18. Move to the new ones before then.
