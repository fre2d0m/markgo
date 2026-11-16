use super::{
    workbench::{Panel, Workbench},
    *,
};
use crate::{
    document::Selection,
    session::{self, Draft, Session},
};
use std::{
    path::Path,
    sync::Arc,
    time::{Duration, Instant},
};

impl Workbench {
    pub(super) fn session_changed(&mut self) {
        if self.ephemeral || self.restoring {
            return;
        }
        self.exit_after_checkpoint = false;
        self.discard_on_exit = false;
        self.session_generation = self.session_generation.saturating_add(1);
        self.session_last_change = Instant::now();
        self.session_first_change
            .get_or_insert(self.session_last_change);
    }
    pub(super) fn remember(&mut self, path: &Path) {
        let path = path.to_string_lossy().into_owned();
        self.recent.retain(|p| p != &path);
        self.recent.insert(0, path);
        self.recent.truncate(20);
        self.session_changed();
    }
    pub(super) fn load_session(&mut self, w: &mut Window, c: &mut Context<Self>) {
        let Some(path) = session::path() else {
            self.session_loading = false;
            self.session_error = Some("找不到本地恢复目录".into());
            return;
        };
        let worker = c.background_executor().spawn(async move {
            let previous = path
                .parent()
                .and_then(|p| p.parent())
                .map(|p| p.join("session.json"));
            session::Journal::load_with_fallback(path, previous).map(|(journal, state)| {
                let restored = state
                    .documents
                    .iter()
                    .cloned()
                    .map(session::restore)
                    .collect::<Vec<_>>();
                (journal, state, restored)
            })
        });
        c.spawn_in(w, async move |this, c| {
            let result = worker.await;
            let _ = this.update_in(c, |s, w, c| {
                s.session_loading = false;
                match result {
                    Ok((journal, state, restored)) => {
                        s.journal = Some(Arc::new(std::sync::Mutex::new(
                            session::OrderedJournal::new(journal),
                        )));
                        s.restoring = true;
                        let changed = s.session_generation > 0;
                        if !changed && !restored.is_empty() {
                            s.tabs.clear();
                        }
                        s.recent = state.recent;
                        s.sidebar = state.sidebar;
                        s.outline = state.outline;
                        let mut active = s.active;
                        for restored in restored {
                            let d = restored.draft;
                            if !d.path.is_empty()
                                && s.tabs.iter().any(|t| {
                                    t.path
                                        .as_ref()
                                        .is_some_and(|p| p.to_string_lossy() == d.path)
                                        && t.writer.read(c).editor.document.source() == d.source
                                })
                            {
                                continue;
                            }
                            s.add_tab(d.name, d.source.clone(), restored.file, w, c);
                            let i = s.tabs.len() - 1;
                            s.tabs[i].saved = Arc::from(d.original);
                            // add_tab keeps the successfully opened canonical
                            // path. Do not replace it with a legacy journal's
                            // non-verbatim Windows spelling and reopen a duplicate.
                            if s.tabs[i].path.is_none() {
                                s.tabs[i].path = (!d.path.is_empty()).then(|| d.path.into());
                            }
                            s.tabs[i].recovered = d.dirty;
                            s.tabs[i].conflict = restored.conflict;
                            s.tabs[i].writer.update(c, |writer, c| {
                                let text = writer.editor.document.source();
                                let boundary = |at: usize| {
                                    let mut at = at.min(text.len());
                                    while !text.is_char_boundary(at) {
                                        at -= 1;
                                    }
                                    at
                                };
                                let selection = Selection {
                                    anchor: boundary(d.anchor),
                                    head: boundary(d.head),
                                };
                                let _ = writer.editor.document.select(selection);
                                writer.mode(d.source_mode, d.reading, w, c);
                                writer.base_dir = s.tabs[i]
                                    .path
                                    .as_ref()
                                    .and_then(|p| p.parent().map(std::path::PathBuf::from));
                                writer.scroll.set_offset(point(px(0.), px(-d.scroll)));
                                c.notify();
                            });
                            if !changed && state.active == d.id {
                                active = i;
                            }
                        }
                        s.active = active.min(s.tabs.len() - 1);
                        s.restoring = false;
                        s.session_changed();
                        s.focus_writer(w, c);
                        if !state.folder.is_empty() {
                            s.load_folder(state.folder.into(), w, c);
                        }
                    }
                    Err(error) => {
                        s.session_error = Some(format!("恢复文件未载入，已保留原文件：{error}"))
                    }
                }
                c.notify();
            });
        })
        .detach();
    }
    fn session_snapshot(&self, c: &App) -> Session {
        Session {
            documents: self
                .tabs
                .iter()
                .map(|tab| {
                    let writer = tab.writer.read(c);
                    let snapshot = writer.editor.snapshot();
                    let selection = writer.editor.document.selection();
                    Draft {
                        id: tab.id.to_string(),
                        name: tab.name.clone(),
                        path: tab
                            .path
                            .as_ref()
                            .map(|p| p.to_string_lossy().into_owned())
                            .unwrap_or_default(),
                        dirty: tab.dirty(c),
                        source: snapshot.source.to_string(),
                        original: tab.saved.to_string(),
                        revision: snapshot.revision,
                        anchor: selection.anchor,
                        head: selection.head,
                        scroll: f32::from(-writer.scroll.offset().y).max(0.),
                        source_mode: writer.editor.source_mode(),
                        reading: writer.read_only,
                    }
                })
                .collect(),
            active: self.tabs[self.active].id.to_string(),
            recent: self.recent.clone(),
            folder: self
                .folder
                .as_ref()
                .map(|p| p.to_string_lossy().into_owned())
                .unwrap_or_default(),
            sidebar: self.sidebar,
            outline: self.outline,
            ..Session::default()
        }
    }
    pub(super) fn checkpoint_due(&mut self, c: &mut Context<Self>) {
        if self.exit_after_checkpoint
            && self.session_saved == self.session_generation
            && !self.preferences_writing
            && self.session_error.is_none()
            && !self.session_loading
            && !self.session_writing
        {
            c.quit();
            return;
        }
        if self.ephemeral
            || self.session_loading
            || self.session_writing
            || self.session_error.is_some()
            || self.session_generation == self.session_saved
        {
            return;
        }
        if self.exit_after_checkpoint
            || self.session_last_change.elapsed() >= Duration::from_millis(750)
            || self
                .session_first_change
                .is_some_and(|t| t.elapsed() >= Duration::from_secs(5))
        {
            self.checkpoint(c);
        }
    }
    fn checkpoint(&mut self, c: &mut Context<Self>) {
        let Some(journal) = self.journal.clone() else {
            return;
        };
        let mut state = self.session_snapshot(c);
        let generation = self.session_generation;
        if self.discard_on_exit {
            state.documents.retain(|d| !d.path.is_empty());
            for draft in &mut state.documents {
                draft.source = draft.original.clone();
                draft.dirty = false;
                draft.anchor = 0;
                draft.head = 0;
            }
        }
        self.session_writing = true;
        let worker = c.background_executor().spawn(async move {
            journal
                .lock()
                .map_err(|_| std::io::Error::other("恢复文件正在关闭"))?
                .write(generation, state)
        });
        c.spawn(async move|this,c|{
            let result=worker.await;
            let _=this.update(c,|s,c|{
                s.session_writing=false;
                match result {
                    Ok(())=>{s.session_saved=generation;if s.session_generation==generation {s.session_first_change=None;}s.checkpoint_due(c);},
                    Err(error)=>{s.session_error=Some(format!("草稿未备份：{error}"));s.exit_after_checkpoint=false;s.panel=Some(Panel::Error(format!("草稿恢复文件写入失败，应用保持打开。\n\n{error}\n\n请先将未完成的文档另存为文件。")));s.panel_focus_pending=true;},
                }
                c.notify();
            });
        }).detach();
    }
    pub(super) fn keep_and_quit(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if self.ephemeral {
            c.quit();
            return;
        }
        if self.session_loading {
            self.notice = Some("正在读取恢复文件，请稍候".into());
            c.notify();
            return;
        }
        if let Some(error) = self.session_error.clone() {
            // A corrupt/unwritable journal must not trap an already-saved
            // workspace. Keep that journal intact and allow an ordinary exit.
            if self.tabs.iter().all(|tab| !tab.dirty(c)) && !self.preferences_writing {
                c.quit();
                return;
            }
            self.show_panel(
                Panel::Error(format!("尚不能退出并保留草稿：{error}\n请先保存文档。")),
                w,
                c,
            );
            return;
        }
        self.panel = None;
        self.session_changed();
        self.exit_after_checkpoint = true;
        self.checkpoint_due(c);
        c.notify();
    }
    pub(super) fn discard_and_quit(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if self.ephemeral {
            c.quit();
            return;
        }
        // Keep live buffers intact until the final journal write succeeds.
        if self.session_error.is_some() || self.session_loading {
            self.keep_and_quit(w, c);
            return;
        }
        self.panel = None;
        self.session_changed();
        self.exit_after_checkpoint = true;
        self.discard_on_exit = true;
        self.checkpoint_due(c);
        c.notify();
    }
    pub(super) fn save_next_and_quit(&mut self, w: &mut Window, c: &mut Context<Self>) {
        self.save_and_exit = true;
        if let Some(index) = self.tabs.iter().position(|t| t.dirty(c)) {
            self.active = index;
            self.save(false, None, w, c);
        } else {
            self.save_and_exit = false;
            self.keep_and_quit(w, c);
        }
    }
    pub(super) fn retry_checkpoint(&mut self, w: &mut Window, c: &mut Context<Self>) {
        if self.journal.is_none() {
            return;
        }
        self.session_error = None;
        self.panel = None;
        self.session_changed();
        self.checkpoint(c);
        self.focus_writer(w, c);
        c.notify();
    }
    pub(super) fn shutdown_checkpoint(&self, c: &mut Context<Self>) -> Task<()> {
        let journal = self.journal.clone();
        let state = self.session_snapshot(c);
        let generation = self.session_generation;
        let skip = self.ephemeral
            || self.session_error.is_some()
            || (self.exit_after_checkpoint && self.session_saved == generation);
        c.background_executor().spawn(async move {
            if !skip
                && let Some(journal) = journal
                && let Ok(mut journal) = journal.lock()
                && let Err(error) = journal.write(generation, state)
            {
                eprintln!("MarkGo: shutdown recovery failed: {error}");
            }
        })
    }
}
