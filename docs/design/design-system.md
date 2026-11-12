# Design system

## Direction

MarkGo is a writing tool, not a dashboard. The document is the visual center; navigation and AI support it without competing for attention. Prefer typography, line length, spacing, and quiet state changes over decorative effects.

## Workbench

The current shell has a 56 DIP header, a 224 DIP document sidebar, a flexible editor, an optional 286 DIP AI panel, and a 28 DIP status bar. The sidebar and AI panel can be hidden without changing the document buffer.

The editor uses 15 DIP text with 1.75 line height, soft wrapping, no line-number gutter, and source marks revealed on the caret's lines. The current live preview is an engineering milestone, not the final typographic renderer.

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
