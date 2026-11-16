use super::theme::Palette;
use super::views::RowLabel;
use crate::{
    document::{Selection, utf16_to_byte},
    editing::Editor,
};
use gpui_kit::{
    component::{
        ActiveTheme,
        button::{Button, ButtonVariants},
    },
    prelude::*,
    *,
};
use std::{cell::RefCell, ops::Range, rc::Rc};

actions!(
    markgo_writer,
    [
        Left,
        Right,
        Up,
        Down,
        SelectLeft,
        SelectRight,
        SelectUp,
        SelectDown,
        Home,
        End,
        SelectHome,
        SelectEnd,
        DocumentHome,
        DocumentEnd,
        SelectAll,
        Backspace,
        Delete,
        Enter,
        Tab,
        Escape,
        Copy,
        Cut,
        Paste,
        Undo,
        Redo,
        Bold,
        Italic,
        InlineCode
    ]
);

pub fn bind(cx: &mut App) {
    cx.bind_keys([
        KeyBinding::new("left", Left, Some("MarkGoWriter")),
        KeyBinding::new("right", Right, Some("MarkGoWriter")),
        KeyBinding::new("up", Up, Some("MarkGoWriter")),
        KeyBinding::new("down", Down, Some("MarkGoWriter")),
        KeyBinding::new("shift-left", SelectLeft, Some("MarkGoWriter")),
        KeyBinding::new("shift-right", SelectRight, Some("MarkGoWriter")),
        KeyBinding::new("shift-up", SelectUp, Some("MarkGoWriter")),
        KeyBinding::new("shift-down", SelectDown, Some("MarkGoWriter")),
        KeyBinding::new("home", Home, Some("MarkGoWriter")),
        KeyBinding::new("end", End, Some("MarkGoWriter")),
        KeyBinding::new("shift-home", SelectHome, Some("MarkGoWriter")),
        KeyBinding::new("shift-end", SelectEnd, Some("MarkGoWriter")),
        KeyBinding::new("backspace", Backspace, Some("MarkGoWriter")),
        KeyBinding::new("delete", Delete, Some("MarkGoWriter")),
        KeyBinding::new("enter", Enter, Some("MarkGoWriter")),
        KeyBinding::new("tab", Tab, Some("MarkGoWriter")),
        KeyBinding::new("escape", Escape, Some("MarkGoWriter")),
    ]);
    for modifier in ["cmd", "ctrl"] {
        cx.bind_keys([
            KeyBinding::new(
                if modifier == "cmd" { "cmd-a" } else { "ctrl-a" },
                SelectAll,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-c" } else { "ctrl-c" },
                Copy,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-x" } else { "ctrl-x" },
                Cut,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-v" } else { "ctrl-v" },
                Paste,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-z" } else { "ctrl-z" },
                Undo,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" {
                    "cmd-shift-z"
                } else {
                    "ctrl-shift-z"
                },
                Redo,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-b" } else { "ctrl-b" },
                Bold,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-i" } else { "ctrl-i" },
                Italic,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-e" } else { "ctrl-e" },
                InlineCode,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" {
                    "cmd-home"
                } else {
                    "ctrl-home"
                },
                DocumentHome,
                Some("MarkGoWriter"),
            ),
            KeyBinding::new(
                if modifier == "cmd" {
                    "cmd-end"
                } else {
                    "ctrl-end"
                },
                DocumentEnd,
                Some("MarkGoWriter"),
            ),
        ]);
    }
}

pub struct Changed;
impl EventEmitter<Changed> for Writer {}

pub struct Writer {
    pub editor: Editor,
    pub palette: Palette,
    pub prose_size: f32,
    pub column: f32,
    pub read_only: bool,
    focus: FocusHandle,
    scroll: ScrollHandle,
    layout: Vec<Row>,
    bounds: Bounds<Pixels>,
    dragging: bool,
    reveal: bool,
    slash: Option<usize>,
    slash_scroll: ScrollHandle,
    slash_resume: bool,
    blink_on: bool,
    last_typing: Option<(std::time::Instant, u64, usize)>,
    _blink: Task<()>,
}

#[derive(Clone)]
struct Row {
    range: Range<usize>,
    shape: Rc<WrappedLine>,
    origin: Point<Pixels>,
    height: Pixels,
    line_height: Pixels,
}

pub const BLOCKS: [(&str, &str, &str, &str); 8] = [
    ("paragraph", "普通段落 Paragraph", "", ""),
    ("format", "一级标题 Heading 1", "# ", ""),
    ("format", "二级标题 Heading 2", "## ", ""),
    ("format", "三级标题 Heading 3", "### ", ""),
    ("list", "无序列表 Bullet list", "- ", ""),
    ("numbered-list", "有序列表 Numbered list", "1. ", ""),
    ("quote", "引用 Quote", "> ", ""),
    ("code", "代码块 Code block", "```\n", "\n```"),
];

impl Writer {
    pub fn new(source: impl Into<String>, palette: Palette, cx: &mut Context<Self>) -> Self {
        let blink = cx.spawn(async move |this, cx| {
            loop {
                cx.background_executor()
                    .timer(std::time::Duration::from_millis(530))
                    .await;
                if this
                    .update(cx, |this, cx| {
                        this.blink_on = !this.blink_on;
                        cx.notify();
                    })
                    .is_err()
                {
                    break;
                }
            }
        });
        Self {
            editor: Editor::new(source),
            palette,
            prose_size: 18.,
            column: 720.,
            read_only: false,
            focus: cx.focus_handle(),
            scroll: ScrollHandle::new(),
            layout: vec![],
            bounds: Bounds::default(),
            dragging: false,
            reveal: false,
            slash: None,
            slash_scroll: ScrollHandle::new(),
            slash_resume: false,
            blink_on: true,
            last_typing: None,
            _blink: blink,
        }
    }
    pub fn focus(&self, window: &mut Window, cx: &mut App) {
        window.focus(&self.focus, cx);
    }
    pub fn cancel_composition(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        self.editor.cancel_composition();
        self.slash = None;
        self.slash_resume = false;
        self.changed(window, cx);
    }
    fn changed(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        self.blink_on = true;
        self.reveal = true;
        if self.editor.marked_range().is_some() {
            self.slash = None;
        }
        if self.slash.is_some() && self.slash_query().is_none() {
            self.slash = None;
        }
        window.notify_text_input_changed(&self.focus);
        cx.emit(Changed);
        cx.notify();
    }
    pub fn mode(
        &mut self,
        source: bool,
        read_only: bool,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        self.editor.set_source_mode(source);
        self.read_only = read_only;
        self.slash = None;
        self.changed(window, cx);
        self.focus(window, cx);
    }
    pub fn format(&mut self, marker: &str, window: &mut Window, cx: &mut Context<Self>) {
        if !self.read_only {
            self.editor.format_inline(marker);
            self.changed(window, cx);
        }
        self.focus(window, cx);
    }
    pub fn block(&mut self, index: usize, window: &mut Window, cx: &mut Context<Self>) {
        if !self.read_only {
            let block = BLOCKS[index];
            self.editor.block(block.2, block.3);
            self.slash = None;
            self.changed(window, cx);
        }
        self.focus(window, cx);
    }
    pub fn history(&mut self, redo: bool, window: &mut Window, cx: &mut Context<Self>) {
        if !self.read_only {
            if redo {
                self.editor.redo()
            } else {
                self.editor.undo()
            };
            self.slash = None;
            self.changed(window, cx);
        }
        self.focus(window, cx);
    }
    pub fn select_source(
        &mut self,
        range: Range<usize>,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        self.editor.cancel_composition();
        let _ = self.editor.document.select(Selection {
            anchor: range.start,
            head: range.end,
        });
        self.changed(window, cx);
    }
    fn slash_query(&self) -> Option<String> {
        let text = self.editor.text();
        let head = self.editor.selection().head;
        let start = text[..head].rfind('\n').map_or(0, |i| i + 1);
        text[start..head]
            .strip_prefix('/')
            .filter(|q| q.len() < 48 && !q.contains(' '))
            .map(str::to_lowercase)
    }
    fn choices(&self) -> Vec<usize> {
        let query = self.slash_query().unwrap_or_default();
        BLOCKS
            .iter()
            .enumerate()
            .filter(|(index, b)| {
                b.1.to_lowercase().contains(&query)
                    || (1..=3).contains(index) && query == format!("h{index}")
            })
            .map(|(i, _)| i)
            .collect()
    }
    fn move_horizontal(
        &mut self,
        forward: bool,
        extend: bool,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        self.editor.horizontal(forward, extend);
        self.slash = None;
        self.changed(window, cx);
    }
    fn move_vertical(
        &mut self,
        down: bool,
        extend: bool,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        if let Some(selected) = self.slash {
            let count = self.choices().len();
            if count > 0 {
                self.slash = Some(if down {
                    (selected + 1) % count
                } else {
                    (selected + count - 1) % count
                });
                self.slash_scroll.scroll_to_item(self.slash.unwrap());
                cx.notify();
            }
            return;
        }
        self.editor.cancel_composition();
        let selection = self.editor.selection();
        if let Some(bounds) = self.caret(selection.head) {
            let target = point(
                bounds.left(),
                bounds.top()
                    + if down {
                        bounds.size.height * 1.05
                    } else {
                        -bounds.size.height * 0.55
                    },
            );
            let at = self.hit(target);
            let _ = self.editor.select(Selection {
                anchor: if extend { selection.anchor } else { at },
                head: at,
            });
        }
        self.changed(window, cx);
    }
    fn line_edge(
        &mut self,
        end: bool,
        extend: bool,
        whole: bool,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        self.editor.cancel_composition();
        let text = self.editor.text();
        let s = self.editor.selection();
        let at = if whole {
            if end { text.len() } else { 0 }
        } else if end {
            text[s.head..].find('\n').map_or(text.len(), |i| s.head + i)
        } else {
            text[..s.head].rfind('\n').map_or(0, |i| i + 1)
        };
        let _ = self.editor.select_native(Selection {
            anchor: if extend { s.anchor } else { at },
            head: at,
        });
        self.changed(window, cx);
    }
    fn hit(&self, point: Point<Pixels>) -> usize {
        let Some(row) = self
            .layout
            .iter()
            .find(|r| point.y < r.origin.y + r.height)
            .or(self.layout.last())
        else {
            return 0;
        };
        let local = point - row.origin;
        let index = row
            .shape
            .closest_index_for_position(
                point_fn(local.x, local.y.max(px(0.)).min(row.height - px(1.))),
                row.line_height,
            )
            .unwrap_or_else(|i| i);
        (row.range.start + index).min(row.range.end)
    }
    fn caret(&self, index: usize) -> Option<Bounds<Pixels>> {
        let row = self
            .layout
            .iter()
            .find(|r| r.range.contains(&index) || r.range.end == index)
            .or(self.layout.last())?;
        let position = row
            .shape
            .position_for_index(
                index.saturating_sub(row.range.start).min(row.shape.len()),
                row.line_height,
            )
            .unwrap_or_default();
        Some(Bounds::new(
            row.origin + position,
            size(px(1.5), row.line_height),
        ))
    }
    fn mouse_down(&mut self, event: &MouseDownEvent, window: &mut Window, cx: &mut Context<Self>) {
        let index = self
            .editor
            .committed_display_offset(self.hit(event.position), false);
        self.editor.cancel_composition();
        self.slash = None;
        self.focus(window, cx);
        self.dragging = true;
        let selection = self.editor.selection();
        let mut next = Selection {
            anchor: if event.modifiers.shift {
                selection.anchor
            } else {
                index
            },
            head: index,
        };
        if event.click_count == 2 {
            use unicode_segmentation::UnicodeSegmentation;
            let text = self.editor.text();
            if let Some((start, word)) = text
                .split_word_bound_indices()
                .find(|(start, word)| *start <= index && index < start + word.len())
            {
                next = Selection {
                    anchor: start,
                    head: start + word.len(),
                };
            }
        } else if event.click_count >= 3 {
            let text = self.editor.text();
            let start = text[..index].rfind('\n').map_or(0, |i| i + 1);
            let end = text[index..].find('\n').map_or(text.len(), |i| index + i);
            next = Selection {
                anchor: start,
                head: end,
            };
        }
        let _ = self.editor.select(next);
        self.changed(window, cx);
    }
    fn mouse_move(&mut self, event: &MouseMoveEvent, window: &mut Window, cx: &mut Context<Self>) {
        if self.dragging {
            let at = self.hit(event.position);
            let _ = self.editor.select(Selection {
                anchor: self.editor.selection().anchor,
                head: at,
            });
            self.changed(window, cx);
        }
    }
    fn copy(&mut self, cut: bool, window: &mut Window, cx: &mut Context<Self>) {
        self.editor.cancel_composition();
        let selection = self.editor.document.selection();
        if selection.range().is_empty() {
            return;
        }
        cx.write_to_clipboard(ClipboardItem::new_string(
            self.editor.document.source()[selection.range()].to_owned(),
        ));
        if cut && !self.read_only {
            let _ = self.editor.insert("");
            self.changed(window, cx);
        }
    }
    fn enter(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if self.read_only {
            return;
        }
        if let Some(selected) = self.slash {
            if let Some(&index) = self.choices().get(selected) {
                self.block(index, window, cx);
            }
            return;
        }
        let _ = self.editor.enter();
        self.changed(window, cx);
    }
}
fn point_fn(x: Pixels, y: Pixels) -> Point<Pixels> {
    point(x, y)
}
impl Focusable for Writer {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl EntityInputHandler for Writer {
    fn text_for_range(
        &mut self,
        range: Range<usize>,
        actual: &mut Option<Range<usize>>,
        _: &mut Window,
        _: &mut Context<Self>,
    ) -> Option<String> {
        let range = self.editor.byte_range(range);
        *actual = Some(self.editor.utf16_range(range.clone()));
        Some(self.editor.text()[range].to_owned())
    }
    fn selected_text_range(
        &mut self,
        _: bool,
        _: &mut Window,
        _: &mut Context<Self>,
    ) -> Option<UTF16Selection> {
        let s = self.editor.selection();
        Some(UTF16Selection {
            range: self.editor.utf16_range(s.range()),
            reversed: s.anchor > s.head,
        })
    }
    fn set_selected_text_range(
        &mut self,
        range: Range<usize>,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        let range = self.editor.byte_range(range);
        let _ = self.editor.select_native(Selection {
            anchor: range.start,
            head: range.end,
        });
        self.changed(window, cx);
    }
    fn text_length_utf16(&mut self, _: &mut Window, _: &mut Context<Self>) -> Option<usize> {
        Some(self.editor.text().encode_utf16().count())
    }
    fn marked_text_range(&self, _: &mut Window, _: &mut Context<Self>) -> Option<Range<usize>> {
        self.editor
            .marked_range()
            .map(|r| self.editor.utf16_range(r))
    }
    fn unmark_text(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let _ = self.editor.commit_composition(None);
        if self.slash_resume {
            self.slash = Some(0);
            self.slash_resume = false;
        }
        self.changed(window, cx);
    }
    fn replace_text_in_range(
        &mut self,
        range: Option<Range<usize>>,
        text: &str,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        if self.read_only {
            return;
        }
        let now = std::time::Instant::now();
        let before = self.editor.document.selection();
        let revision = self.editor.document.revision();
        let typing = range.is_none()
            && self.editor.marked_range().is_none()
            && !text.is_empty()
            && !text.contains(['\r', '\n']);
        let group = typing
            && before.range().is_empty()
            && self.last_typing.is_some_and(|(time, version, head)| {
                now.duration_since(time).as_millis() < 750
                    && version == revision
                    && head == before.head
            });
        let trigger = text == "/" && self.editor.can_open_slash();
        let range = range.map(|r| self.editor.byte_range(r));
        let _ = self.editor.replace_native(range, text);
        if group {
            self.editor.document.coalesce_typing();
        }
        self.last_typing = typing.then_some((
            now,
            self.editor.document.revision(),
            self.editor.document.selection().head,
        ));
        if trigger || self.slash_resume || self.slash.is_some() {
            self.slash = Some(0)
        }
        self.slash_resume = false;
        if self.slash.is_some() {
            self.slash_scroll.scroll_to_item(0);
        }
        self.changed(window, cx);
    }
    fn replace_and_mark_text_in_range(
        &mut self,
        range: Option<Range<usize>>,
        text: &str,
        selected: Option<Range<usize>>,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) {
        if self.read_only {
            return;
        }
        let range = range.map(|r| self.editor.byte_range(r));
        let selection =
            selected.map(|r| utf16_to_byte(text, r.start, false)..utf16_to_byte(text, r.end, true));
        let _ = self.editor.mark(range, text, selection);
        self.slash_resume |= self.slash.is_some();
        self.slash = None;
        self.changed(window, cx);
    }
    fn bounds_for_range(
        &mut self,
        range: Range<usize>,
        _: Bounds<Pixels>,
        _: &mut Window,
        _: &mut Context<Self>,
    ) -> Option<Bounds<Pixels>> {
        let range = self.editor.byte_range(range);
        self.caret(range.start)
    }
    fn selection_bounds_for_range(
        &mut self,
        range: Range<usize>,
        _: Bounds<Pixels>,
        _: &mut Window,
        _: &mut Context<Self>,
    ) -> Vec<Bounds<Pixels>> {
        selection_rects(&self.layout, self.editor.byte_range(range))
    }
    fn character_index_for_point(
        &mut self,
        point: Point<Pixels>,
        _: &mut Window,
        _: &mut Context<Self>,
    ) -> Option<usize> {
        let at = self.hit(point);
        Some(self.editor.utf16_range(at..at).start)
    }
}

impl Render for Writer {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.palette;
        let mut root = div()
            .id("writer")
            .role(Role::MultilineTextInput)
            .aria_label("Markdown 文档")
            .key_context("MarkGoWriter")
            .track_focus(&self.focus)
            .relative()
            .size_full()
            .min_h_0()
            .on_action(cx.listener(|s, _: &Left, w, c| s.move_horizontal(false, false, w, c)))
            .on_action(cx.listener(|s, _: &Right, w, c| s.move_horizontal(true, false, w, c)))
            .on_action(cx.listener(|s, _: &SelectLeft, w, c| s.move_horizontal(false, true, w, c)))
            .on_action(cx.listener(|s, _: &SelectRight, w, c| s.move_horizontal(true, true, w, c)))
            .on_action(cx.listener(|s, _: &Up, w, c| s.move_vertical(false, false, w, c)))
            .on_action(cx.listener(|s, _: &Down, w, c| s.move_vertical(true, false, w, c)))
            .on_action(cx.listener(|s, _: &SelectUp, w, c| s.move_vertical(false, true, w, c)))
            .on_action(cx.listener(|s, _: &SelectDown, w, c| s.move_vertical(true, true, w, c)))
            .on_action(cx.listener(|s, _: &Home, w, c| s.line_edge(false, false, false, w, c)))
            .on_action(cx.listener(|s, _: &End, w, c| s.line_edge(true, false, false, w, c)))
            .on_action(cx.listener(|s, _: &SelectHome, w, c| s.line_edge(false, true, false, w, c)))
            .on_action(cx.listener(|s, _: &SelectEnd, w, c| s.line_edge(true, true, false, w, c)))
            .on_action(
                cx.listener(|s, _: &DocumentHome, w, c| s.line_edge(false, false, true, w, c)),
            )
            .on_action(cx.listener(|s, _: &DocumentEnd, w, c| s.line_edge(true, false, true, w, c)))
            .on_action(cx.listener(|s, _: &SelectAll, w, c| {
                s.editor.cancel_composition();
                let len = s.editor.text().len();
                let _ = s.editor.select(Selection {
                    anchor: 0,
                    head: len,
                });
                s.changed(w, c);
            }))
            .on_action(cx.listener(|s, _: &Backspace, w, c| {
                if !s.read_only {
                    s.editor.delete(false);
                    s.changed(w, c);
                }
            }))
            .on_action(cx.listener(|s, _: &Delete, w, c| {
                if !s.read_only {
                    s.editor.delete(true);
                    s.changed(w, c);
                }
            }))
            .on_action(cx.listener(|s, _: &Enter, w, c| s.enter(w, c)))
            .on_action(cx.listener(|s, _: &Tab, w, c| {
                if s.slash.is_some() {
                    s.enter(w, c)
                } else if !s.read_only {
                    let _ = s.editor.insert("    ");
                    s.changed(w, c);
                }
            }))
            .on_action(cx.listener(|s, _: &Escape, w, c| {
                s.slash = None;
                s.editor.cancel_composition();
                s.changed(w, c);
            }))
            .on_action(cx.listener(|s, _: &Copy, w, c| s.copy(false, w, c)))
            .on_action(cx.listener(|s, _: &Cut, w, c| s.copy(true, w, c)))
            .on_action(cx.listener(|s, _: &Paste, w, c| {
                if !s.read_only
                    && let Some(text) = c.read_from_clipboard().and_then(|i| i.text())
                {
                    s.slash = None;
                    let _ = s.editor.insert(&text);
                    s.changed(w, c);
                }
            }))
            .on_action(cx.listener(|s, _: &Undo, w, c| s.history(false, w, c)))
            .on_action(cx.listener(|s, _: &Redo, w, c| s.history(true, w, c)))
            .on_action(cx.listener(|s, _: &Bold, w, c| s.format("**", w, c)))
            .on_action(cx.listener(|s, _: &Italic, w, c| s.format("*", w, c)))
            .on_action(cx.listener(|s, _: &InlineCode, w, c| s.format("`", w, c)));
        root = root.child(
            div()
                .id("writer-scroll")
                .size_full()
                .overflow_y_scroll()
                .track_scroll(&self.scroll)
                .child(
                    div()
                        .w_full()
                        .max_w(px(self.column + 96.))
                        .mx_auto()
                        .px(px(48.))
                        .pt(px(32.))
                        .pb(px(160.))
                        .cursor_text()
                        .on_mouse_down(MouseButton::Left, cx.listener(Self::mouse_down))
                        .on_mouse_move(cx.listener(Self::mouse_move))
                        .on_mouse_up(
                            MouseButton::Left,
                            cx.listener(|s, _, _, _| s.dragging = false),
                        )
                        .on_mouse_up_out(
                            MouseButton::Left,
                            cx.listener(|s, _, _, _| s.dragging = false),
                        )
                        .child(Surface {
                            writer: cx.entity(),
                        }),
                ),
        );
        if let Some(selected) = self.slash {
            let caret = self
                .caret(self.editor.selection().head)
                .unwrap_or(self.bounds);
            let viewport = self.scroll.bounds();
            let choices = self.choices();
            let full_height = 50. + 40. * choices.len().max(1) as f32;
            let above = (caret.top() - viewport.top() - px(8.)).max(px(0.));
            let below_start = caret.bottom() - viewport.top() + px(8.);
            let below_space = (viewport.size.height - below_start - px(8.)).max(px(0.));
            let flip = below_space < px(full_height) && above > below_space;
            let available = if flip { above } else { below_space };
            let rows = (((f32::from(available) - 50.) / 40.).floor() as usize)
                .max(1)
                .min(choices.len().max(1));
            let height = 50. + 40. * rows as f32;
            let left = (caret.left() - viewport.left())
                .max(px(12.))
                .min((viewport.size.width - px(336.)).max(px(12.)));
            let top = if flip {
                caret.top() - viewport.top() - px(height + 8.)
            } else {
                below_start
            };
            let mut menu = div()
                .id("slash-menu")
                .absolute()
                .left(left)
                .top(top.max(px(8.)))
                .w(px(320.))
                .p(px(4.))
                .bg(rgb(p.paper))
                .border_1()
                .border_color(rgb(p.edge))
                .rounded(px(12.))
                .shadow_md()
                .on_mouse_down(MouseButton::Left, |_, _, cx| cx.stop_propagation())
                .child(
                    div()
                        .h(px(40.))
                        .px(px(12.))
                        .flex()
                        .items_center()
                        .text_size(px(12.))
                        .text_color(rgb(p.muted))
                        .child("插入格式"),
                );
            let mut items = div()
                .id("slash-choices")
                .h(px(40. * rows as f32))
                .overflow_y_scroll()
                .track_scroll(&self.slash_scroll);
            if choices.is_empty() {
                items = items.child(div().h(px(40.)).px(px(12.)).child("没有匹配的格式"));
            }
            for (position, index) in choices.into_iter().enumerate() {
                let b = BLOCKS[index];
                items = items.child(
                    Button::new(("slash", index))
                        .ghost()
                        .h(px(40.))
                        .w_full()
                        .justify_start()
                        .row(b.0, b.1.split_whitespace().next().unwrap())
                        .when(position == selected, |b| b.bg(rgb(p.hover)))
                        .on_click(cx.listener(move |s, _, w, c| s.block(index, w, c))),
                );
            }
            menu = menu.child(items);
            root = root.child(menu);
        }
        root
    }
}

struct Surface {
    writer: Entity<Writer>,
}
impl IntoElement for Surface {
    type Element = Self;
    fn into_element(self) -> Self {
        self
    }
}
impl Element for Surface {
    type RequestLayoutState = Rc<RefCell<Vec<Row>>>;
    type PrepaintState = Vec<Row>;
    fn id(&self) -> Option<ElementId> {
        Some("writing-surface".into())
    }
    fn source_location(&self) -> Option<&'static core::panic::Location<'static>> {
        None
    }
    fn request_layout(
        &mut self,
        _: Option<&GlobalElementId>,
        _: Option<&InspectorElementId>,
        window: &mut Window,
        _cx: &mut App,
    ) -> (LayoutId, Self::RequestLayoutState) {
        let rows = Rc::new(RefCell::new(vec![]));
        let output = rows.clone();
        let writer = self.writer.clone();
        let mut style = Style::default();
        style.size.width = relative(1.).into();
        let id = window.request_measured_layout(style, move |known, available, window, cx| {
            let width = known
                .width
                .unwrap_or_else(|| match available.width {
                    AvailableSpace::Definite(w) => w,
                    _ => px(720.),
                })
                .max(px(80.));
            let view = writer.read(cx);
            let text = view.editor.text();
            let mut top = px(0.);
            let mut start = 0;
            let mut shaped = vec![];
            for line in text.split('\n') {
                let heading = view.editor.heading_at(start);
                let font_size = px(view.prose_size
                    * match heading {
                        1 => 2.,
                        2 => 1.5,
                        3 => 1.25,
                        _ => 1.,
                    });
                let line_height = if line.is_empty() && !text.is_empty() {
                    px(view.prose_size * 0.9)
                } else if heading > 0 {
                    font_size * 1.4
                } else {
                    font_size * 1.75
                };
                let mut font = font(if view.editor.source_mode() {
                    cx.theme().mono_font_family.clone()
                } else {
                    cx.theme().font_family.clone()
                });
                if heading > 0 {
                    font.weight = FontWeight::SEMIBOLD;
                }
                let shown: SharedString = if text.is_empty() {
                    "写点什么，或输入 / 插入格式".into()
                } else if line.is_empty() {
                    " ".into()
                } else {
                    line.to_owned().into()
                };
                let run = TextRun {
                    len: shown.len(),
                    font,
                    color: rgb(if text.is_empty() {
                        view.palette.muted
                    } else {
                        view.palette.ink
                    })
                    .into(),
                    background_color: None,
                    underline: None,
                    strikethrough: None,
                };
                if let Ok(lines) =
                    window
                        .text_system()
                        .shape_text(shown, font_size, &[run], Some(width), None)
                {
                    for shape in lines {
                        let height = shape.size(line_height).height;
                        shaped.push(Row {
                            range: start..start + line.len(),
                            shape: Rc::new(shape),
                            origin: point(px(0.), top),
                            height,
                            line_height,
                        });
                        top += height;
                    }
                }
                start += line.len() + 1;
            }
            *output.borrow_mut() = shaped;
            size(width, top.max(px(40.)))
        });
        (id, rows)
    }
    fn prepaint(
        &mut self,
        _: Option<&GlobalElementId>,
        _: Option<&InspectorElementId>,
        bounds: Bounds<Pixels>,
        state: &mut Self::RequestLayoutState,
        window: &mut Window,
        cx: &mut App,
    ) -> Vec<Row> {
        let mut rows = state.borrow().clone();
        for row in &mut rows {
            row.origin += bounds.origin;
        }
        self.writer.update(cx, |writer, _| {
            writer.layout = rows.clone();
            writer.bounds = bounds;
            if writer.reveal {
                if let Some(caret) = writer.caret(writer.editor.selection().head) {
                    window.request_autoscroll(caret);
                }
                writer.reveal = false;
            }
        });
        rows
    }
    fn paint(
        &mut self,
        _: Option<&GlobalElementId>,
        _: Option<&InspectorElementId>,
        bounds: Bounds<Pixels>,
        _: &mut Self::RequestLayoutState,
        rows: &mut Vec<Row>,
        window: &mut Window,
        cx: &mut App,
    ) {
        let writer = self.writer.read(cx);
        let focus = writer.focus.clone();
        let selection = writer.editor.selection();
        let p = writer.palette;
        let caret = writer.caret(selection.head);
        let marked = writer.editor.marked_range();
        let caret_visible = writer.blink_on && !writer.read_only && focus.is_focused(window);
        if !writer.read_only {
            window.handle_input(
                &focus,
                ElementInputHandler::new(bounds, self.writer.clone()),
                cx,
            );
        }
        for rect in selection_rects(rows, selection.range()) {
            window.paint_quad(fill(
                rect,
                Hsla {
                    a: 0.2,
                    ..rgb(p.accent).into()
                },
            ));
        }
        for row in rows.iter() {
            let _ = row.shape.paint(
                row.origin,
                row.line_height,
                TextAlign::Left,
                None,
                window,
                cx,
            );
        }
        if let Some(marked) = marked {
            for rect in selection_rects(rows, marked) {
                window.paint_quad(fill(
                    Bounds::new(
                        point(rect.left(), rect.bottom() - px(2.)),
                        size(rect.size.width, px(1.)),
                    ),
                    rgb(p.accent),
                ));
            }
        }
        if selection.range().is_empty()
            && caret_visible
            && let Some(caret) = caret
        {
            window.paint_quad(fill(caret, rgb(p.accent)));
        }
    }
}

fn selection_rects(rows: &[Row], range: Range<usize>) -> Vec<Bounds<Pixels>> {
    if range.is_empty() {
        return vec![];
    }
    let mut rects = vec![];
    for row in rows {
        let start = range.start.max(row.range.start);
        let end = range.end.min(row.range.end);
        if start > end || range.end <= row.range.start || range.start > row.range.end {
            continue;
        }
        let a = row
            .shape
            .position_for_index(start - row.range.start, row.line_height)
            .unwrap_or_default();
        let b = row
            .shape
            .position_for_index(end - row.range.start, row.line_height)
            .unwrap_or_default();
        let mut y = a.y;
        while y <= b.y {
            let x = if y == a.y { a.x } else { px(0.) };
            let right = if y == b.y { b.x } else { row.shape.width() };
            rects.push(Bounds::new(
                row.origin + point(x, y),
                size((right - x).max(px(2.)), row.line_height),
            ));
            y += row.line_height;
        }
    }
    rects
}
