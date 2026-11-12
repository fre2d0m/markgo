# Security policy

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** for this repository. Do not open a public issue containing credentials, private documents, model keys, or sensitive reproduction data.

Include the affected commit or release, operating system, minimal reproduction, expected and actual behavior, and potential impact. A maintainer will coordinate the fix and disclosure.

## Supported versions

MarkGo has not published a release. Security fixes are applied to `main`. Supported release lines will be listed after releases exist.

## Security model

Markdown files, linked resources, and model output are untrusted data.

- Preview and export must not silently execute active content from a document.
- File operations must protect unsaved changes and handle external modifications explicitly.
- Document content must not be sent to a model or network service without the user's chosen context and provider.
- Model output must be shown for review before it changes a document, and accepted edits must remain undoable.
- Credentials must use the operating system's secret store when provider settings are implemented.
- Logs and issue attachments must not contain document contents or secrets unless the user deliberately includes them.

The current foundation has no network provider or real file persistence. These requirements apply as those capabilities are introduced.
