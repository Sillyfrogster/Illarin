# Installing and updating

What your app does with an extension delivery, from its first install to later updates, and when to refuse one.

## Know where each extension came from

Record the delivery's `assetId` against every extension you install from Illarin. That record decides what a later delivery is. Compare the delivered extension with anything already installed under the same name, meaning the name your app installs it under, such as the `identifier` in `spindle.json`:

| Already installed under that name | The delivery is |
| --- | --- |
| Nothing | A [first install](#install-a-first-delivery-switched-off) |
| An extension installed from the same `assetId` | An [update](#keep-an-update-running) |
| An extension installed another way, or from a different `assetId` | A [replacement, which you refuse](#refuse-to-replace-an-extension-from-another-source) |

Illarin does not reserve extension names. Two developers can list extensions that declare the same `identifier`, so the name alone never tells you where an extension came from.

## Install a first delivery switched off

1. Install the extension disabled. It does not run until its owner turns it on.
2. Acknowledge the delivery once its files are stored and it is recorded, even though it is off. The reader's Illarin page then says it is installed and switched off until they approve it.
3. Before it first runs, show the owner every permission its manifest asks for and ask them to approve. For Spindle that is the `permissions` list, including the permissions only an administrator can grant. SillyTavern extensions list no permissions, so ask the owner to turn the extension on.

## Keep an update running

An update is a delivery of an asset you already installed, usually with a larger `contentGeneration` than the one you hold.

- Replace the extension's files with the archive's contents.
- Leave it as it was: enabled if it was enabled, disabled if it was disabled.
- Keep every permission the owner already granted.
- Ask the owner only about permissions the new manifest adds. Until they approve, the extension runs with the grants it already had.
- Keep the extension's stored data, whether the owner approves the added permissions or not.
- Acknowledge the delivery once the new files are in place.

A reader can send the version you already hold. When `contentGeneration` matches yours, install it again or skip it, and acknowledge it either way.

## Refuse to replace an extension from another source

Refuse a delivery that would replace an extension that did not come from the same `assetId`: one cloned from its repository, copied in by hand, or installed from a different Illarin asset.

- Leave the installed extension exactly as it is.
- Show the owner both. For the installed extension, say where it came from, such as its repository address or the Illarin asset it was installed from. For the delivered one, give its `name` and its Illarin page, `/a/{assetId}` on the Illarin site.
- Do not acknowledge the delivery, because nothing was installed.

The owner decides what happens next. If they remove the installed extension, the delivery installs as a first install the next time it comes round.

An unacknowledged delivery comes round again once its lease runs out, at `leaseExpiresAt`. Show the refusal once per delivery `id`, not on every take. After a few takes without an acknowledgement, Illarin stops offering it, and the reader's page says the application kept taking the delivery without installing it.
