# Reporting back

Tell Illarin which extensions are installed and which version of your app runs them, and show the owner when Illarin withholds one.

## Report each installed extension

Send a library report whenever an extension is installed, updated or removed, and a full snapshot now and then:

```http
POST /api/v1/library/sync
Authorization: Bearer <access token>
Content-Type: application/json

{
  "snapshot": false,
  "applicationVersion": "1.3.0",
  "entries": [
    {
      "assetId": "3a9f5c2e-8b1d-4e7a-9c6f-2d4b8e1a7c35",
      "contentGeneration": 3
    }
  ],
  "removed": ["5c1b7e9d-2f4a-4b8c-9e3d-6a0f2b8c4d71"]
}
```

The reply counts what was recorded:

```json LibraryReportResult
{
  "accepted": 1,
  "removed": 1,
  "ignored": 0,
  "withheld": []
}
```

- Name each extension by its `assetId`, never by its name.
- When the owner removes an extension, list its `assetId` in `removed`.

## Report your app version

Send `applicationVersion` with every report: the version of your app this copy runs, as printable text of at most 64 characters. Each report replaces the one before.

An extension's page lists the versions of its app it is installed on, such as "Installed by readers on Lumiverse 1.3.0 and 1.2.0." It counts only copies that declare the app's `extension-install` capability, and lists a version only once at least five of them report it, so no single copy can be picked out. The page shows versions, never counts.

A copy running a release from before this field leaves it out. Illarin still accepts the report and counts that copy under the `applicationVersion` in its declaration, or under no version if the declaration has none.

## Show withheld notices

Illarin can withhold a listed extension, which takes it off the site. Every copy of your app that reports it installed then gets a notice on its next library report or delivery wait, whichever comes first. On a library report, the notice is in the reply:

```json LibraryReportResult
{
  "accepted": 1,
  "removed": 0,
  "ignored": 0,
  "withheld": [
    {
      "assetId": "3a9f5c2e-8b1d-4e7a-9c6f-2d4b8e1a7c35",
      "name": "Quiet Toolbox",
      "withheldAt": "2026-09-14T06:00:00Z"
    }
  ]
}
```

On a delivery wait, it comes beside any work. A wait with a notice and no work answers `200` with no deliveries:

```json DeliveryWorkList
{
  "deliveries": [],
  "withheld": [
    {
      "assetId": "3a9f5c2e-8b1d-4e7a-9c6f-2d4b8e1a7c35",
      "name": "Quiet Toolbox",
      "withheldAt": "2026-09-14T06:00:00Z"
    }
  ]
}
```

- Store the notice as soon as it arrives. Illarin sends each withhold once, and does not send it again even if the reply never reached you.
- Show the owner which extension it names. Whether to switch the extension off or remove it is the owner's call, so leave it as it is until they choose.
- Illarin never contacts your app to deliver a notice. It rides only on a request your app makes.
- A delivery of the withheld extension that has not been collected stops as `withdrawn` and never arrives.
- If Illarin lifts the withhold and later withholds the extension again, a new notice comes.
- Revoking the link deletes any notice that copy has not collected.

Only extensions carry notices. Every other kind is content your app reads, not code it runs.
