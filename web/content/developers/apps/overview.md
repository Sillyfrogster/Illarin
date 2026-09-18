# Connect an app to Illarin

Let readers send an extension from its Illarin page straight to their own copy of your app.

## How it works

Illarin lists Lumiverse and SillyTavern extensions. A reader who has linked their copy of your app to Illarin can press **Install** on an extension's page and choose that copy. Your app collects the delivery, installs it and reports it back. The reader watches it on the page as it goes from queued to picked up to installed.

Illarin has no install code for any app. It hands your app the developer's archive exactly as they uploaded it, and your app installs it by the rules on these pages. Until an app follows them, the pages of its extensions offer a download only.

Illarin never runs an extension and does not review one before listing it. It never contacts your app either: everything on these pages happens in a request your app makes.

## Before you start

These pages build on the linked-instance protocol. The [integration guide](/protocol) covers linking a copy of your app, keeping its tokens, waiting for deliveries and reporting its library. Build those first.

Link with both scopes:

| Scope | What it lets an extension install do |
| --- | --- |
| `asset:receive` | Collect the extensions readers send. |
| `library:sync` | Report what is installed and your app version, and receive withheld notices. |

A declaration update cannot add a scope, so a copy linked without `library:sync` has to link again.

## Declare the capability

Your app says it installs extensions by declaring the capability for its own extension format. The part before the colon is the reverse domain your app already puts before its capabilities.

| App and manifest | Capability | Format id |
| --- | --- | --- |
| Lumiverse, `spindle.json` | `chat.lumiverse:extension-install` | `extension_spindle` |
| SillyTavern, `manifest.json` | `app.sillytavern:extension-install` | `extension_sillytavern` |

Declare it in the release that ships installs, with the format id among your accepted targets. A declaration update replaces the whole declaration, so keep every target you already accept:

```http
PUT /api/v1/instances/me
Authorization: Bearer <access token>
Content-Type: application/json

{
  "applicationVersion": "1.3.0",
  "protocolVersion": 1,
  "capabilities": ["chat.lumiverse:extension-install"],
  "acceptedTargets": ["extension_spindle", "chara_card_v3", "lorebook"]
}
```

A reader sees **Install** only for copies that declare the capability of the extension's app. The application name you send makes no difference, and a capability not in the table above grants nothing. Lumiverse and SillyTavern extensions do not run in each other's app, so an extension made for both is listed twice, and each listing goes only to its own app.

Declaring the capability is a promise to follow every rule on these pages. Illarin cannot check it, so check your app against the [conformance checklist](/developers/apps/checklist) first. If a later release stops installing extensions, declare without it: a delivery still queued for that copy then stops as `unsupported`.

## Receive an extension

Extensions arrive through the ordinary delivery wait. An extension's delivery has `kind` set to `extension`, and `format` names its manifest when you accept that format id:

```json DeliveryWorkList
{
  "deliveries": [
    {
      "id": "7d1e2f3a-4b5c-4d6e-8f7a-9b0c1d2e3f4a",
      "assetId": "3a9f5c2e-8b1d-4e7a-9c6f-2d4b8e1a7c35",
      "contentGeneration": 3,
      "kind": "extension",
      "name": "Quiet Toolbox",
      "format": "extension_spindle",
      "label": "Spindle extension",
      "queuedAt": "2026-09-14T09:30:00Z",
      "leaseExpiresAt": "2026-09-14T09:45:00Z",
      "artifacts": [
        {
          "kind": "export",
          "url": "https://illarin.com/delivery/7d1e2f3a-4b5c-4d6e-8f7a-9b0c1d2e3f4a/export?expires=…&signature=…"
        }
      ]
    }
  ],
  "withheld": []
}
```

Fetch the `export` artifact with an ordinary `GET` before `leaseExpiresAt`. If you do not accept the format id, `format` is `raw` and the artifact holds the same bytes. Any `picture` artifacts are pictures from the extension's Illarin page; you do not need them to install it.

## Read the archive

The export is a `.zip`, and it is the archive the developer uploaded, byte for byte. Illarin never edits the manifest and never adds a file, so what you read is what the developer wrote.

The manifest is in one of two places:

- At the top of the archive.
- Inside the folder that holds every file in the archive. A repository's **Download ZIP** and a folder zipped by the operating system both arrive wrapped like this, sometimes more than one folder deep.

Install the contents of the folder that holds the manifest, not the folders wrapped around it. The archive holds what a clone of the extension's repository would: source, `dist/` or both. Build it as you would after a clone.

Illarin refuses an upload that fails the app's own install checks:

| App | Illarin checks |
| --- | --- |
| Lumiverse | `spindle.json` has `version`, `name`, `author`, `github`, `homepage`, `permissions` and an `identifier` matching `^[a-z][a-z0-9_]*$`, and the entry files, or the `src/backend.ts` and `src/frontend.ts` Lumiverse builds from, exist. |
| SillyTavern | `manifest.json` has `display_name`, `js` and `author`, and the file named by `js` exists. |

An extension archive holds at most 4,096 files and 32 MB. Check it yourself all the same, as your app would check a clone, and refuse any path that climbs out of the folder you install into.

## Pages in this section

| Task | Page |
| --- | --- |
| Install a first delivery, update one, or refuse a replacement | [Installing and updating](/developers/apps/installing) |
| Report what is installed and your app version, and show withheld notices | [Reporting back](/developers/apps/reporting) |
| Check your app against every rule | [Conformance checklist](/developers/apps/checklist) |
