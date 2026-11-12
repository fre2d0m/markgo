# Design system

## Direction

MarkGo is a writing tool, not a dashboard. The document is the visual center; navigation and AI support it without competing for attention. Prefer typography, line length, spacing, and quiet state changes over decorative effects.

## Workbench

The workbench has a 56 DIP header, a 224 DIP document sidebar, a flexible editor, and a 28 DIP status bar. The sidebar switches between files and the current document's outline. Focus mode hides navigation, formatting controls, and status without changing the document buffer. AI controls are absent from the writing interface.

The editor uses 16 DIP system text with 1.75 line height, a centered 760 DIP maximum content width, soft wrapping, and no line-number gutter or active-line background. Source marks are revealed for the element containing a caret or selection. The 42 DIP formatting toolbar uses 32 DIP controls which preserve the selection and return keyboard focus to writing. The compact find panel uses the same 32 DIP input and button heights.

Read mode renders native Markdown typography, tables, code blocks, and local images. It is a separate reading view; do not describe it as directly editable rich text. The editable view currently uses fixed line metrics.

## Theme tokens

Use MyGo theme values for ordinary surfaces and controls. `internal/app` customizes the shared accent and related states:

| Token | Value |
| --- | --- |
| Accent | `#6c5ce7` |
| Accent hover | `#7c6cf0` |
| Accent pressed | `#5948cf` |
| Accent text | `#ffffff` |
| Control radius | 8 DIP |
| Selection | Accent at 24% opacity |
| Focus | Accent at 58% opacity |

Background, surface, border, text, and muted text continue to follow the system's light or dark appearance. New components should consume these tokens instead of introducing isolated colors.

## Interaction and accessibility

- Every control needs a useful accessible label and keyboard behavior.
- Preserve input-method composition, selection, undo, and focus when customizing editor visuals.
- Use animation only when it clarifies state or preserves spatial context; respect reduced-motion settings.
- Do not communicate modified, error, or AI-proposed states by color alone.
- AI output is visually and behaviorally a proposal until the user accepts it.

## Visual verification

UI changes need `ui.Tester` behavior coverage and a real application screenshot. Record the operating system and display scale. Exercise both panes open and closed, long labels, narrow supported windows, and light/dark appearance when the change can affect them.
