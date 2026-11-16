use super::{
    SAMPLE,
    theme::{PALETTES, Palette},
    writer::Writer,
};
use crate::preferences::{self, Preferences};
use crate::storage::{self, FileSnapshot, FolderEntry};
use gpui_kit::{
    component::input::{InputEvent, InputState},
    *,
};
use std::{ops::Range, path::PathBuf, sync::Arc};

pub(super) struct TabState {
    pub id: usize,
    pub name: String,
    pub writer: Entity<Writer>,
    pub file: Option<FileSnapshot>,
    pub saved: Arc<str>,
    pub saving: bool,
    _subscription: Subscription,
}
impl TabState {
    pub fn dirty(&self, cx: &App) -> bool {
        self.writer.read(cx).editor.document.source() != self.saved.as_ref()
    }
}
#[derive(Clone, PartialEq)]
pub(super) enum Panel {
    Appearance,
    View,
    Format,
    Close { index: usize, window: bool },
    Error(String),
}

pub struct Workbench {
    pub(super) tabs: Vec<TabState>,
    pub(super) active: usize,
    next_id: usize,
    pub(super) palette: usize,
    pub(super) hovered_theme: Option<usize>,
    pub(super) prose: f32,
    pub(super) column: f32,
    pub(super) sidebar: bool,
    pub(super) outline: bool,
    pub(super) focus_mode: bool,
    pub(super) panel: Option<Panel>,
    pub(super) panel_focus: FocusHandle,
    pub(super) root_focus: FocusHandle,
    pub(super) tab_scroll: ScrollHandle,
    pub(super) sidebar_scroll: ScrollHandle,
    pub(super) search: Entity<InputState>,
    pub(super) replace: Entity<InputState>,
    pub(super) search_open: bool,
    pub(super) replace_open: bool,
    pub(super) matches: Vec<Range<usize>>,
    pub(super) match_index: usize,
    pub(super) folder: Option<PathBuf>,
    pub(super) files: Vec<FolderEntry>,
    pub(super) loading: bool,
    pub(super) ephemeral: bool,
    pub(super) notice: Option<String>,
    _subscriptions: Vec<Subscription>,
    preferences_store: Option<preferences::Store>,
    preferences_generation: u64,
    preferences_writing: bool,
}
impl Workbench {
    fn load_preferences(&mut self, w: &mut Window, c: &mut Context<Self>) {
        let Some(path) = preferences::path() else {
            return;
        };
        let job = c
            .background_executor()
            .spawn(async move { preferences::Store::load(path) });
        c.spawn_in(w, async move |this, c| {
            let result = job.await;
            let _ = this.update_in(c, |s, w, c| match result {
                Ok((store, prefs)) => {
                    s.preferences_store = Some(store);
                    if s.preferences_generation == 0 {
                        s.palette = PALETTES
                            .iter()
                            .position(|p| p.id == prefs.theme)
                            .unwrap_or(0);
                        s.prose = f32::from(prefs.prose);
                        s.column = f32::from(prefs.column);
                        let p = s.p();
                        p.apply(Some(w), c);
                        for tab in &s.tabs {
                            tab.writer.update(c, |writer, c| {
                                writer.palette = p;
                                writer.prose_size = s.prose;
                                writer.column = s.column;
                                c.notify();
                            });
                        }
                    } else {
                        s.persist_preferences(c);
                    }
                    c.notify();
                }
                Err(error) => {
                    s.notice = Some(format!("未加载外观设置：{error}"));
                    c.notify();
                }
            });
        })
        .detach();
    }
    fn persist_preferences(&mut self, c: &mut Context<Self>) {
        if self.ephemeral {
            return;
        }
        self.preferences_generation += 1;
        if self.preferences_writing {
            return;
        }
        let Some(store) = self.preferences_store.take() else {
            return;
        };
        self.preferences_writing = true;
        let generation = self.preferences_generation;
        let prefs = Preferences {
            theme: self.p().id.into(),
            prose: self.prose as u8,
            column: self.column as u16,
        };
        let job = c
            .background_executor()
            .spawn(async move { store.write(prefs) });
        c.spawn(async move |this, c| {
            let result = job.await;
            let _ = this.update(c, |s, c| {
                s.preferences_writing = false;
                match result {
                    Ok(store) => {
                        s.preferences_store = Some(store);
                        if s.preferences_generation != generation {
                            s.persist_preferences(c);
                        }
                    }
                    Err(error) => s.notice = Some(format!("外观已应用，但未能保存设置：{error}")),
                }
                c.notify();
            });
        })
        .detach();
    }
    pub fn new(preview: bool, window: &mut Window, cx: &mut Context<Self>) -> Self {
        let search = cx.new(|cx| InputState::new(window, cx).placeholder("查找文档内容"));
        let replace = cx.new(|cx| InputState::new(window, cx).placeholder("替换为…"));
        let subscriptions =
            vec![
                cx.subscribe_in(&search, window, |s, _, event, w, c| match event {
                    InputEvent::Change => {
                        s.find_matches(c);
                        s.reveal_match(w, c);
                    }
                    InputEvent::PressEnter { shift, .. } => s.next_match(!shift, w, c),
                    _ => {}
                }),
            ];
        let mut s = Self {
            tabs: vec![],
            active: 0,
            next_id: 1,
            palette: 0,
            hovered_theme: None,
            prose: 18.,
            column: 720.,
            sidebar: true,
            outline: false,
            focus_mode: false,
            panel: None,
            panel_focus: cx.focus_handle(),
            root_focus: cx.focus_handle(),
            tab_scroll: ScrollHandle::new(),
            sidebar_scroll: ScrollHandle::new(),
            search,
            replace,
            search_open: false,
            replace_open: false,
            matches: vec![],
            match_index: 0,
            folder: None,
            files: vec![],
            loading: false,
            ephemeral: preview,
            notice: None,
            _subscriptions: subscriptions,
            preferences_store: None,
            preferences_generation: 0,
            preferences_writing: false,
        };
        if preview {
            s.add_tab(
                "写作，从这里开始.md".into(),
                SAMPLE.into(),
                None,
                window,
                cx,
            );
            s.add_tab(
                "设计笔记.md".into(),
                "# 设计笔记\n\n让工具安静，让文字发光。\n".into(),
                None,
                window,
                cx,
            );
            s.add_tab(
                "一个很长的文档名称，用来检查标签栏布局.md".into(),
                "# 长标题\n\n每一份文档拥有自己的编辑状态。\n".into(),
                None,
                window,
                cx,
            );
            s.active = 0;
        } else {
            s.new_document(window, cx);
        }
        let weak = cx.weak_entity();
        window.on_window_should_close(cx, move |w, c| {
            weak.update(c, |s, c| s.request_window_close(w, c))
                .unwrap_or(true)
        });
        s.focus_writer(window, cx);
        if !preview {
            s.load_preferences(window, cx);
        }
        s
    }
    pub(super) fn p(&self) -> Palette {
        PALETTES[self.palette]
    }
    pub(super) fn writer(&self) -> Entity<Writer> {
        self.tabs[self.active].writer.clone()
    }
    #[cfg(feature = "render-check")]
    pub fn inspection(&self, cx: &App) -> (usize, usize, String, crate::document::Selection, bool) {
        let tab = &self.tabs[self.active];
        let writer = tab.writer.read(cx);
        (
            self.tabs.len(),
            tab.id,
            writer.editor.document.source().to_owned(),
            writer.editor.document.selection(),
            tab.dirty(cx),
        )
    }
    #[cfg(feature = "render-check")]
    pub fn input_entity(&self) -> Entity<Writer> {
        self.writer()
    }
    #[cfg(feature = "render-check")]
    pub fn empty_preview(w: &mut Window, c: &mut Context<Self>) -> Self {
        let mut s = Self::new(true, w, c);
        s.tabs.clear();
        s.next_id = 1;
        s.new_document(w, c);
        s
    }
    #[cfg(feature = "render-check")]
    pub fn render_error_fixture(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.show_panel(
            Panel::Error("文件已被其他程序修改。你的修改仍保留在编辑器中，请另存副本。".into()),
            w,
            c,
        );
    }
    pub(super) fn focus_writer(&self, w: &mut Window, c: &mut App) {
        let focus = self.writer().focus_handle(c);
        w.focus(&focus, c);
    }
    fn add_tab(
        &mut self,
        name: String,
        source: String,
        file: Option<FileSnapshot>,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        let saved: Arc<str> = Arc::from(source.as_str());
        let entity = c.new(|c| Writer::new(source, self.p(), c));
        entity.update(c, |writer, _| {
            writer.prose_size = self.prose;
            writer.column = self.column;
        });
        let id = self.next_id;
        self.next_id += 1;
        let sub = c.subscribe(&entity, |s, _, _, c| {
            s.notice = None;
            if s.search_open {
                s.find_matches(c);
            }
            c.notify();
        });
        self.tabs.push(TabState {
            id,
            name,
            writer: entity,
            file,
            saved,
            saving: false,
            _subscription: sub,
        });
        self.active = self.tabs.len() - 1;
        self.tab_scroll.scroll_to_item(self.active);
        self.focus_writer(w, c);
        c.notify();
    }
    pub(super) fn new_document(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if !self.tabs.is_empty() {
            self.cancel_input(w, c);
        }
        self.add_tab(
            format!("未命名 {}.md", self.next_id),
            String::new(),
            None,
            w,
            c,
        );
    }
    fn cancel_input(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.writer()
            .update(c, |writer, c| writer.cancel_composition(w, c));
        self.panel = None;
        self.search_open = false;
        self.matches.clear();
    }
    pub(super) fn activate(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        if index >= self.tabs.len() {
            return;
        }
        self.cancel_input(w, c);
        self.active = index;
        self.tab_scroll.scroll_to_item(index);
        self.focus_writer(w, c);
        c.notify();
    }
    pub(super) fn show_panel(&mut self, panel: Panel, w: &mut Window, c: &mut Context<Self>) {
        self.cancel_input(w, c);
        self.panel = Some(panel);
        w.focus(&self.panel_focus, c);
        c.notify();
    }
    pub(super) fn dismiss(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.panel = None;
        self.search_open = false;
        self.matches.clear();
        self.focus_writer(w, c);
        c.notify();
    }
    pub(super) fn request_close(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        if self.tabs[index].saving {
            return;
        }
        if self.tabs[index].dirty(c) {
            self.show_panel(
                Panel::Close {
                    index,
                    window: false,
                },
                w,
                c,
            );
        } else {
            self.remove_tab(index, w, c);
        }
    }
    pub(super) fn request_window_close(&mut self, w: &mut Window, c: &mut Context<Self>) -> bool {
        if self.tabs.iter().any(|t| t.saving) {
            self.notice = Some("正在保存，请稍候再关闭".into());
            c.notify();
            return false;
        }
        if let Some(index) = self.tabs.iter().position(|t| t.dirty(c)) {
            self.show_panel(
                Panel::Close {
                    index,
                    window: true,
                },
                w,
                c,
            );
            false
        } else {
            true
        }
    }
    fn remove_tab(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        self.panel = None;
        self.tabs.remove(index);
        if self.tabs.is_empty() {
            self.new_document(w, c);
        } else {
            if index < self.active {
                self.active -= 1
            }
            self.active = self.active.min(self.tabs.len() - 1);
            self.focus_writer(w, c);
        }
        c.notify();
    }
    pub(super) fn discard_close(
        &mut self,
        index: usize,
        all: bool,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        self.remove_tab(index, w, c);
        if all && self.request_window_close(w, c) {
            w.remove_window();
        }
    }
    pub(super) fn prompt_open(&mut self, folder: bool, w: &mut Window, c: &mut Context<Self>) {
        if self.ephemeral {
            self.notice = Some("预览使用内存中的示例文档".into());
            c.notify();
            return;
        }
        self.cancel_input(w, c);
        let receiver = c.prompt_for_paths(PathPromptOptions {
            files: !folder,
            directories: folder,
            multiple: !folder,
            prompt: Some(
                if folder {
                    "打开文件夹"
                } else {
                    "打开 Markdown 文档"
                }
                .into(),
            ),
        });
        c.spawn_in(w, async move |this, c| match receiver.await {
            Ok(Ok(Some(paths))) => {
                let _ = this.update_in(c, |s, w, c| {
                    if folder {
                        if let Some(path) = paths.first() {
                            s.load_folder(path.clone(), w, c)
                        }
                    } else {
                        for path in paths {
                            s.open_path(path, w, c)
                        }
                    }
                });
            }
            Ok(Err(error)) => {
                let _ = this.update_in(c, |s, w, c| {
                    s.show_panel(Panel::Error(error.to_string()), w, c)
                });
            }
            _ => {}
        })
        .detach();
    }
    pub(super) fn open_path(&mut self, path: PathBuf, w: &mut Window, c: &mut Context<Self>) {
        let worker = c
            .background_executor()
            .spawn(async move { storage::open(&path) });
        c.spawn_in(w, async move |this, c| {
            let result = worker.await;
            let _ = this.update_in(c, |s, w, c| match result {
                Ok(file) => {
                    if let Some(index) = s
                        .tabs
                        .iter()
                        .position(|t| t.file.as_ref().is_some_and(|f| f.path == file.path))
                    {
                        s.activate(index, w, c);
                        return;
                    }
                    let name = file
                        .path
                        .file_name()
                        .unwrap_or_default()
                        .to_string_lossy()
                        .into_owned();
                    let source = file.source.to_string();
                    s.cancel_input(w, c);
                    s.add_tab(name, source, Some(file), w, c);
                }
                Err(error) => s.show_panel(Panel::Error(format!("无法打开文档：{error}")), w, c),
            });
        })
        .detach();
    }
    fn load_folder(&mut self, path: PathBuf, w: &mut Window, c: &mut Context<Self>) {
        self.loading = true;
        c.notify();
        let copy = path.clone();
        let worker = c
            .background_executor()
            .spawn(async move { storage::list_folder(&copy) });
        c.spawn_in(w, async move |this, c| {
            let result = worker.await;
            let _ = this.update_in(c, |s, w, c| {
                s.loading = false;
                match result {
                    Ok(files) => {
                        s.files = files;
                        s.folder = Some(path);
                        s.sidebar = true;
                    }
                    Err(error) => {
                        s.show_panel(Panel::Error(format!("无法读取文件夹：{error}")), w, c)
                    }
                }
                c.notify();
            });
        })
        .detach();
    }
    pub(super) fn save(
        &mut self,
        save_as: bool,
        close: Option<(usize, bool)>,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        let index = close.map_or(self.active, |c| c.0);
        if self.ephemeral {
            self.notice = Some("预览文档保留在内存中，不写入磁盘".into());
            c.notify();
            return;
        }
        if self.tabs[index].saving {
            return;
        }
        let id = self.tabs[index].id;
        if save_as || self.tabs[index].file.is_none() {
            let directory = self.tabs[index]
                .file
                .as_ref()
                .and_then(|f| f.path.parent())
                .map(PathBuf::from)
                .or_else(|| self.folder.clone())
                .unwrap_or_else(|| {
                    std::env::var_os("HOME")
                        .map(PathBuf::from)
                        .unwrap_or_else(|| PathBuf::from("."))
                });
            let receiver = c.prompt_for_new_path(&directory, Some(&self.tabs[index].name));
            c.spawn_in(w, async move |this, c| match receiver.await {
                Ok(Ok(Some(path))) => {
                    let _ = this.update_in(c, |s, w, c| {
                        s.write_document(id, Some(path), close.map(|c| c.1), w, c)
                    });
                }
                Ok(Err(error)) => {
                    let _ = this.update_in(c, |s, w, c| {
                        s.show_panel(Panel::Error(error.to_string()), w, c)
                    });
                }
                _ => {}
            })
            .detach();
        } else {
            self.write_document(id, None, close.map(|c| c.1), w, c);
        }
    }
    fn write_document(
        &mut self,
        id: usize,
        path: Option<PathBuf>,
        close: Option<bool>,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        let Some(index) = self.tabs.iter().position(|t| t.id == id) else {
            return;
        };
        if self.tabs[index].saving {
            return;
        }
        self.tabs[index].saving = true;
        c.notify();
        let snapshot = self.tabs[index].writer.read(c).editor.snapshot();
        let file = self.tabs[index].file.clone();
        let worker = c.background_executor().spawn(async move {
            if let Some(path) = path {
                storage::create(&path, snapshot.source)
            } else {
                storage::save(&file.expect("existing file"), snapshot.source)
            }
        });
        c.spawn_in(w, async move |this, c| {
            let result = worker.await;
            let _ = this.update_in(c, |s, w, c| {
                let Some(index) = s.tabs.iter().position(|t| t.id == id) else {
                    return;
                };
                s.tabs[index].saving = false;
                match result {
                    Ok(file) => {
                        s.tabs[index].saved = file.source.clone();
                        s.tabs[index].name = file.path.file_name().unwrap_or_default()
                            .to_string_lossy().into_owned();
                        s.tabs[index].file = Some(file);
                        if let Some(all) = close {
                            if !s.tabs[index].dirty(c) {
                                s.discard_close(index, all, w, c);
                            } else {
                                s.show_panel(Panel::Close { index, window: all }, w, c);
                            }
                        }
                    }
                    Err(error) => s.show_panel(
                        Panel::Error(format!("保存没有完成：{error}\n\n修改仍在编辑器中。可返回继续编辑，或另存为新文件。")),
                        w, c,
                    ),
                }
                c.notify();
            });
        }).detach();
    }
    pub(super) fn open_search(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.cancel_input(w, c);
        self.search_open = true;
        self.find_matches(c);
        self.reveal_match(w, c);
        w.focus(&self.search.focus_handle(c), c);
        c.notify();
    }
    fn find_matches(&mut self, c: &mut Context<Self>) {
        let query = self.search.read(c).value();
        let writer = self.writer();
        let source = writer.read(c).editor.document.source();
        self.matches = if query.is_empty() {
            vec![]
        } else {
            source
                .match_indices(query.as_str())
                .map(|(i, _)| i..i + query.len())
                .collect()
        };
        self.match_index = self.match_index.min(self.matches.len().saturating_sub(1));
        c.notify();
    }
    fn reveal_match(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if let Some(range) = self.matches.get(self.match_index).cloned() {
            self.writer()
                .update(c, |writer, c| writer.select_source(range, w, c));
        }
    }
    pub(super) fn next_match(&mut self, forward: bool, w: &mut Window, c: &mut Context<Self>) {
        if self.matches.is_empty() {
            return;
        }
        self.match_index = if forward {
            (self.match_index + 1) % self.matches.len()
        } else {
            (self.match_index + self.matches.len() - 1) % self.matches.len()
        };
        self.reveal_match(w, c);
        c.notify();
    }
    pub(super) fn replace_match(&mut self, all: bool, w: &mut Window, c: &mut Context<Self>) {
        if self.matches.is_empty() || self.writer().read(c).read_only {
            return;
        }
        let replacement = self.replace.read(c).value().to_string();
        let matches = self.matches.clone();
        let index = self.match_index;
        self.writer().update(c, |writer, c| {
            writer.editor.cancel_composition();
            if all {
                let source = writer.editor.document.source();
                let len = source.len();
                let mut next = source.to_owned();
                for range in matches.iter().rev() {
                    next.replace_range(range.clone(), &replacement);
                }
                let _ = writer.editor.document.replace(0..len, &next);
            } else {
                let _ = writer
                    .editor
                    .document
                    .replace(matches[index].clone(), &replacement);
            }
            writer.editor.reproject();
            writer.select_source(writer.editor.document.selection().range(), w, c);
        });
        self.find_matches(c);
    }
    pub fn set_palette(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        self.palette = index;
        let p = self.p();
        p.apply(Some(w), c);
        for tab in &self.tabs {
            tab.writer.update(c, |writer, c| {
                writer.palette = p;
                c.notify();
            });
        }
        c.notify();
        self.persist_preferences(c);
    }
    pub(super) fn resize_prose(&mut self, delta: f32, c: &mut Context<Self>) {
        self.prose = (self.prose + delta).clamp(14., 24.);
        for t in &self.tabs {
            t.writer.update(c, |w, c| {
                w.prose_size = self.prose;
                c.notify();
            });
        }
        c.notify();
        self.persist_preferences(c);
    }
    pub(super) fn set_column(&mut self, column: f32, c: &mut Context<Self>) {
        self.column = column;
        for t in &self.tabs {
            t.writer.update(c, |w, c| {
                w.column = column;
                c.notify();
            });
        }
        c.notify();
        self.persist_preferences(c);
    }
    pub(super) fn set_mode(
        &mut self,
        source: bool,
        read_only: bool,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        self.panel = None;
        self.writer()
            .update(c, |writer, c| writer.mode(source, read_only, w, c));
        c.notify();
    }
}
