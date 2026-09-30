# Publish an extension from GitHub

Illarin stores the extension archive it receives and serves those bytes without changing the manifest or running the code. A verified GitHub repository can supply the first version and later versions automatically. The app still decides whether and when to install or run an extension.

## Two ways to connect a repository

**Start from GitHub.** On the upload page, choose **From GitHub**, paste the repository address and choose the release file. Illarin shows a proof code; commit it as described below, then select **Verify and import**. The latest eligible release becomes a new extension draft, and it becomes version 1 when you publish the draft. You do not upload a zip.

**Connect a published extension.** Upload the extension's zip and publish it, then connect the repository from the published extension's page.

Either way, the archive needs a supported manifest and must pass the same validation as an uploaded release. The repository must be public, and you must be able to add a file at its root.

Choose one source for every release:

- **GitHub source archive:** Illarin fetches the archive GitHub builds for the release tag.
- **Named attachment:** Enter one exact attachment name. Every imported release must have an attachment with that name. If one is missing, that import fails; Illarin never substitutes the source archive or another attachment.

Stable releases are included by default. Turn on prereleases only if you want them published on Illarin too. Choose the source and prerelease policy before verification; the saved choice applies to later releases.

## Verify repository control

Illarin displays a proof code. Add a file named `.illarin-proof` at the repository root with that code as its content. Commit the file to the repository's default branch, then select **Verify and import** on the upload page or **Verify repository** on the extension page. A repository address or GitHub author name alone does not prove control.

The proof belongs to the account that owns the Illarin extension. Another account cannot take over its source. Keep the proof file in place while verification is pending.

## Imports and versions

Illarin checks for eligible releases about once an hour after verification; **Check now** on the extension page checks sooner. It imports releases published after verification, or after the release a draft started from, validates the selected archive, then publishes a new Illarin version. The imported archive is retained exactly as fetched. Rechecking the same release does not publish it twice.

A release with an invalid archive, a missing named attachment, or a publication error appears as **Failed** on the extension page. The current published version remains available. Fix the release and select **Try again** on that import.

If the extension has unpublished edits, or is still a draft, the import is **Held**. Illarin keeps those edits and the current published version. Publish or discard your edits, then select **Resume**. Review the pending release before resuming; resuming publishes that release as the next version.

Disconnecting the repository stops future imports. It does not remove versions already published. A taken-down or deleted extension cannot be republished by an import.

## Installation remains an app decision

Publishing a release does not give any app permission to receive or run it. A connected app needs the owner's `work:receive` grant and must declare the relevant `extension-install` capability. The app installs a first send disabled, asks the owner before first run, and asks again only for permissions added by an update. [The app integration guide](/docs/app-integration#extension-install-checklist) has the complete install contract.
