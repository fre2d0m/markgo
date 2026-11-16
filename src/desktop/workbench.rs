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
    pub path: Option<PathBuf>,
    pub recovered: bool,
    pub conflict: bool,
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
    Close {
        index: usize,
        window: bool,
    },
    Error(String),
    Exit,
    Recent,
    Tabs,
    Document(usize),
    Reload {
        tab: usize,
        revision: u64,
    },
    Link {
        tab: usize,
        revision: u64,
        range: Range<usize>,
        existing: bool,
    },
    Code {
        tab: usize,
        revision: u64,
        index: usize,
    },
}

pub struct Workbench {
    pub(super) tabs: Vec<TabState>,
    pub(super) active: usize,
    next_id: usize,
    pub(super) palette: usize,
    pub(super) hovered_theme: Option<usize>,
    pub(super) hovered_tab: Option<usize>,
    pub(super) prose: f32,
    pub(super) column: f32,
    pub(super) serif: bool,
    pub(super) follow_system: bool,
    pub(super) sidebar: bool,
    pub(super) outline: bool,
    pub(super) focus_mode: bool,
    pub(super) panel: Option<Panel>,
    pub(super) presented_panel: Option<Panel>,
    pub(super) panel_anchor: Point<Pixels>,
    pub(super) panel_focus: FocusHandle,
    pub(super) panel_focus_pending: bool,
    pub(super) root_focus: FocusHandle,
    pub(super) tab_scroll: ScrollHandle,
    pub(super) sidebar_scroll: ScrollHandle,
    pub(super) search: Entity<InputState>,
    pub(super) replace: Entity<InputState>,
    pub(super) entry_label: Entity<InputState>,
    pub(super) entry_value: Entity<InputState>,
    pub(super) entry_error: Option<String>,
    pub(super) entry_label_invalid: bool,
    pub(super) search_open: bool,
    pub(super) search_options: crate::search::Search,
    pub(super) search_error: Option<String>,
    pub(super) replace_open: bool,
    pub(super) matches: Vec<Range<usize>>,
    pub(super) match_index: usize,
    pub(super) folder: Option<PathBuf>,
    pub(super) files: Vec<FolderEntry>,
    pub(super) folder_filter: Entity<InputState>,
    pub(super) recent_filter: Entity<InputState>,
    pub(super) collapsed: std::collections::HashSet<PathBuf>,
    pub(super) folder_generation: u64,
    pub(super) loading: bool,
    pub(super) ephemeral: bool,
    pub(super) notice: Option<String>,
    _subscriptions: Vec<Subscription>,
    preferences_store: Option<preferences::Store>,
    preferences_generation: u64,
    pub(super) preferences_writing: bool,
    pub(super) journal: Option<Arc<std::sync::Mutex<crate::session::OrderedJournal>>>,
    pub(super) session_loading: bool,
    pub(super) restoring: bool,
    pub(super) session_writing: bool,
    pub(super) session_generation: u64,
    pub(super) session_saved: u64,
    pub(super) session_error: Option<String>,
    pub(super) session_first_change: Option<std::time::Instant>,
    pub(super) session_last_change: std::time::Instant,
    pub(super) recent: Vec<String>,
    pub(super) exit_after_checkpoint: bool,
    pub(super) save_and_exit: bool,
    pub(super) discard_on_exit: bool,
    _checkpoint: Task<()>,
}
impl Workbench {
    fn load_preferences(&mut self, w: &mut Window, c: &mut Context<Self>) {
        let Some(path) = preferences::path() else {
            return;
        };
        let job = c.background_executor().spawn(async move {
            let previous = path
                .parent()
                .and_then(|p| p.parent())
                .map(|p| p.join("appearance.json"));
            preferences::Store::load_with_fallback(path, previous)
        });
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
                        s.serif = prefs.serif;
                        s.follow_system = prefs.system;
                        if s.follow_system {
                            s.palette = if matches!(
                                w.appearance(),
                                WindowAppearance::Dark | WindowAppearance::VibrantDark
                            ) {
                                8
                            } else {
                                0
                            };
                        }
                        let p = s.p();
                        p.apply(Some(w), c);
                        for tab in &s.tabs {
                            tab.writer.update(c, |writer, c| {
                                writer.palette = p;
                                writer.prose_size = s.prose;
                                writer.column = s.column;
                                writer.serif = s.serif;
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
            serif: self.serif,
            system: self.follow_system,
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
        let entry_label = cx.new(|cx| InputState::new(window, cx).placeholder("链接文字"));
        let entry_value = cx.new(|cx| InputState::new(window, cx).placeholder("地址或代码语言"));
        let folder_filter = cx.new(|cx| InputState::new(window, cx).placeholder("筛选文件…"));
        let recent_filter =
            cx.new(|c| InputState::new(window, c).placeholder("搜索最近文件的名称或路径…"));
        let subscriptions = vec![
            cx.on_app_quit(|s, c| s.shutdown_checkpoint(c)),
            cx.observe_window_appearance(window, |s, w, c| {
                if s.follow_system {
                    let index = if matches!(
                        w.appearance(),
                        WindowAppearance::Dark | WindowAppearance::VibrantDark
                    ) {
                        8
                    } else {
                        0
                    };
                    s.set_palette(index, w, c);
                    s.follow_system = true;
                    s.persist_preferences(c);
                }
            }),
            cx.subscribe(&folder_filter, |_, _, _: &InputEvent, c| c.notify()),
            cx.subscribe(&recent_filter, |_, _, _: &InputEvent, c| c.notify()),
            cx.subscribe_in(&entry_value, window, |s, _, event, w, c| {
                if matches!(event, InputEvent::PressEnter { .. }) {
                    s.apply_entry(false, w, c);
                }
            }),
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
            serif: false,
            follow_system: false,
            sidebar: true,
            outline: false,
            focus_mode: false,
            panel: None,
            presented_panel: None,
            panel_anchor: point(px(0.), px(56.)),
            hovered_tab: None,
            panel_focus: cx.focus_handle(),
            panel_focus_pending: false,
            root_focus: cx.focus_handle(),
            tab_scroll: ScrollHandle::new(),
            sidebar_scroll: ScrollHandle::new(),
            search,
            replace,
            entry_label,
            entry_value,
            entry_error: None,
            entry_label_invalid: false,
            search_open: false,
            search_options: crate::search::Search::default(),
            search_error: None,
            replace_open: false,
            matches: vec![],
            match_index: 0,
            folder: None,
            files: vec![],
            folder_filter,
            recent_filter,
            collapsed: Default::default(),
            folder_generation: 0,
            loading: false,
            ephemeral: preview,
            notice: None,
            _subscriptions: subscriptions,
            preferences_store: None,
            preferences_generation: 0,
            preferences_writing: false,
            journal: None,
            session_loading: !preview,
            restoring: true,
            session_writing: false,
            session_generation: 0,
            session_saved: 0,
            session_error: None,
            session_first_change: None,
            session_last_change: std::time::Instant::now(),
            recent: vec![],
            exit_after_checkpoint: false,
            save_and_exit: false,
            discard_on_exit: false,
            _checkpoint: cx.spawn(async move |this, c| {
                loop {
                    c.background_executor()
                        .timer(std::time::Duration::from_millis(250))
                        .await;
                    if this.update(c, |s, c| s.checkpoint_due(c)).is_err() {
                        break;
                    }
                }
            }),
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
        s.restoring = false;
        if !preview {
            s.load_preferences(window, cx);
            s.load_session(window, cx);
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
        // A cancelled asynchronous file dialog also restores focus. It must
        // respect a still-open modal rather than expose the editor below it.
        self.focus_writer(w, c);
    }
    #[cfg(feature = "render-check")]
    pub fn menu_fixture(&mut self) {
        // Display-only paths in ephemeral sample tabs, never opened or saved.
        for tab in self.tabs.iter_mut().take(3) {
            tab.path = Some(PathBuf::from(format!(
                r"\\?\C:\Writing\很长的项目文件夹\{}",
                tab.name
            )));
        }
        self.recent = self
            .tabs
            .iter()
            .take(3)
            .map(|t| format!("/Writing/很长的项目文件夹/{}", t.name))
            .collect();
    }
    pub(super) fn focus_writer(&self, w: &mut Window, c: &mut App) {
        let focus = if self.panel.is_some() {
            self.panel_focus.clone()
        } else {
            self.writer().focus_handle(c)
        };
        w.focus(&focus, c);
    }
    pub(super) fn add_tab(
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
            writer.serif = self.serif;
            writer.base_dir = file
                .as_ref()
                .and_then(|f| f.path.parent().map(PathBuf::from));
        });
        let id = self.next_id;
        self.next_id += 1;
        let sub = c.subscribe_in(&entity, w, |s, _, event, w, c| {
            if let super::writer::Changed::Code(index) = event {
                s.open_code(*index, w, c);
                return;
            }
            s.notice = None;
            if s.search_open {
                s.find_matches(c);
            }
            s.session_changed();
            c.notify();
        });
        self.tabs.push(TabState {
            id,
            name,
            writer: entity,
            path: file.as_ref().map(|f| f.path.clone()),
            file,
            saved,
            saving: false,
            recovered: false,
            conflict: false,
            _subscription: sub,
        });
        self.active = self.tabs.len() - 1;
        self.tab_scroll.scroll_to_item(self.active);
        self.focus_writer(w, c);
        self.session_changed();
        c.notify();
    }
    pub(super) fn new_document(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if self.tabs.len() >= 100 {
            self.notice = Some("已打开 100 篇文档，请先关闭部分文档".into());
            c.notify();
            return;
        }
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
        self.search_options.clear_scope();
    }
    pub(super) fn activate(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        if index >= self.tabs.len() {
            return;
        }
        self.cancel_input(w, c);
        self.active = index;
        self.session_changed();
        self.tab_scroll.scroll_to_item(index);
        self.focus_writer(w, c);
        c.notify();
    }
    pub(super) fn show_panel(&mut self, panel: Panel, w: &mut Window, c: &mut Context<Self>) {
        self.cancel_input(w, c);
        self.panel_anchor = w.mouse_position();
        self.panel = Some(panel);
        w.focus(&self.panel_focus, c);
        c.notify();
    }
    pub(super) fn dismiss(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.save_and_exit = false;
        self.exit_after_checkpoint = false;
        self.discard_on_exit = false;
        self.panel = None;
        self.search_open = false;
        self.matches.clear();
        self.search_options.clear_scope();
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
        if self.tabs.iter().any(|t| t.dirty(c)) {
            self.show_panel(Panel::Exit, w, c);
            false
        } else if self.ephemeral {
            true
        } else {
            self.keep_and_quit(w, c);
            false
        }
    }
    fn remove_tab(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        self.panel = None;
        self.tabs.remove(index);
        self.session_changed();
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
        if self.ephemeral {
            return;
        }
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
                        .position(|t| t.path.as_ref().is_some_and(|p| p == &file.path))
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
                    if s.tabs.len() >= 100 {
                        s.notice = Some("已打开 100 篇文档，请先关闭部分文档".into());
                        c.notify();
                        return;
                    }
                    s.remember(&file.path);
                    s.cancel_input(w, c);
                    s.add_tab(name, source, Some(file), w, c);
                }
                Err(error) => s.show_panel(Panel::Error(format!("无法打开文档：{error}")), w, c),
            });
        })
        .detach();
    }
    pub(super) fn load_folder(&mut self, path: PathBuf, w: &mut Window, c: &mut Context<Self>) {
        if self.ephemeral {
            return;
        }
        self.folder_generation += 1;
        let generation = self.folder_generation;
        self.loading = true;
        c.notify();
        let copy = path.clone();
        let worker = c
            .background_executor()
            .spawn(async move { storage::list_folder(&copy) });
        c.spawn_in(w, async move |this, c| {
            let result = worker.await;
            let _ = this.update_in(c, |s, w, c| {
                if generation != s.folder_generation {
                    return;
                }
                s.loading = false;
                match result {
                    Ok(files) => {
                        s.files = files;
                        s.folder = Some(path);
                        s.sidebar = true;
                        s.session_changed();
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
                        s.save_and_exit = false;
                        s.show_panel(Panel::Error(error.to_string()), w, c)
                    });
                }
                _ => {
                    let _ = this.update_in(c, |s, w, c| {
                        s.save_and_exit = false;
                        s.focus_writer(w, c);
                        c.notify();
                    });
                }
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
                        s.tabs[index].path = s.tabs[index].file.as_ref().map(|f|f.path.clone());
                        let base=s.tabs[index].path.as_ref().and_then(|p|p.parent().map(PathBuf::from));
                        s.tabs[index].writer.update(c,|writer,_|writer.base_dir=base);
                        s.tabs[index].recovered=false;s.tabs[index].conflict=false;
                        if let Some(path)=s.tabs[index].path.clone(){s.remember(&path);}
                        s.session_changed();
                        if let Some(all) = close {
                            if !s.tabs[index].dirty(c) {
                                s.discard_close(index, all, w, c);
                            } else {
                                s.show_panel(Panel::Close { index, window: all }, w, c);
                            }
                        }
                        if s.save_and_exit {s.save_next_and_quit(w,c);}
                    }
                    Err(error) => {s.save_and_exit=false;s.show_panel(
                        Panel::Error(format!("保存没有完成：{error}\n\n修改仍在编辑器中。可返回继续编辑，或另存为新文件。")),
                        w, c,
                    )},
                }
                c.notify();
            });
        }).detach();
    }
    pub(super) fn open_search(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.cancel_input(w, c);
        let entity = self.writer();
        let writer = entity.read(c);
        self.search_options.begin(
            writer.editor.document.source(),
            writer.editor.document.selection().range(),
        );
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
        self.search_options.reconcile(source);
        match self.search_options.matches(source, query.as_str()) {
            Ok(matches) => {
                self.matches = matches;
                self.search_error = None;
            }
            Err(error) => {
                self.matches.clear();
                self.search_error = Some(error);
            }
        }
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
        let entity = self.writer();
        let result = self.search_options.replacement(
            entity.read(c).editor.document.source(),
            self.search.read(c).value().as_str(),
            &replacement,
            (!all).then_some(self.match_index),
        );
        let (range, replacement) = match result {
            Ok(Some(edit)) => edit,
            Ok(None) => return,
            Err(error) => {
                self.search_error = Some(error);
                c.notify();
                return;
            }
        };
        self.writer().update(c, |writer, c| {
            writer.editor.cancel_composition();
            let _ = writer.editor.document.replace(range, &replacement);
            writer.editor.reproject();
            writer.select_source(writer.editor.document.selection().range(), w, c);
        });
        self.find_matches(c);
    }
    pub(super) fn toggle_search_option(
        &mut self,
        index: usize,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        match index {
            0 => self.search_options.case = !self.search_options.case,
            1 => self.search_options.words = !self.search_options.words,
            2 => self.search_options.regex = !self.search_options.regex,
            _ => self.search_options.scoped = !self.search_options.scoped,
        }
        self.find_matches(c);
        self.reveal_match(w, c);
    }
    pub(super) fn open_link(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if self.writer().read(c).read_only {
            return;
        }
        self.cancel_input(w, c);
        let entity = self.writer();
        let writer = entity.read(c);
        let selection = writer.editor.document.selection();
        let mut range = selection.range();
        let mut label = writer.editor.document.source()[range.clone()].to_owned();
        let mut url = String::new();
        let mut existing = false;
        if let Some(link) = writer.editor.projection().syntax.inlines.iter().find(|i| {
            i.style.link.is_some() && i.full.start <= range.start && range.end <= i.full.end
        }) {
            range = link.full.clone();
            label = writer.editor.document.source()[link.body.clone()].to_owned();
            url = link.style.link.clone().unwrap_or_default();
            existing = true;
        }
        let panel = Panel::Link {
            tab: self.tabs[self.active].id,
            revision: writer.editor.document.revision(),
            range,
            existing,
        };
        self.entry_label.update(c, |s, c| s.set_value(label, w, c));
        self.entry_value.update(c, |s, c| s.set_value(url, w, c));
        self.entry_error = None;
        self.show_panel(panel, w, c);
        w.focus(&self.entry_value.focus_handle(c), c);
    }
    pub(super) fn open_code(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        let entity = self.writer();
        let writer = entity.read(c);
        if writer.read_only {
            return;
        }
        let Some(code) = writer.editor.projection().syntax.codes.get(index) else {
            return;
        };
        let value = code.language.clone();
        let panel = Panel::Code {
            tab: self.tabs[self.active].id,
            revision: writer.editor.document.revision(),
            index,
        };
        self.entry_value.update(c, |s, c| s.set_value(value, w, c));
        self.entry_error = None;
        self.show_panel(panel, w, c);
        w.focus(&self.entry_value.focus_handle(c), c);
    }
    pub(super) fn apply_entry(&mut self, remove: bool, w: &mut Window, c: &mut Context<Self>) {
        let Some(panel) = self.panel.clone() else {
            return;
        };
        let label = self.entry_label.read(c).value().to_string();
        let value = self.entry_value.read(c).value().trim().to_string();
        self.entry_label_invalid = matches!(panel, Panel::Link { .. })
            && (label.trim().is_empty() || label.contains(['\n', '\r']) || label.len() > 4096);
        let (tab, revision) = match &panel {
            Panel::Link { tab, revision, .. } | Panel::Code { tab, revision, .. } => {
                (*tab, *revision)
            }
            _ => return,
        };
        let entity = self.writer();
        if self.tabs[self.active].id != tab || entity.read(c).editor.document.revision() != revision
        {
            self.entry_error = Some("文档已改变，请重新打开编辑面板".into());
            c.notify();
            return;
        }
        let result = entity.update(c, |writer, c| {
            match panel {
                Panel::Code { index, .. } => writer.editor.code_language(index, value.trim())?,
                Panel::Link { range, .. } => {
                    if label.trim().is_empty() || label.contains(['\n', '\r']) || label.len() > 4096
                    {
                        return Err("请输入单行链接文字".to_owned());
                    }
                    let lower = value.to_ascii_lowercase();
                    if !remove
                        && (value.trim().is_empty()
                            || value.contains(['\n', '\r', '\0', '<', '>'])
                            || lower.starts_with("javascript:")
                            || lower.starts_with("data:")
                            || value.len() > 8192)
                    {
                        return Err("请输入有效的网页地址或本地路径".to_owned());
                    }
                    let label = label
                        .replace('\\', "\\\\")
                        .replace('[', "\\[")
                        .replace(']', "\\]");
                    let text = if remove {
                        label.clone()
                    } else {
                        format!("[{label}](<{}>)", value.trim())
                    };
                    let start = range.start + usize::from(!remove);
                    writer
                        .editor
                        .document
                        .replace_and_select(
                            range,
                            &text,
                            crate::document::Selection {
                                anchor: start,
                                head: start + label.len(),
                            },
                        )
                        .map_err(|e| e.to_string())?;
                    writer.editor.reproject();
                }
                _ => return Ok(()),
            }
            writer.changed(w, c);
            Ok(())
        });
        if let Err(error) = result {
            self.entry_error = Some(error);
            c.notify();
        } else {
            self.dismiss(w, c);
        }
    }
    pub fn set_palette(&mut self, index: usize, w: &mut Window, c: &mut Context<Self>) {
        self.follow_system = false;
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
    pub(super) fn toggle_serif(&mut self, c: &mut Context<Self>) {
        self.serif = !self.serif;
        for tab in &self.tabs {
            tab.writer.update(c, |writer, c| {
                writer.serif = self.serif;
                c.notify();
            });
        }
        self.persist_preferences(c);
        c.notify();
    }
    pub(super) fn toggle_system(&mut self, w: &mut Window, c: &mut Context<Self>) {
        let enabled = !self.follow_system;
        if enabled {
            self.set_palette(
                if matches!(
                    w.appearance(),
                    WindowAppearance::Dark | WindowAppearance::VibrantDark
                ) {
                    8
                } else {
                    0
                },
                w,
                c,
            );
        }
        self.follow_system = enabled;
        self.persist_preferences(c);
        c.notify();
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
