# Design reference decisions

Reviewed 2026-11-16. The adopted direction is defined in [MarkGo's design contract](../../DESIGN.md); these references are supporting material, not additional authorities.

## What was reviewed

[VoltAgent/awesome-design-md](https://github.com/VoltAgent/awesome-design-md) collects third-party analyses of brand websites. The five files reviewed identify themselves as `version: alpha` and `*-design-analysis`. Their contents mix design tokens and component examples with marketing-page rules. They are not official desktop application specifications, and the repository's short brand descriptions do not always describe the current file accurately.

The source links below point to the files examined. They may change upstream. No brand file has been copied wholesale into MarkGo, and future upstream changes must not automatically change this project's design contract.

| Reference | Adopt | Exclude | Role |
| --- | --- | --- | --- |
| [Claude](https://github.com/VoltAgent/awesome-design-md/blob/main/design-md/claude/DESIGN.md) | Paper/ink warmth, editorial hierarchy, separate prose and interface roles, restrained accent | Large display headlines, coral promotional bands, alternating marketing sections, brand marks and brand-font requirements | Primary visual reference |
| [Linear](https://github.com/VoltAgent/awesome-design-md/blob/main/design-md/linear.app/DESIGN.md) | Distinct neutral surface levels, quiet boundaries, a single purposeful accent, disciplined component roles | Almost-black default canvas, large negative-tracked marketing headings, pricing and screenshot-card layouts | Supporting workbench and dark-theme reference |
| [Mintlify](https://github.com/VoltAgent/awesome-design-md/blob/main/design-md/mintlify/DESIGN.md) | Document/sidebar/outline hierarchy, code and inline-text distinction, navigation that supports prose | Sky gradients, marketing illustration, mandatory three-column arrangement, pill-shaped primary marketing actions | Supporting document reference |
| [Cal.com](https://github.com/VoltAgent/awesome-design-md/blob/main/design-md/cal/DESIGN.md) | Neutral controls and an explicit spacing/radius vocabulary | Scheduling artifacts, marketing card grids, dark website footer, display-font identity | Secondary component comparison, not a second style |
| [Notion](https://github.com/VoltAgent/awesome-design-md/blob/main/design-md/notion/DESIGN.md) | No governing style adopted | The examined file includes a navy hero, purple primary actions and a broad pastel feature-card palette, unlike the quiet editing workspace implied by the collection summary | Not selected as a baseline |

Claude's warm canvas does not prescribe a beige-only product or a coral MarkGo brand. Linear's surface ladder does not prescribe a dark default. Mintlify's documentation layout does not prescribe an extra permanent sidebar. Adopt the underlying relationships and translate them into one native writing system.

## MarkGo-specific decisions

The multi-document tab shape, native window behavior, tool entry points, source-preserving editor, save/recovery semantics, composition policy and twelve-theme requirement come from MarkGo's product decisions and implementation. They are not claimed to be supplied by these reference files.

The compact workbench dimensions, typography roles and other starting targets in the contract are MarkGo design decisions for a future rendered pass. They are not measurements certified by those brands or already completed changes to this client. The [implementation record](design-system.md) identifies the current values separately.

The root contract resolves conflicts before implementation. Do not load several complete brand specifications and ask an agent to average them. Do not add landing-page components, CSS libraries, web fonts or a WebView to reproduce a reference screenshot.

## Attribution and reuse

The repository's [MIT license](https://github.com/VoltAgent/awesome-design-md/blob/main/LICENSE) is copyright 2026 VoltAgent. MarkGo credits the adapted design guidance and includes that notice in [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md).

No brand logos, proprietary fonts, illustrations, screenshots or other third-party visual assets are bundled by this change. The repository's text license is not evidence of a separate license to redistribute such assets. Use the existing MarkGo icon, original control vectors and platform fonts.
