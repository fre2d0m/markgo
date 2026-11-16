use super::{
    workbench::{Panel, Workbench},
    *,
};
use std::path::PathBuf;

impl Workbench {
    pub(super) fn request_reload(&mut self, id: usize, w: &mut Window, c: &mut Context<Self>) {
        let Some(tab) = self.tabs.iter().find(|t| t.id == id) else {
            return;
        };
        if tab.saving || tab.path.is_none() {
            return;
        }
        let revision = tab.writer.read(c).editor.document.revision();
        if tab.dirty(c) {
            self.show_panel(Panel::Reload { tab: id, revision }, w, c);
        } else {
            self.reload_document(id, revision, w, c);
        }
    }
    pub(super) fn reload_document(
        &mut self,
        id: usize,
        revision: u64,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        if self.ephemeral {
            return;
        }
        let Some(tab) = self.tabs.iter().find(|t| t.id == id) else {
            return;
        };
        let Some(path) = tab.path.clone() else {
            return;
        };
        let current = tab.writer.read(c).editor.snapshot();
        if current.revision != revision {
            self.show_panel(
                Panel::Error("文档已继续编辑，请重新确认载入。".into()),
                w,
                c,
            );
            return;
        }
        let worker = c
            .background_executor()
            .spawn(async move { crate::storage::open(&path) });
        c.spawn_in(w, async move |this, c| {
            let result = worker.await;
            let _ = this.update_in(c, |s, w, c| {
                let Some(index) = s.tabs.iter().position(|t| t.id == id) else {
                    return;
                };
                let latest = s.tabs[index].writer.read(c).editor.snapshot();
                if s.tabs[index].saving
                    || latest.revision != current.revision
                    || latest.source != current.source
                {
                    s.show_panel(
                        Panel::Error("读取期间文档已有新修改，保留了编辑内容。请重新载入。".into()),
                        w,
                        c,
                    );
                    return;
                }
                match result {
                    Ok(file) => {
                        s.tabs[index].writer.update(c, |writer, c| {
                            writer.cancel_composition(w, c);
                            let selection = writer.editor.document.selection();
                            let len = writer.editor.document.source().len();
                            let _ = writer.editor.document.replace(0..len, &file.source);
                            let boundary = |at: usize| {
                                let mut at = at.min(file.source.len());
                                while !file.source.is_char_boundary(at) {
                                    at -= 1;
                                }
                                at
                            };
                            let _ = writer.editor.document.select(crate::document::Selection {
                                anchor: boundary(selection.anchor),
                                head: boundary(selection.head),
                            });
                            writer.editor.reproject();
                            writer.mode(writer.editor.source_mode(), writer.read_only, w, c);
                        });
                        s.tabs[index].saved = file.source.clone();
                        s.tabs[index].file = Some(file);
                        s.tabs[index].conflict = false;
                        s.tabs[index].recovered = false;
                        s.panel = None;
                        s.session_changed();
                        s.focus_writer(w, c);
                    }
                    Err(error) => s.show_panel(
                        Panel::Error(format!("未能重新载入，原文仍在编辑器中：{error}")),
                        w,
                        c,
                    ),
                }
                c.notify();
            });
        })
        .detach();
    }
    pub(super) fn open_external(
        &mut self,
        paths: Vec<PathBuf>,
        w: &mut Window,
        c: &mut Context<Self>,
    ) {
        if self.ephemeral {
            self.notice = Some("预览不读取真实文件".into());
            c.notify();
            return;
        }
        let worker = c.background_executor().spawn(async move {
            paths
                .into_iter()
                .take(100)
                .map(|p| {
                    let directory = std::fs::symlink_metadata(&p).is_ok_and(|m| m.is_dir());
                    (p, directory)
                })
                .collect::<Vec<_>>()
        });
        c.spawn_in(w, async move |this, c| {
            let paths = worker.await;
            let _ = this.update_in(c, |s, w, c| {
                for (path, directory) in paths {
                    if directory {
                        s.load_folder(path, w, c);
                    } else {
                        s.open_path(path, w, c);
                    }
                }
            });
        })
        .detach();
    }
}
