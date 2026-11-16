use super::workbench::Panel;
use super::*;
use assets::icon;
use gpui_kit::base::FocusTrapElement;
use gpui_kit::{
    component::{
        Disableable, Sizable, TitleBar,
        button::{Button, ButtonCustomVariant, ButtonVariants},
        input::Input,
    },
    prelude::*,
};
use theme::{PALETTES, Palette};

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

fn button(id: impl Into<ElementId>, name: &str, label: &str, p: Palette, cx: &App) -> Button {
    Button::new(id)
        .icon(icon(name).with_size(px(18.)))
        .accessibility_label(label.to_owned())
        .tooltip(label.to_owned())
        .h(px(40.))
        .w(px(40.))
        .rounded(px(8.))
        .custom(
            ButtonCustomVariant::new(cx)
                .color(transparent_black())
                .foreground(rgb(p.ink).into())
                .hover(rgb(p.hover).into())
                .active(rgb(p.pressed).into()),
        )
}

impl Workbench {
    fn titlebar(&self, window: &Window, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        let brand_width = if cfg!(target_os = "macos") {
            144.
        } else {
            212.
        };
        let tabs = self
            .tabs
            .iter()
            .enumerate()
            .map(|(index, tab)| {
                let active = index == self.active;
                let dirty = tab.dirty(cx);
                let name = tab.name.clone();
                let tooltip = tab.file.as_ref().map_or_else(
                    || name.clone(),
                    |file| file.path.to_string_lossy().into_owned(),
                );
                div()
                    .id(("tab", tab.id))
                    .h(px(48.))
                    .flex_shrink_0()
                    .flex()
                    .items_center()
                    .bg(rgb(if active { p.paper } else { p.rail }))
                    .border_b_1()
                    .border_color(rgb(if active { p.paper } else { p.edge }))
                    .child(
                        Button::new(("activate", tab.id))
                            .ghost()
                            .h(px(48.))
                            .w(px(164.))
                            .rounded(px(0.))
                            .justify_start()
                            .row("file", name.clone())
                            .tooltip(tooltip)
                            .on_click(cx.listener(move |s, _, w, c| s.activate(index, w, c))),
                    )
                    .child(
                        button(
                            ("close", tab.id),
                            if dirty { "minus" } else { "close" },
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
                            .flex()
                            .items_center()
                            .flex_1()
                            .min_w_0()
                            .overflow_x_scroll()
                            .track_scroll(&self.tab_scroll)
                            .children(tabs),
                    )
                    .child(
                        button("new-tab", "plus", "新建文档", p, cx)
                            .on_click(cx.listener(|s, _, w, c| s.new_document(w, c))),
                    )
                    .child(
                        button("appearance", "appearance", "外观", p, cx).on_click(
                            cx.listener(|s, _, w, c| s.show_panel(Panel::Appearance, w, c)),
                        ),
                    )
                    .child(
                        button("focus", "focus", "专注模式", p, cx).on_click(cx.listener(
                            |s, _, _, c| {
                                s.focus_mode = !s.focus_mode;
                                c.notify();
                            },
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
                            |file| file.path.to_string_lossy().into_owned(),
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
                    .child(name),
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
                let path = file.path.clone();
                entries = entries.child(
                    Button::new(("folder-file", index))
                        .ghost()
                        .h(px(40.))
                        .w_full()
                        .justify_start()
                        .pl(px(8. + file.depth as f32 * 8.))
                        .row("file", file.name.clone())
                        .tooltip(file.path.to_string_lossy().into_owned())
                        .on_click(cx.listener(move |s, _, w, c| s.open_path(path.clone(), w, c))),
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
        let status = self.notice.clone().unwrap_or_else(|| {
            if tab.saving {
                "正在保存…".into()
            } else if tab.dirty(cx) {
                "有未保存的修改".into()
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
    fn panel_view(&self, panel: Panel, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        let title = match &panel {
            Panel::Appearance => "外观",
            Panel::View => "视图与文件",
            Panel::Format => "插入格式",
            Panel::Close { .. } => "保留这次修改？",
            Panel::Error(_) => "操作未完成",
        };
        let mut content = div()
            .id("panel-body")
            .flex()
            .flex_col()
            .gap(px(16.))
            .p(px(24.))
            .w(px(440.))
            .max_h(px(568.))
            .overflow_y_scroll()
            .bg(rgb(p.paper))
            .border_1()
            .border_color(rgb(p.edge))
            .rounded(px(16.))
            .shadow_lg()
            .on_mouse_down(MouseButton::Left, |_, _, c| c.stop_propagation())
            .child(
                div()
                    .flex()
                    .items_center()
                    .justify_between()
                    .child(
                        div()
                            .text_size(px(16.))
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
                        Button::new(("mode", index as usize))
                            .ghost()
                            .h(px(40.))
                            .justify_start()
                            .label(label)
                            .icon(icon(if checked { "check" } else { "file" }).with_size(px(18.)))
                            .on_click(cx.listener(move |s, _, w, c| {
                                s.set_mode(index == 1, index == 2, w, c)
                            })),
                    );
                }
                content = content
                    .child(div().h(px(1.)).bg(rgb(p.edge)))
                    .child(
                        Button::new("menu-open")
                            .ghost()
                            .h(px(40.))
                            .justify_start()
                            .icon(icon("file").with_size(px(18.)))
                            .label("打开文档…")
                            .on_click(cx.listener(|s, _, w, c| s.prompt_open(false, w, c))),
                    )
                    .child(
                        Button::new("menu-save-as")
                            .ghost()
                            .h(px(40.))
                            .justify_start()
                            .icon(icon("save").with_size(px(18.)))
                            .label("另存为…")
                            .on_click(cx.listener(|s, _, w, c| s.save(true, None, w, c))),
                    );
            }
            Panel::Format => {
                for (index, b) in writer::BLOCKS.iter().enumerate() {
                    content = content.child(
                        Button::new(("format", index))
                            .ghost()
                            .h(px(40.))
                            .justify_start()
                            .row(b.0, b.1.split_whitespace().next().unwrap())
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
                        self.tabs[index].name
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
                    .child(
                        Button::new("error-return")
                            .primary()
                            .h(px(40.))
                            .label("返回编辑")
                            .on_click(cx.listener(|s, _, w, c| s.dismiss(w, c))),
                    );
            }
        }
        div()
            .id("modal")
            .role(Role::Dialog)
            .aria_label(title)
            .track_focus(&self.panel_focus)
            .occlude()
            .absolute()
            .inset_0()
            .flex()
            .items_center()
            .justify_center()
            .bg(Hsla { a: 0.18, ..black() })
            .on_mouse_down(MouseButton::Left, |_, _, c| c.stop_propagation())
            .child(content)
            .focus_trap("workbench-dialog", &self.panel_focus)
    }
}
impl Render for Workbench {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let p = self.p();
        let writer = self.writer();
        let panel = self.panel.clone();
        div()
            .id("markgo")
            .key_context(if panel.is_some() {
                "MarkGoModal"
            } else {
                "MarkGo"
            })
            .track_focus(&self.root_focus)
            .relative()
            .size_full()
            .flex()
            .flex_col()
            .bg(rgb(p.rail))
            .text_color(rgb(p.ink))
            .text_size(px(13.))
            .font_family(".SystemUIFont")
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
            .when_some(panel, |d, p| d.child(self.panel_view(p, cx)))
    }
}
