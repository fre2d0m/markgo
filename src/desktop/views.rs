use super::workbench::Panel;
use super::*;
use assets::icon;
use gpui_kit::base::FocusTrapElement;
use gpui_kit::base::Selectable;
use gpui_kit::base::motion::{self, Presence, Transition};
use gpui_kit::{
    component::{
        ActiveTheme, Disableable, Sizable, TitleBar,
        button::{Button, ButtonCustomVariant, ButtonVariants},
        input::{Input, InputGroup},
    },
    prelude::*,
};
use std::time::Duration;
use theme::{PALETTES, Palette};

fn menu_row(
    id: impl Into<ElementId>,
    name: &str,
    label: &str,
    shortcut: &str,
    p: Palette,
) -> Button {
    Button::new(id)
        .ghost()
        .h(px(40.))
        .w_full()
        .px(px(12.))
        .rounded(px(8.))
        .accessibility_label(label.to_owned())
        .child(
            div()
                .w_full()
                .min_w_0()
                .flex()
                .items_center()
                .gap(px(12.))
                .text_size(px(13.))
                .font_weight(FontWeight::NORMAL)
                .child(
                    div()
                        .size(px(18.))
                        .flex_shrink_0()
                        .child(icon(name).with_size(px(18.))),
                )
                .child(
                    div()
                        .flex_1()
                        .min_w_0()
                        .text_ellipsis()
                        .child(label.to_owned()),
                )
                .child(
                    div()
                        .text_size(px(11.))
                        .text_color(rgb(p.muted))
                        .child(shortcut.to_owned()),
                ),
        )
}

fn separator(p: Palette) -> impl IntoElement {
    div()
        .h(px(1.))
        .flex_shrink_0()
        .my(px(6.))
        .mx(px(12.))
        .bg(rgb(p.edge))
}

pub(super) trait RowLabel {
    fn row(self, name: &str, label: impl Into<SharedString>) -> Self;
}
impl RowLabel for Button {
    fn row(self, name: &str, label: impl Into<SharedString>) -> Self {
        let label = label.into();
        self.accessibility_label(label.clone())
            .font_weight(FontWeight::NORMAL)
            .child(
                div()
                    .w_full()
                    .min_w_0()
                    .flex()
                    .items_center()
                    .gap(px(8.))
                    .when(!name.is_empty(), |d| {
                        d.child(icon(name).with_size(px(16.)).flex_shrink_0())
                    })
                    .child(div().flex_1().min_w_0().text_ellipsis().child(label)),
            )
    }
}

fn button(
    id: impl Into<ElementId>,
    name: &str,
    label: &str,
    p: Palette,
    _: &App,
) -> super::controls::IconButton {
    super::controls::IconButton::new(id, name, label, p)
}

impl Workbench {
    fn titlebar(&self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        let brand_width = if cfg!(target_os = "macos") {
            144.
        } else {
            212.
        };
        let tabs =
            self.tabs
                .iter()
                .enumerate()
                .map(|(index, tab)| {
                    let active = index == self.active;
                    let dirty = tab.dirty(cx);
                    let name = tab.name.clone();
                    let tooltip = tab
                        .path
                        .as_ref()
                        .map_or_else(|| name.clone(), |path| crate::storage::display_path(path));
                    let id = tab.id;
                    let background = motion::transition(
                        ElementId::from(("tab-color", id)),
                        Hsla::from(rgb(if active {
                            p.paper
                        } else if self.hovered_tab == Some(id) {
                            p.hover
                        } else {
                            p.rail
                        })),
                        Transition::new(Duration::from_millis(100)),
                        window,
                        cx,
                    );
                    div()
                        .id(("tab", tab.id))
                        // The toolkit title bar labels its entire inner region as
                        // draggable. Occlude that native hitbox beneath tabs, and
                        // keep button presses from starting its mouse-drag fallback.
                        .occlude()
                        .on_mouse_down(MouseButton::Left, |_, _, c| c.stop_propagation())
                        .h(px(48.))
                        .relative()
                        .flex_shrink_0()
                        .flex()
                        .items_center()
                        .bg(background)
                        .on_hover(cx.listener(move |s, hovered: &bool, _, c| {
                            if *hovered {
                                s.hovered_tab = Some(id);
                            } else if s.hovered_tab == Some(id) {
                                s.hovered_tab = None;
                            }
                            c.notify();
                        }))
                        .border_b_1()
                        .border_color(rgb(if active { p.paper } else { p.edge }))
                        .child(div().absolute().top_0().left_0().right_0().h(px(2.)).bg(
                            if active {
                                Hsla::from(rgb(p.accent))
                            } else {
                                transparent_black()
                            },
                        ))
                        .on_mouse_down(
                            MouseButton::Right,
                            cx.listener(move |s, _, w, c| s.show_panel(Panel::Document(id), w, c)),
                        )
                        .child(
                            Button::new(("activate", tab.id))
                                .role(Role::Tab)
                                .selected(active)
                                .custom(
                                    ButtonCustomVariant::new(cx)
                                        .color(transparent_black())
                                        .foreground(
                                            rgb(if active { p.ink } else { p.muted }).into(),
                                        )
                                        .hover(transparent_black())
                                        .active(transparent_black()),
                                )
                                .h(px(48.))
                                .w(px(164.))
                                .rounded(px(0.))
                                .justify_start()
                                .row("file", name.clone())
                                .when(active, |b| b.font_weight(FontWeight::MEDIUM))
                                .tooltip(tooltip)
                                .on_click(cx.listener(move |s, _, w, c| s.activate(index, w, c))),
                        )
                        .child(
                            button(
                                ("close", tab.id),
                                if dirty { "dirty" } else { "close" },
                                if dirty {
                                    "关闭未保存文档"
                                } else {
                                    "关闭文档"
                                },
                                p,
                                cx,
                            )
                            .on_click(cx.listener(move |s, _, w, c| s.request_close(index, w, c))),
                        )
                })
                .collect::<Vec<_>>();
        TitleBar::new()
            .h(px(48.))
            .bg(rgb(p.rail))
            .border_color(rgb(p.edge))
            .child(
                div()
                    .flex()
                    .items_center()
                    .w((window.viewport_size().width
                        - px(if cfg!(target_os = "macos") { 80. } else { 117. }))
                    .max(px(200.)))
                    .min_w_0()
                    .when(self.sidebar && !self.focus_mode, |d| {
                        d.child(
                            div()
                                .w(px(brand_width))
                                .flex_shrink_0()
                                .flex()
                                .items_center()
                                .gap(px(8.))
                                .pl(px(12.))
                                .text_size(px(15.))
                                .font_weight(FontWeight::SEMIBOLD)
                                .child(icon("mark").with_size(px(22.)))
                                .child("MarkGo"),
                        )
                    })
                    .child(
                        div()
                            .id("tabs")
                            .role(Role::TabList)
                            .flex()
                            .items_center()
                            .flex_1()
                            .min_w_0()
                            .overflow_x_scroll()
                            .track_scroll(&self.tab_scroll)
                            .children(tabs),
                    )
                    .child(
                        div()
                            .id("titlebar-actions")
                            .occlude()
                            .on_mouse_down(MouseButton::Left, |_, _, c| c.stop_propagation())
                            .flex()
                            .items_center()
                            .flex_shrink_0()
                            .child(
                                button("new-tab", "plus", "新建文档", p, cx)
                                    .on_click(cx.listener(|s, _, w, c| s.new_document(w, c))),
                            )
                            .child(button("all-tabs", "chevron", "所有文档", p, cx).on_click(
                                cx.listener(|s, _, w, c| s.show_panel(Panel::Tabs, w, c)),
                            ))
                            .child(button("appearance", "appearance", "外观", p, cx).on_click(
                                cx.listener(|s, _, w, c| s.show_panel(Panel::Appearance, w, c)),
                            ))
                            .child(button("focus", "focus", "专注模式", p, cx).on_click(
                                cx.listener(|s, _, _, c| {
                                    s.focus_mode = !s.focus_mode;
                                    c.notify();
                                }),
                            )),
                    ),
            )
    }
    fn sidebar_view(&self, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        let mut entries = div()
            .id("navigation-list")
            .flex_1()
            .min_h_0()
            .overflow_y_scroll()
            .track_scroll(&self.sidebar_scroll)
            .px(px(12.));
        if self.outline {
            let writer = self.writer();
            let projection = writer.read(cx).editor.projection();
            for line in projection.lines().iter().filter(|l| l.heading > 0) {
                let range = line.content.clone();
                let text = writer.read(cx).editor.document.source()[range.clone()].to_owned();
                let level = line.heading;
                entries = entries.child(
                    Button::new(("heading", line.source_start))
                        .ghost()
                        .h(px(40.))
                        .w_full()
                        .justify_start()
                        .pl(px(12. + f32::from(level.saturating_sub(1)) * 12.))
                        .row("", text)
                        .on_click(cx.listener(move |s, _, w, c| {
                            s.writer().update(c, |writer, c| {
                                writer.select_source(range.start..range.start, w, c)
                            });
                            s.focus_writer(w, c);
                        })),
                );
            }
            if !projection.lines().iter().any(|l| l.heading > 0) {
                entries = entries.child(
                    div()
                        .px(px(12.))
                        .py(px(24.))
                        .text_color(rgb(p.muted))
                        .child("标题会显示在这里"),
                );
            }
        } else {
            entries = entries.child(
                div()
                    .h(px(40.))
                    .px(px(8.))
                    .flex()
                    .items_center()
                    .text_size(px(12.))
                    .text_color(rgb(p.muted))
                    .child(format!("打开的文档  ·  {}", self.tabs.len())),
            );
            for (index, tab) in self.tabs.iter().enumerate() {
                entries = entries.child(
                    Button::new(("file", tab.id))
                        .ghost()
                        .h(px(40.))
                        .w_full()
                        .justify_start()
                        .row("file", tab.name.clone())
                        .tooltip(tab.file.as_ref().map_or_else(
                            || tab.name.clone(),
                            |file| crate::storage::display_path(&file.path),
                        ))
                        .when(index == self.active, |b| b.bg(rgb(p.hover)))
                        .on_click(cx.listener(move |s, _, w, c| s.activate(index, w, c))),
                );
            }
            let name = self
                .folder
                .as_ref()
                .and_then(|p| p.file_name())
                .map(|s| s.to_string_lossy().into_owned())
                .unwrap_or_else(|| "文件夹".into());
            entries = entries.child(
                div()
                    .mt(px(20.))
                    .h(px(40.))
                    .px(px(8.))
                    .flex()
                    .items_center()
                    .text_color(rgb(p.muted))
                    .text_size(px(12.))
                    .child(div().flex_1().min_w_0().text_ellipsis().child(name))
                    .child(
                        button("refresh-folder", "refresh", "刷新文件夹", p, cx)
                            .disabled(self.loading || self.folder.is_none())
                            .on_click(cx.listener(|s, _, w, c| {
                                if let Some(path) = s.folder.clone() {
                                    s.load_folder(path, w, c);
                                }
                            })),
                    ),
            );
            if self.loading {
                entries = entries.child(div().px(px(8.)).child("正在读取…"));
            } else if self.files.is_empty() {
                entries = entries.child(
                    div()
                        .px(px(8.))
                        .text_size(px(12.))
                        .text_color(rgb(p.muted))
                        .child(if self.folder.is_some() {
                            "没有 Markdown 文档"
                        } else {
                            "打开文件夹，让文档在手边"
                        }),
                );
            }
            for (index, file) in self.files.iter().enumerate() {
                let query = self.folder_filter.read(cx).value().to_lowercase();
                if !query.is_empty() && !file.path.to_string_lossy().to_lowercase().contains(&query)
                {
                    continue;
                }
                if query.is_empty()
                    && self
                        .collapsed
                        .iter()
                        .any(|p| p != &file.path && file.path.starts_with(p))
                {
                    continue;
                }
                let path = file.path.clone();
                let directory = file.directory;
                entries = entries.child(
                    Button::new(("folder-file", index))
                        .ghost()
                        .h(px(40.))
                        .w_full()
                        .justify_start()
                        .pl(px(8. + file.depth as f32 * 8.))
                        .row(if directory { "folder" } else { "file" }, file.name.clone())
                        .tooltip(crate::storage::display_path(&file.path))
                        .on_click(cx.listener(move |s, _, w, c| {
                            if directory {
                                if !s.collapsed.remove(&path) {
                                    s.collapsed.insert(path.clone());
                                }
                                c.notify();
                            } else {
                                s.open_path(path.clone(), w, c);
                            }
                        })),
                );
            }
        }
        div()
            .w(px(224.))
            .h_full()
            .flex_shrink_0()
            .flex()
            .flex_col()
            .bg(rgb(p.rail))
            .child(
                div()
                    .h(px(56.))
                    .px(px(12.))
                    .flex()
                    .items_center()
                    .gap(px(4.))
                    .child(
                        Button::new("files-view")
                            .ghost()
                            .h(px(40.))
                            .label("文件")
                            .when(!self.outline, |b| b.bg(rgb(p.hover)))
                            .on_click(cx.listener(|s, _, _, c| {
                                s.outline = false;
                                c.notify();
                            })),
                    )
                    .child(
                        Button::new("outline-view")
                            .ghost()
                            .h(px(40.))
                            .label("大纲")
                            .when(self.outline, |b| b.bg(rgb(p.hover)))
                            .on_click(cx.listener(|s, _, _, c| {
                                s.outline = true;
                                c.notify();
                            })),
                    )
                    .child(div().flex_1())
                    .child(
                        button("open-file", "file", "打开文档", p, cx)
                            .on_click(cx.listener(|s, _, w, c| s.prompt_open(false, w, c))),
                    ),
            )
            .when(!self.outline, |d| {
                d.child(
                    div()
                        .px(px(12.))
                        .pb(px(8.))
                        .child(Input::new(&self.folder_filter).h(px(40.))),
                )
            })
            .child(entries)
            .child(
                div().p(px(12.)).child(
                    Button::new("open-folder")
                        .ghost()
                        .h(px(40.))
                        .w_full()
                        .justify_start()
                        .row("folder", "打开文件夹")
                        .on_click(cx.listener(|s, _, w, c| s.prompt_open(true, w, c))),
                ),
            )
    }
    fn toolbar(&self, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        let readonly = self.writer().read(cx).read_only;
        div()
            .h(px(48.))
            .flex_shrink_0()
            .mx(px(20.))
            .flex()
            .items_center()
            .gap(px(4.))
            .border_b_1()
            .border_color(rgb(p.edge))
            .child(
                button("sidebar", "sidebar", "切换侧栏", p, cx).on_click(cx.listener(
                    |s, _, _, c| {
                        s.sidebar = !s.sidebar;
                        c.notify();
                    },
                )),
            )
            .child(div().w(px(1.)).h(px(20.)).mx(px(8.)).bg(rgb(p.edge)))
            .child(
                button("bold", "bold", "加粗", p, cx)
                    .disabled(readonly)
                    .on_click(cx.listener(|s, _, w, c| {
                        s.writer().update(c, |writer, c| writer.format("**", w, c))
                    })),
            )
            .child(
                button("italic", "italic", "斜体", p, cx)
                    .disabled(readonly)
                    .on_click(cx.listener(|s, _, w, c| {
                        s.writer().update(c, |writer, c| writer.format("*", w, c))
                    })),
            )
            .child(
                button("inline-code", "code", "行内代码", p, cx)
                    .disabled(readonly)
                    .on_click(cx.listener(|s, _, w, c| {
                        s.writer().update(c, |writer, c| writer.format("`", w, c))
                    })),
            )
            .child(
                button("insert-format", "plus", "插入格式", p, cx)
                    .disabled(readonly)
                    .on_click(cx.listener(|s, _, w, c| s.show_panel(Panel::Format, w, c))),
            )
            .child(
                button("edit-link", "link", "编辑链接", p, cx)
                    .disabled(readonly)
                    .on_click(cx.listener(|s, _, w, c| s.open_link(w, c))),
            )
            .child(div().flex_1())
            .child(
                button("search", "search", "查找与替换", p, cx)
                    .on_click(cx.listener(|s, _, w, c| s.open_search(w, c))),
            )
            .child(
                button("save", "save", "保存文档", p, cx)
                    .on_click(cx.listener(|s, _, w, c| s.save(false, None, w, c))),
            )
            .child(
                button("view-menu", "more", "视图与文件选项", p, cx)
                    .on_click(cx.listener(|s, _, w, c| s.show_panel(Panel::View, w, c))),
            )
    }
    fn search_view(&self, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        div()
            .flex_shrink_0()
            .px(px(20.))
            .py(px(8.))
            .flex()
            .flex_col()
            .gap(px(8.))
            .border_b_1()
            .border_color(rgb(p.edge))
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap(px(4.))
                    .child(div().flex_1().child(Input::new(&self.search).h(px(40.))))
                    .child(
                        div()
                            .w(px(72.))
                            .text_center()
                            .text_size(px(12.))
                            .text_color(rgb(p.muted))
                            .child(if self.matches.is_empty() {
                                "无匹配".into()
                            } else {
                                format!("{} / {}", self.match_index + 1, self.matches.len())
                            }),
                    )
                    .child(
                        button("previous-match", "previous", "上一个", p, cx)
                            .disabled(self.matches.is_empty())
                            .on_click(cx.listener(|s, _, w, c| s.next_match(false, w, c))),
                    )
                    .child(
                        button("next-match", "next", "下一个", p, cx)
                            .disabled(self.matches.is_empty())
                            .on_click(cx.listener(|s, _, w, c| s.next_match(true, w, c))),
                    )
                    .child(
                        Button::new("toggle-replace")
                            .ghost()
                            .h(px(40.))
                            .label("替换")
                            .on_click(cx.listener(|s, _, _, c| {
                                s.replace_open = !s.replace_open;
                                c.notify();
                            })),
                    )
                    .child(
                        button("close-search", "close", "关闭查找", p, cx)
                            .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                    ),
            )
            .child(
                div().flex().gap(px(4.)).children(
                    [
                        (0, "Aa", self.search_options.case),
                        (1, "全词", self.search_options.words),
                        (2, ".*", self.search_options.regex),
                        (3, "选区内", self.search_options.scoped),
                    ]
                    .into_iter()
                    .map(|(index, label, active)| {
                        Button::new(("search-option", index))
                            .ghost()
                            .h(px(40.))
                            .label(label)
                            .when(active, |b| b.bg(rgb(p.hover)))
                            .on_click(
                                cx.listener(move |s, _, w, c| s.toggle_search_option(index, w, c)),
                            )
                    }),
                ),
            )
            .when_some(self.search_error.clone(), |d, error| {
                d.child(
                    div()
                        .text_size(px(12.))
                        .text_color(rgb(p.accent))
                        .child(error),
                )
            })
            .when(self.replace_open, |d| {
                d.child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .child(div().flex_1().child(Input::new(&self.replace).h(px(40.))))
                        .child(
                            Button::new("replace-one")
                                .ghost()
                                .h(px(40.))
                                .label("替换当前")
                                .disabled(self.matches.is_empty())
                                .on_click(cx.listener(|s, _, w, c| s.replace_match(false, w, c))),
                        )
                        .child(
                            Button::new("replace-all")
                                .ghost()
                                .h(px(40.))
                                .label("全部替换")
                                .disabled(self.matches.is_empty())
                                .on_click(cx.listener(|s, _, w, c| s.replace_match(true, w, c))),
                        ),
                )
            })
    }
    fn footer(&self, cx: &App) -> impl IntoElement {
        let p = self.p();
        let tab = &self.tabs[self.active];
        let writer = tab.writer.read(cx);
        let source = writer.editor.document.source();
        let head = writer.editor.document.selection().head;
        let status = self
            .session_error
            .clone()
            .or(self.notice.clone())
            .unwrap_or_else(|| {
                if tab.saving {
                    "正在保存…".into()
                } else if tab.conflict {
                    "原文件已变更或不可用 · 请另存副本".into()
                } else if tab.dirty(cx) {
                    if tab.conflict {
                        "原文件已变更 · 请另存副本".into()
                    } else if tab.recovered {
                        "恢复的草稿 · 请检查后保存".into()
                    } else if self.session_saved == self.session_generation && !self.ephemeral {
                        "草稿已备份 · 文档尚未保存".into()
                    } else {
                        "有未保存的修改".into()
                    }
                } else if self.ephemeral {
                    "示例文档 · 仅在内存中".into()
                } else if tab.file.is_some() {
                    "已保存到本地".into()
                } else {
                    "未保存的文档".into()
                }
            });
        let line = source[..head].bytes().filter(|&b| b == b'\n').count() + 1;
        div()
            .h(px(28.))
            .flex_shrink_0()
            .px(px(16.))
            .flex()
            .items_center()
            .gap(px(16.))
            .bg(rgb(p.rail))
            .text_size(px(12.))
            .text_color(rgb(p.muted))
            .child(status)
            .child(div().flex_1())
            .child(if writer.read_only {
                "阅读"
            } else if writer.editor.source_mode() {
                "Markdown 源码"
            } else {
                "Markdown"
            })
            .child(
                div()
                    .w(px(120.))
                    .text_right()
                    .child(format!("{} 字符", source.chars().count())),
            )
            .child(div().w(px(72.)).text_right().child(format!("第 {line} 行")))
    }
    fn panel_view(
        &self,
        panel: Panel,
        progress: f32,
        interactive: bool,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) -> AnyElement {
        let p = self.p();
        let utility = matches!(
            &panel,
            Panel::View | Panel::Format | Panel::Tabs | Panel::Recent | Panel::Document(_)
        );
        let width = match &panel {
            Panel::Tabs => 360.,
            Panel::Recent => 400.,
            Panel::View | Panel::Format | Panel::Document(_) => 304.,
            _ => 440.,
        };
        let viewport = window.viewport_size();
        let anchor = match &panel {
            Panel::Tabs => point(
                viewport.width
                    - px(if cfg!(target_os = "macos") { 0. } else { 117. })
                    - px(80. + width),
                px(56.),
            ),
            Panel::Format => point(
                px(if self.sidebar && !self.focus_mode {
                    441.
                } else {
                    217.
                }),
                px(100.),
            ),
            Panel::Document(_) => self.panel_anchor,
            _ => point(viewport.width - px(width + 28.), px(100.)),
        };
        let anchor = point(
            anchor.x.clamp(px(8.), viewport.width - px(width + 8.)),
            anchor.y.clamp(px(56.), viewport.height - px(248.)),
        );
        let max_height = if utility {
            viewport.height - anchor.y - px(36.)
        } else {
            viewport.height - px(64.)
        };
        let title = match &panel {
            Panel::Appearance => "外观",
            Panel::View => "视图与文件",
            Panel::Format => "插入格式",
            Panel::Close { .. } => "保留这次修改？",
            Panel::Error(_) => "操作未完成",
            Panel::Exit => "退出前，保留未完成的文字",
            Panel::Recent => "最近打开",
            Panel::Tabs => "打开的文档",
            Panel::Document(_) => "文档操作",
            Panel::Reload { .. } => "重新载入文件",
            Panel::Link { .. } => "编辑链接",
            Panel::Code { .. } => "代码语言",
        };
        let mut content = div()
            .id("panel-body")
            .flex()
            .flex_col()
            .gap(px(if utility { 2. } else { 16. }))
            .p(px(if utility { 8. } else { 24. }))
            .w(px(width))
            .max_h(max_height)
            .overflow_y_scroll()
            .bg(rgb(p.paper))
            .border_1()
            .border_color(rgb(p.edge))
            .rounded(px(16.))
            .shadow_md()
            .opacity(progress)
            .relative()
            .top(px((1. - progress) * if interactive { -4. } else { -2. }))
            .on_mouse_down(MouseButton::Left, |_, _, c| c.stop_propagation())
            .child(
                div()
                    .h(px(40.))
                    .flex_shrink_0()
                    .when(utility, |d| d.pl(px(12.)))
                    .flex()
                    .items_center()
                    .justify_between()
                    .child(
                        div()
                            .text_size(px(if utility { 13. } else { 16. }))
                            .text_color(rgb(if utility { p.muted } else { p.ink }))
                            .font_weight(FontWeight::SEMIBOLD)
                            .child(title),
                    )
                    .child(
                        button("close-panel", "close", "关闭", p, cx)
                            .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                    ),
            );
        match panel {
            Panel::Appearance => {
                content = content.child(
                    div()
                        .text_color(rgb(p.muted))
                        .text_size(px(13.))
                        .child("为此刻的写作，选一张合适的纸。"),
                );
                let mut grid = div().flex().flex_wrap().gap(px(8.));
                for (index, theme) in PALETTES.iter().enumerate() {
                    let selected = self.palette == index;
                    grid = grid.child(
                        Button::new(("theme", index))
                            .h(px(48.))
                            .w(px(124.))
                            .label(theme.name)
                            .icon(
                                icon(if selected { "check" } else { "appearance" })
                                    .with_size(px(16.)),
                            )
                            .custom(
                                ButtonCustomVariant::new(cx)
                                    .color(rgb(theme.paper).into())
                                    .foreground(rgb(theme.ink).into())
                                    .hover(rgb(theme.hover).into())
                                    .active(rgb(theme.pressed).into()),
                            )
                            .border_1()
                            .bg(rgb(if self.hovered_theme == Some(index) {
                                theme.hover
                            } else {
                                theme.paper
                            }))
                            .text_color(rgb(theme.ink))
                            .on_hover(cx.listener(move |s, hovered, _, c| {
                                s.hovered_theme = if *hovered { Some(index) } else { None };
                                c.notify();
                            }))
                            .border_color(rgb(if selected { theme.accent } else { theme.edge }))
                            .on_click(cx.listener(move |s, _, w, c| s.set_palette(index, w, c))),
                    );
                }
                content = content
                    .child(grid)
                    .child(
                        div()
                            .flex()
                            .gap(px(8.))
                            .child(
                                Button::new("system-theme")
                                    .ghost()
                                    .h(px(40.))
                                    .label("跟随系统")
                                    .icon(
                                        icon(if self.follow_system {
                                            "check"
                                        } else {
                                            "appearance"
                                        })
                                        .with_size(px(18.)),
                                    )
                                    .on_click(cx.listener(|s, _, w, c| s.toggle_system(w, c))),
                            )
                            .child(
                                Button::new("serif-font")
                                    .ghost()
                                    .h(px(40.))
                                    .label(if self.serif {
                                        "正文：宋体"
                                    } else {
                                        "正文：系统字体"
                                    })
                                    .on_click(cx.listener(|s, _, _, c| s.toggle_serif(c))),
                            ),
                    )
                    .child(div().h(px(1.)).bg(rgb(p.edge)))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .child("正文字号")
                            .child(div().flex_1())
                            .child(
                                button("smaller", "minus", "减小字号", p, cx)
                                    .disabled(self.prose <= 14.)
                                    .on_click(cx.listener(|s, _, _, c| s.resize_prose(-1., c))),
                            )
                            .child(
                                div()
                                    .w(px(28.))
                                    .text_center()
                                    .child(format!("{}", self.prose)),
                            )
                            .child(
                                button("larger", "plus", "增大字号", p, cx)
                                    .disabled(self.prose >= 24.)
                                    .on_click(cx.listener(|s, _, _, c| s.resize_prose(1., c))),
                            ),
                    )
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .child("阅读宽度")
                            .child(div().flex_1())
                            .children([640., 720., 840.].into_iter().map(|width| {
                                Button::new(("width", width as usize))
                                    .ghost()
                                    .h(px(40.))
                                    .label(if width == 640. {
                                        "紧凑"
                                    } else if width == 720. {
                                        "适中"
                                    } else {
                                        "宽松"
                                    })
                                    .when(width == self.column, |b| b.bg(rgb(p.hover)))
                                    .on_click(cx.listener(move |s, _, _, c| s.set_column(width, c)))
                            })),
                    );
            }
            Panel::View => {
                let writer = self.writer();
                let writer = writer.read(cx);
                let source = writer.editor.source_mode();
                let reading = writer.read_only;
                for (index, label, checked) in [
                    (0, "直接编辑", !source && !reading),
                    (1, "Markdown 源码", source),
                    (2, "阅读", reading),
                ] {
                    content = content.child(
                        menu_row(
                            ("mode", index as usize),
                            if checked {
                                "check"
                            } else if index == 2 {
                                "read"
                            } else {
                                "code"
                            },
                            label,
                            if index == 1 {
                                if cfg!(target_os = "macos") {
                                    "⇧⌘M"
                                } else {
                                    "Ctrl+Shift+M"
                                }
                            } else {
                                ""
                            },
                            p,
                        )
                        .when(checked, |b| b.bg(rgb(p.hover)))
                        .on_click(
                            cx.listener(move |s, _, w, c| s.set_mode(index == 1, index == 2, w, c)),
                        ),
                    );
                }
                content = content
                    .child(separator(p))
                    .child(
                        menu_row(
                            "menu-open",
                            "folder",
                            "打开文档…",
                            if cfg!(target_os = "macos") {
                                "⌘O"
                            } else {
                                "Ctrl+O"
                            },
                            p,
                        )
                        .on_click(cx.listener(|s, _, w, c| s.prompt_open(false, w, c))),
                    )
                    .child(
                        menu_row(
                            "menu-save-as",
                            "save",
                            "另存为…",
                            if cfg!(target_os = "macos") {
                                "⇧⌘S"
                            } else {
                                "Ctrl+Shift+S"
                            },
                            p,
                        )
                        .on_click(cx.listener(|s, _, w, c| s.save(true, None, w, c))),
                    )
                    .child(
                        menu_row("menu-recent", "recovery", "最近打开…", "", p)
                            .on_click(cx.listener(|s, _, w, c| s.show_panel(Panel::Recent, w, c))),
                    );
            }
            Panel::Tabs => {
                for (index, tab) in self.tabs.iter().enumerate() {
                    let detail = tab
                        .path
                        .as_ref()
                        .map(|p| crate::storage::display_path(p))
                        .unwrap_or_else(|| "尚未保存".into());
                    content = content.child(
                        Button::new(("all-tab", tab.id))
                            .ghost()
                            .selected(index == self.active)
                            .h(px(56.))
                            .w_full()
                            .px(px(12.))
                            .when(index == self.active, |b| b.bg(rgb(p.hover)))
                            .accessibility_label(format!("{}，{}", tab.name, detail))
                            .tooltip(detail.clone())
                            .justify_start()
                            .child(
                                div()
                                    .w_full()
                                    .flex()
                                    .items_center()
                                    .gap(px(12.))
                                    .min_w_0()
                                    .child(icon("file").with_size(px(18.)).flex_shrink_0())
                                    .child(
                                        div()
                                            .flex_1()
                                            .min_w_0()
                                            .flex()
                                            .flex_col()
                                            .gap(px(4.))
                                            .child(
                                                div()
                                                    .text_size(px(13.))
                                                    .text_ellipsis()
                                                    .child(tab.name.clone()),
                                            )
                                            .child(
                                                div()
                                                    .text_size(px(11.))
                                                    .text_color(rgb(p.muted))
                                                    .text_ellipsis()
                                                    .child(detail),
                                            ),
                                    )
                                    .child(
                                        div()
                                            .size(px(18.))
                                            .flex_shrink_0()
                                            .when(index == self.active, |d| {
                                                d.child(icon("check").with_size(px(18.)))
                                            }),
                                    ),
                            )
                            .on_click(cx.listener(move |s, _, w, c| {
                                s.panel = None;
                                s.activate(index, w, c);
                            })),
                    );
                }
            }
            Panel::Document(id) => {
                if let Some(index) = self.tabs.iter().position(|t| t.id == id) {
                    let tab = &self.tabs[index];
                    content = content.child(
                        div()
                            .id("context-document-name")
                            .w_full()
                            .min_w_0()
                            .px(px(12.))
                            .py(px(4.))
                            .text_size(px(12.))
                            .text_color(rgb(p.muted))
                            .text_ellipsis()
                            .aria_label(tab.name.clone())
                            .child(tab.name.clone()),
                    );
                    if let Some(path) = tab.path.clone() {
                        content = content.child(
                            menu_row("copy-path", "link", "复制完整路径", "", p).on_click(
                                cx.listener(move |s, _, w, c| {
                                    c.write_to_clipboard(ClipboardItem::new_string(
                                        path.to_string_lossy().into_owned(),
                                    ));
                                    s.dismiss(w, c);
                                    s.notice = Some("已复制文件路径".into());
                                }),
                            ),
                        );
                    }
                    content = content
                        .child(
                            menu_row("context-save", "save", "保存此文档", "", p).on_click(
                                cx.listener(move |s, _, w, c| {
                                    s.activate(index, w, c);
                                    s.panel = None;
                                    s.save(false, None, w, c);
                                }),
                            ),
                        )
                        .child(
                            menu_row("reload-file", "refresh", "从磁盘重新载入…", "", p)
                                .disabled(tab.path.is_none() || tab.saving)
                                .on_click(
                                    cx.listener(move |s, _, w, c| s.request_reload(id, w, c)),
                                ),
                        )
                        .child(
                            menu_row("context-close", "close", "关闭此文档", "", p).on_click(
                                cx.listener(move |s, _, w, c| s.request_close(index, w, c)),
                            ),
                        );
                }
            }
            Panel::Reload { tab, revision } => {
                content = content
                    .child("当前修改会替换为磁盘上的内容。重新载入后仍可撤销，恢复修改。")
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap(px(8.))
                            .child(
                                Button::new("cancel-reload")
                                    .h(px(40.))
                                    .label("继续编辑")
                                    .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                            )
                            .child(
                                Button::new("confirm-reload")
                                    .primary()
                                    .h(px(40.))
                                    .label("重新载入")
                                    .on_click(cx.listener(move |s, _, w, c| {
                                        s.reload_document(tab, revision, w, c)
                                    })),
                            ),
                    );
            }
            Panel::Format => {
                for (index, b) in writer::BLOCKS.iter().enumerate() {
                    content = content.child(
                        menu_row(
                            ("format", index),
                            b.0,
                            b.1.split_whitespace().next().unwrap(),
                            "",
                            p,
                        )
                        .on_click(cx.listener(move |s, _, w, c| {
                            s.panel = None;
                            s.writer().update(c, |writer, c| writer.block(index, w, c));
                            c.notify();
                        })),
                    );
                }
            }
            Panel::Close { index, window: all } => {
                content = content
                    .child(div().text_size(px(14.)).child(format!(
                        "“{}”有尚未保存的修改。关闭后，这些修改将丢失。",
                        self.tabs.get(index).map_or("文档", |t| t.name.as_str())
                    )))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap(px(8.))
                            .pt(px(8.))
                            .child(
                                Button::new("discard")
                                    .ghost()
                                    .h(px(40.))
                                    .label("不保存")
                                    .on_click(cx.listener(move |s, _, w, c| {
                                        s.discard_close(index, all, w, c)
                                    })),
                            )
                            .child(div().flex_1())
                            .child(
                                Button::new("cancel-close")
                                    .h(px(40.))
                                    .label("继续编辑")
                                    .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                            )
                            .child(
                                Button::new("save-close")
                                    .primary()
                                    .h(px(40.))
                                    .label("保存")
                                    .on_click(cx.listener(move |s, _, w, c| {
                                        s.save(false, Some((index, all)), w, c)
                                    })),
                            ),
                    );
            }
            Panel::Error(message) => {
                content = content
                    .child(div().text_size(px(14.)).child(message))
                    .when(
                        self.session_error.is_some() && self.journal.is_some(),
                        |d| {
                            d.child(
                                Button::new("retry-backup")
                                    .h(px(40.))
                                    .label("重试草稿备份")
                                    .on_click(cx.listener(|s, _, w, c| s.retry_checkpoint(w, c))),
                            )
                        },
                    )
                    .child(
                        Button::new("error-return")
                            .primary()
                            .h(px(40.))
                            .label("返回编辑")
                            .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                    );
            }
            Panel::Exit => {
                content = content.child(div().text_size(px(14.)).child(if self.ephemeral {
                    "预览中的示例仅保存在内存。关闭后将丢弃本次修改。"
                } else {
                    "保留草稿会保存本次写作进度，下次打开时恢复；不会改动原始 Markdown 文件。"
                }));
                content = content.child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.))
                        .mt(px(24.))
                        .child(
                            Button::new("discard-exit")
                                .ghost()
                                .label("不保留")
                                .h(px(40.))
                                .on_click(cx.listener(|s, _, w, c| s.discard_and_quit(w, c))),
                        )
                        .child(div().flex_1())
                        .child(
                            Button::new("cancel-exit")
                                .ghost()
                                .label("取消")
                                .h(px(40.))
                                .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                        )
                        .when(!self.ephemeral, |d| {
                            d.child(
                                Button::new("save-exit")
                                    .ghost()
                                    .label("保存全部")
                                    .h(px(40.))
                                    .on_click(cx.listener(|s, _, w, c| s.save_next_and_quit(w, c))),
                            )
                        })
                        .when(!self.ephemeral, |d| {
                            d.child(
                                Button::new("keep-exit")
                                    .primary()
                                    .label("保留草稿")
                                    .h(px(40.))
                                    .on_click(cx.listener(|s, _, w, c| s.keep_and_quit(w, c))),
                            )
                        }),
                );
            }
            Panel::Recent => {
                content = content.child(
                    Input::new(&self.recent_filter)
                        .h(px(40.))
                        .aria_label("搜索最近文档"),
                );
                let query = self.recent_filter.read(cx).value().to_lowercase();
                let mut count = 0;
                let mut rows = div()
                    .id("recent-list")
                    .max_h(px(320.))
                    .overflow_y_scroll()
                    .flex()
                    .flex_col()
                    .gap(px(2.));
                for (index, path) in self.recent.iter().enumerate() {
                    if !path.to_lowercase().contains(&query) {
                        continue;
                    }
                    count += 1;
                    let path = std::path::PathBuf::from(path);
                    let open = path.clone();
                    rows = rows.child(
                        div()
                            .flex()
                            .flex_shrink_0()
                            .items_center()
                            .child(
                                menu_row(
                                    ("recent", index),
                                    "file",
                                    &path.file_name().unwrap_or_default().to_string_lossy(),
                                    "",
                                    p,
                                )
                                .flex_1()
                                .min_w_0()
                                .tooltip(crate::storage::display_path(&path))
                                .on_click(cx.listener(
                                    move |s, _, w, c| {
                                        s.panel = None;
                                        s.open_path(open.clone(), w, c);
                                    },
                                )),
                            )
                            .child(
                                button(("forget-recent", index), "close", "移除这条记录", p, cx)
                                    .on_click(cx.listener(move |s, _, _, c| {
                                        if index < s.recent.len() {
                                            s.recent.remove(index);
                                            s.session_changed();
                                            c.notify();
                                        }
                                    })),
                            ),
                    );
                }
                if self.recent.is_empty() {
                    rows = rows.child(div().py(px(24.)).child("打开过的文档会出现在这里"));
                } else if count == 0 {
                    rows = rows.child(div().py(px(24.)).child("没有匹配的最近文档"));
                }
                content = content.child(rows).child(separator(p)).child(
                    menu_row("clear-recent", "recovery", "清除记录", "", p)
                        .disabled(self.recent.is_empty())
                        .on_click(cx.listener(|s, _, _, c| {
                            s.recent.clear();
                            s.session_changed();
                            c.notify();
                        })),
                );
            }
            Panel::Link { existing, .. } => {
                content = content
                    .child(div().text_size(px(12.)).child("显示文字"))
                    .child(
                        InputGroup::new("link-label-field")
                            .h(px(40.))
                            .invalid(self.entry_error.is_some() && self.entry_label_invalid)
                            .input(Input::new(&self.entry_label).aria_label("显示文字")),
                    )
                    .child(div().text_size(px(12.)).child("链接地址"))
                    .child(
                        InputGroup::new("link-address-field")
                            .h(px(40.))
                            .invalid(self.entry_error.is_some() && !self.entry_label_invalid)
                            .input(Input::new(&self.entry_value).aria_label("链接地址")),
                    )
                    .when_some(self.entry_error.clone(), |d, error| {
                        d.child(div().text_color(rgb(p.accent)).child(error))
                    })
                    .child(
                        div()
                            .flex()
                            .gap(px(8.))
                            .when(existing, |d| {
                                d.child(
                                    Button::new("remove-link")
                                        .ghost()
                                        .label("移除链接")
                                        .h(px(40.))
                                        .on_click(
                                            cx.listener(|s, _, w, c| s.apply_entry(true, w, c)),
                                        ),
                                )
                            })
                            .child(div().flex_1())
                            .child(
                                Button::new("cancel-entry")
                                    .ghost()
                                    .label("取消")
                                    .h(px(40.))
                                    .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                            )
                            .child(
                                Button::new("apply-entry")
                                    .primary()
                                    .label("应用")
                                    .h(px(40.))
                                    .on_click(cx.listener(|s, _, w, c| s.apply_entry(false, w, c))),
                            ),
                    );
            }
            Panel::Code { .. } => {
                content = content
                    .child(
                        div()
                            .text_size(px(13.))
                            .child("例如 rust、python 或 javascript；留空表示纯文本。"),
                    )
                    .child(
                        InputGroup::new("code-language-field")
                            .h(px(40.))
                            .invalid(self.entry_error.is_some())
                            .input(Input::new(&self.entry_value).aria_label("代码语言")),
                    )
                    .when_some(self.entry_error.clone(), |d, error| {
                        d.child(div().text_color(rgb(p.accent)).child(error))
                    })
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap(px(8.))
                            .child(
                                Button::new("cancel-entry")
                                    .ghost()
                                    .h(px(40.))
                                    .label("取消")
                                    .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                            )
                            .child(
                                Button::new("apply-entry")
                                    .primary()
                                    .label("应用")
                                    .h(px(40.))
                                    .on_click(cx.listener(|s, _, w, c| s.apply_entry(false, w, c))),
                            ),
                    );
            }
        }
        let overlay = div()
            .id("modal")
            .role(if utility { Role::Menu } else { Role::Dialog })
            .aria_label(title)
            .track_focus(&self.panel_focus)
            .when(interactive, |d| d.occlude())
            .absolute()
            .inset_0()
            .flex()
            .items_center()
            .justify_center()
            .when(!utility, |d| {
                d.bg(Hsla {
                    a: 0.18 * progress,
                    ..black()
                })
            })
            .when(interactive, |d| {
                d.on_mouse_down(
                    MouseButton::Left,
                    cx.listener(move |s, _, w, c| {
                        c.stop_propagation();
                        if utility {
                            s.dismiss(w, c);
                        }
                    }),
                )
            })
            .child(
                content
                    .when(!interactive, |d| {
                        d.child(
                            div()
                                .absolute()
                                .inset_0()
                                .occlude()
                                .on_mouse_down(MouseButton::Left, |_, _, c| c.stop_propagation()),
                        )
                    })
                    .when(utility, |d| {
                        d.absolute().left(anchor.x).top(
                            anchor.y + px((1. - progress) * if interactive { -4. } else { -2. }),
                        )
                    }),
            );
        if interactive {
            overlay
                .focus_trap("workbench-dialog", &self.panel_focus)
                .into_any_element()
        } else {
            overlay.into_any_element()
        }
    }
}
impl Render for Workbench {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        if std::mem::take(&mut self.panel_focus_pending) && self.panel.is_some() {
            window.focus(&self.panel_focus, cx);
        }
        let p = self.p();
        let writer = self.writer();
        let has_panel = self.panel.is_some();
        if has_panel {
            self.presented_panel = self.panel.clone();
        }
        let presence = Presence::new("workbench-panel", has_panel)
            .transition(Transition::new(Duration::from_millis(if has_panel {
                130
            } else {
                90
            })))
            .sample(window, cx);
        // Build the overlay before assembling the workbench. Passing the large
        // root div through when_some while building a dialog multiplies debug
        // stack frames on Windows.
        let panel = if presence.should_render() {
            self.presented_panel
                .clone()
                .map(|p| self.panel_view(p, presence.progress, has_panel, window, cx))
        } else {
            self.presented_panel = None;
            None
        };
        div()
            .id("markgo")
            .key_context(if has_panel { "MarkGoModal" } else { "MarkGo" })
            .track_focus(&self.root_focus)
            .relative()
            .size_full()
            .flex()
            .flex_col()
            .bg(rgb(p.rail))
            .text_color(rgb(p.ink))
            .text_size(px(13.))
            .font_family(cx.theme().font_family.clone())
            .on_drop::<ExternalPaths>(cx.listener(|s, paths: &ExternalPaths, w, c| {
                s.open_external(paths.paths().to_vec(), w, c)
            }))
            .on_action(cx.listener(|s, _: &NewDocument, w, c| s.new_document(w, c)))
            .on_action(cx.listener(|s, _: &OpenDocument, w, c| s.prompt_open(false, w, c)))
            .on_action(cx.listener(|s, _: &OpenFolder, w, c| s.prompt_open(true, w, c)))
            .on_action(cx.listener(|s, _: &SaveDocument, w, c| s.save(false, None, w, c)))
            .on_action(cx.listener(|s, _: &SaveAsDocument, w, c| s.save(true, None, w, c)))
            .on_action(cx.listener(|s, _: &CloseDocument, w, c| s.request_close(s.active, w, c)))
            .on_action(cx.listener(|s, _: &NextDocument, w, c| {
                s.activate((s.active + 1) % s.tabs.len(), w, c)
            }))
            .on_action(cx.listener(|s, _: &PreviousDocument, w, c| {
                s.activate((s.active + s.tabs.len() - 1) % s.tabs.len(), w, c)
            }))
            .on_action(cx.listener(|s, _: &Search, w, c| s.open_search(w, c)))
            .on_action(cx.listener(|s, _: &Appearance, w, c| s.show_panel(Panel::Appearance, w, c)))
            .on_action(cx.listener(|s, _: &EditLink, w, c| s.open_link(w, c)))
            .on_action(cx.listener(|s, _: &ToggleSource, w, c| {
                let source = !s.writer().read(c).editor.source_mode();
                s.set_mode(source, false, w, c);
            }))
            .on_action(cx.listener(|s, _: &ToggleSidebar, _, c| {
                s.sidebar = !s.sidebar;
                c.notify();
            }))
            .on_action(cx.listener(|s, _: &Dismiss, w, c| s.dismiss(w, c)))
            .on_action(cx.listener(|s, _: &Quit, w, c| {
                if s.request_window_close(w, c) {
                    w.remove_window();
                }
            }))
            .child(self.titlebar(window, cx))
            .child(
                div()
                    .flex_1()
                    .min_h_0()
                    .flex()
                    .when(self.sidebar && !self.focus_mode, |d| {
                        d.child(self.sidebar_view(cx))
                    })
                    .child(
                        div()
                            .flex_1()
                            .min_w_0()
                            .h_full()
                            .mr(px(8.))
                            .mb(px(8.))
                            .flex()
                            .flex_col()
                            .bg(rgb(p.paper))
                            .child(self.toolbar(cx))
                            .when(self.search_open, |d| d.child(self.search_view(cx)))
                            .child(div().flex_1().min_h_0().child(writer)),
                    ),
            )
            .child(self.footer(cx))
            .children(panel)
    }
}
