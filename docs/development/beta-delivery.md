# Beta acceptance gates

Rust + GPUI is the sole client. Source/build migration is not release certification. AI remains deferred.

| Area | Implemented boundary | Acceptance still required |
| --- | --- | --- |
| Documents | Tabs/folders/outline/recent; guarded files, recovery and exit | Native dialogs, cancellation, external-change failures, crash/restart per OS |
| Editing | Headings/inline/links, lists/tasks, quotes, top-level fences | Installed IME, complex nested Markdown and large-document latency |
| Reading/output | Native tables and local images; remote images blocked | Editable tables/images, insertion and export remain separate product work |
| Design | Shared scale, vectors, twelve themes, prose settings | Native keyboard/accessibility, fractional scale, reduced motion and transition refinement |
| Distribution | Cargo builds, native-host bundles, core/desktop CI | Signed packages, installation/association, complete binary notices and provenance |

Recovery is not document autosave, encryption or cloud sync. Normal exit waits for a checkpoint; forced termination and OS shutdown timeout can lose the latest interval. External writes can race a final file check. Track release gaps as observable issues; do not claim beta quality from dependency popularity or a passing build.
