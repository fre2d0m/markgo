# Rust client decision

Date: 2026-11-16.

The product owner selected a direct replacement with Rust and GPUI Kit. The root Cargo package is the sole executable/build graph. Previous source, dependencies, packaging configuration, CI and framework-specific instructions are removed. Earlier dual-client gates no longer apply.

[DESIGN.md](../../DESIGN.md) remains authoritative: quiet native writing, multiple tabs, source fidelity, twelve themes and deliberate feedback. GPUI supplies rendering and native services; MarkGo owns its document model, editing and safety.

GPUI Kit is pinned to 0.9.1 with the upstream engine. Astrolith informed architecture discussion only; no code/assets were imported. Dependencies and toolkit assets retain their own attribution requirements in [THIRD_PARTY.md](../../THIRD_PARTY.md).

The replacement includes rich direct editing, reading/source, slash commands, advanced search, tabs/folders/outline/recent paths, guarded files, preferences, recovery and exit decisions. [Architecture](overview.md) defines boundaries. [Beta gates](../development/beta-delivery.md) distinguish implementation from platform acceptance; switching the default client does not waive native testing.
