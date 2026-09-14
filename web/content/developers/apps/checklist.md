# Conformance checklist

Check your app against every item here before it declares an extension-install capability.

These items cover extension installs. Run the checklist at the end of the [integration guide](/protocol) as well, for linking, tokens, deliveries and library reports.

## Declaring

- The app declares `chat.lumiverse:extension-install` for Spindle extensions or `app.sillytavern:extension-install` for SillyTavern ones, and only in a release that passes every item here.
- Its accepted targets include the matching format id, `extension_spindle` or `extension_sillytavern`.
- Each copy is linked with both `asset:receive` and `library:sync`.

## Installing

- The manifest is read from the top of the archive, or from inside the folder that holds every file.
- The contents of the folder that holds the manifest are what gets installed.
- Every extension installed from a delivery is recorded against its `assetId`.
- A first delivery installs disabled.
- A first install asks the owner to approve every permission its manifest lists before it first runs.
- An update stays enabled if it was, with every grant it had.
- An update asks the owner only about permissions the new manifest adds.
- An update keeps the extension's stored data, whether the owner approves the added permissions or not.
- A delivery that would replace an extension from another source, whether a clone, a copy made by hand or a different Illarin asset, is refused and leaves the installed extension untouched.
- A refusal shows the owner both extensions and where each came from.
- A delivery is acknowledged only once it is installed, and a refused delivery never is.

## Reporting

- Each extension is reported in the library when it is installed, updated or removed.
- Every library report carries the app's `applicationVersion`.
- A withheld notice from a library report or a delivery wait is stored and shown to the owner, naming the extension.
- A withheld extension stays as it is until the owner decides what to do with it.
