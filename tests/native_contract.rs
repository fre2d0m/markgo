use markgo::{document::Selection, editing::Editor, storage};
use std::{
    path::PathBuf,
    sync::{
        Arc,
        atomic::{AtomicU64, Ordering},
    },
};

#[test]
fn composition_is_staged_and_one_undo_restores_source() {
    let mut e = Editor::new("# 标题\n\n正文");
    e.select(Selection::caret(0)).unwrap();
    e.mark(None, "ni", Some(2..2)).unwrap();
    assert_eq!(e.text(), "ni标题\n\n正文");
    assert_eq!(e.snapshot().source.as_ref(), "# 标题\n\n正文");
    e.mark(None, "你", Some(3..3)).unwrap();
    e.commit_composition(None).unwrap();
    assert_eq!(e.document.source(), "# 你标题\n\n正文");
    e.undo();
    assert_eq!(e.document.source(), "# 标题\n\n正文");
    e.redo();
    assert_eq!(e.document.source(), "# 你标题\n\n正文");
}
#[test]
fn native_composition_can_replace_a_subrange_of_preedit() {
    let mut e = Editor::new("abcd");
    e.select(Selection { anchor: 1, head: 3 }).unwrap();
    e.mark(None, "nihao", None).unwrap();
    e.mark(Some(3..6), "好", None).unwrap();
    assert_eq!(e.text(), "ani好d");
    assert_eq!(e.snapshot().source.as_ref(), "abcd");
    e.replace_native(None, "你好").unwrap();
    assert_eq!(e.document.source(), "a你好d");
    e.undo();
    assert_eq!(e.document.source(), "abcd");
}
#[test]
fn composition_expansion_retains_surrounding_source_until_commit() {
    let mut e = Editor::new("abcd");
    e.select(Selection::caret(2)).unwrap();
    e.mark(None, "你", None).unwrap();
    e.mark(Some(1..6), "好", None).unwrap();
    assert_eq!(e.text(), "a好d");
    assert_eq!(e.snapshot().source.as_ref(), "abcd");
    e.commit_composition(None).unwrap();
    assert_eq!(e.document.source(), "a好d");
    e.undo();
    assert_eq!(e.document.source(), "abcd");
}
#[test]
fn composition_into_empty_heading_keeps_valid_markdown() {
    let mut e = Editor::new("### ");
    e.mark(None, "标题", None).unwrap();
    e.commit_composition(None).unwrap();
    assert_eq!(e.document.source(), "### 标题");
    e.undo();
    assert_eq!(e.document.source(), "### ");
}

#[test]
fn typing_heading_markers_waits_for_space_and_backspace_exits() {
    for level in 1..=6 {
        let mut e = Editor::new("");
        for count in 1..=level {
            e.insert("#").unwrap();
            assert_eq!(e.text(), "#".repeat(count));
            assert_eq!(e.selection(), Selection::caret(count));
            assert_eq!(e.heading_at(count), 0);
        }
        e.insert(" ").unwrap();
        assert_eq!(e.text(), "");
        assert_eq!(e.heading_at(0), level as u8);
        let source = e.document.source().to_owned();
        e.delete(false);
        assert_eq!(e.document.source(), "");
        e.undo();
        assert_eq!(e.document.source(), source);
        e.insert("标题 👩‍💻").unwrap();
        assert_eq!(e.text(), "标题 👩‍💻");
        e.select(Selection::caret(0)).unwrap();
        e.delete(false);
        assert_eq!(e.document.source(), "标题 👩‍💻");
        e.undo();
        assert_eq!(e.text(), "标题 👩‍💻");
        assert_eq!(e.heading_at(0), level as u8);
    }
}

#[test]
fn hashes_in_prose_and_code_remain_literal() {
    for source in [
        "#",
        "##",
        "标签 #话题",
        "word#hash",
        "####### text",
        "```\n# code\n```",
    ] {
        let e = Editor::new(source);
        assert!(e.text().contains('#'), "{source:?}");
    }
}
#[test]
fn view_switch_cancels_preedit_and_preserves_source_selection() {
    let mut e = Editor::new("# Heading\nbody");
    e.select(Selection::caret(2)).unwrap();
    let selection = e.document.selection();
    e.mark(None, "中文", None).unwrap();
    e.set_source_mode(true);
    assert_eq!(e.text(), "# Heading\nbody");
    assert_eq!(e.document.selection(), selection);
    e.set_source_mode(false);
    assert_eq!(e.selection().head, 2);
}
#[test]
fn native_utf16_covers_emoji_and_surrogate_split() {
    let e = Editor::new("# a🙂中");
    assert_eq!(e.byte_range(2..3), 1..5);
    assert_eq!(e.utf16_range(1..5), 1..3);
}
#[test]
fn navigation_and_deletion_treat_composed_graphemes_as_one() {
    let mut e = Editor::new("👩‍💻e\u{301}中");
    e.horizontal(true, false);
    assert_eq!(e.selection().head, "👩‍💻".len());
    e.horizontal(true, false);
    assert_eq!(e.selection().head, "👩‍💻e\u{301}".len());
    e.delete(false);
    assert_eq!(e.document.source(), "👩‍💻中");
    e.undo();
    assert_eq!(e.document.source(), "👩‍💻e\u{301}中");
}
#[test]
fn structured_slash_conversion_is_one_history_entry() {
    let mut e = Editor::new("before\n/h1\nafter");
    e.set_source_mode(true);
    e.select(Selection::caret(10)).unwrap();
    e.block("# ", "");
    assert_eq!(e.document.source(), "before\n# \nafter");
    e.undo();
    assert_eq!(e.document.source(), "before\n/h1\nafter");
}

static TEMP: AtomicU64 = AtomicU64::new(0);

#[test]
fn enter_keeps_heading_suffix_and_existing_line_endings() {
    let mut e = Editor::new("# Title ###\r\nnext");
    e.select(Selection::caret(2)).unwrap();
    e.enter().unwrap();
    assert_eq!(e.document.source(), "# Ti ###\r\ntle\r\nnext");
    e.undo();
    assert_eq!(e.document.source(), "# Title ###\r\nnext");
}
#[test]
fn enter_continues_lists_tasks_quotes_and_exits_empty_items() {
    for (source, expected) in [
        ("- first", "- first\n- "),
        ("3. first", "3. first\n4. "),
        ("- [x] done", "- [x] done\n- [ ] "),
        ("> quote", "> quote\n> "),
        ("- ", ""),
        ("- [ ] ", ""),
    ] {
        let mut e = Editor::new(source);
        e.document.select(Selection::caret(source.len())).unwrap();
        e.enter().unwrap();
        assert_eq!(e.document.source(), expected);
        e.undo();
        assert_eq!(e.document.source(), source);
    }
    let mut e = Editor::new("```\n- literal\n```");
    e.document.select(Selection::caret(13)).unwrap();
    e.enter().unwrap();
    assert_eq!(e.document.source(), "```\n- literal\n\n```");
}
#[test]
fn block_conversion_does_not_nest_existing_heading_markers() {
    let mut e = Editor::new("### Title ###\nbody");
    e.block("# ", "");
    assert_eq!(e.document.source(), "# Title\nbody");
    e.undo();
    assert_eq!(e.document.source(), "### Title ###\nbody");
}
#[test]
fn inline_toolbar_can_remove_a_format_in_one_transaction() {
    let mut e = Editor::new("before **word** after");
    e.document
        .select(Selection {
            anchor: 9,
            head: 13,
        })
        .unwrap();
    e.format_inline("**");
    assert_eq!(e.document.source(), "before word after");
    e.undo();
    assert_eq!(e.document.source(), "before **word** after");
}

#[test]
fn composition_keeps_heading_metrics_and_native_selection_staged() {
    let mut e = Editor::new("# 标题\n\n## 下一节");
    e.select(Selection::caret(0)).unwrap();
    e.mark(None, "nihao", None).unwrap();
    assert_eq!(e.heading_at(0), 1);
    e.select_native(Selection { anchor: 2, head: 5 }).unwrap();
    assert!(e.marked_range().is_some());
    assert_eq!(e.selection(), Selection { anchor: 2, head: 5 });
    assert_eq!(e.snapshot().source.as_ref(), "# 标题\n\n## 下一节");
    let next = e.text().find("下一节").unwrap();
    assert_eq!(e.heading_at(next), 2);
}
#[test]
fn invalid_native_selection_does_not_mutate_preedit() {
    let mut e = Editor::new("# 原文");
    e.mark(None, "你好", Some(0..0)).unwrap();
    let before = (e.text(), e.selection(), e.snapshot().source);
    assert!(e.select_native(Selection::caret(1)).is_err());
    assert_eq!((e.text(), e.selection(), e.snapshot().source), before);
    e.commit_composition(None).unwrap();
    assert_eq!(e.document.source(), "# 你好原文");
}
#[test]
fn slash_does_not_arm_in_literal_contexts_or_nonempty_paragraphs() {
    for (source, at) in [
        ("```\n\n```", 4),
        ("---\ntitle: a\n\n---\n", 13),
        ("# ", 2),
        ("abc", 3),
        ("- ", 2),
        ("/tmp", 4),
    ] {
        let mut e = Editor::new(source);
        e.document.select(Selection::caret(at)).unwrap();
        assert!(!e.can_open_slash(), "{source:?}");
    }
    assert!(Editor::new("").can_open_slash());
}
#[test]
fn typing_coalesces_contiguous_transactions_only() {
    use markgo::document::Document;
    let mut d = Document::new("");
    d.replace(0..0, "你").unwrap();
    d.replace(3..3, "好").unwrap();
    d.coalesce_typing();
    d.undo();
    assert_eq!(d.source(), "");
    d.redo();
    assert_eq!(d.source(), "你好");
    d.replace(0..0, "前").unwrap();
    d.coalesce_typing();
    d.undo();
    assert_eq!(d.source(), "你好");
}
#[test]
fn appearance_round_trip_and_conflict_protection() {
    use markgo::preferences::{Preferences, Store, THEMES};
    let dir = Dir::new();
    let path = dir.0.join("settings.conf");
    let (mut store, _) = Store::load(path.clone()).unwrap();
    for theme in THEMES {
        let settings = Preferences {
            theme: theme.into(),
            prose: 18,
            column: 720,
            ..Preferences::default()
        };
        store = store.write(settings.clone()).unwrap();
        assert_eq!(
            Preferences::parse(&std::fs::read_to_string(&path).unwrap()).unwrap(),
            settings
        );
    }
    std::fs::write(&path, "future-version\n").unwrap();
    assert!(store.write(Preferences::default()).is_err());
    assert_eq!(std::fs::read_to_string(&path).unwrap(), "future-version\n");
    assert!(Store::load(path).is_err());
}
struct Dir(PathBuf);
impl Dir {
    fn new() -> Self {
        let p = std::env::temp_dir().join(format!(
            "markgo-native-test-{}-{}",
            std::process::id(),
            TEMP.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&p).unwrap();
        Self(p)
    }
}
impl Drop for Dir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

#[test]
fn guarded_save_preserves_bytes_and_rejects_external_change() {
    let dir = Dir::new();
    let path = dir.0.join("writing.md");
    let original = "\u{feff}# 中文\r\n\r\nline  \n";
    let file = storage::create(&path, Arc::from(original)).unwrap();
    assert_eq!(storage::open(&path).unwrap().source.as_ref(), original);
    let saved = storage::save(&file, Arc::from("# changed\r\n")).unwrap();
    assert_eq!(std::fs::read_to_string(&path).unwrap(), "# changed\r\n");
    std::fs::write(&path, "external").unwrap();
    assert!(storage::save(&saved, Arc::from("mine")).is_err());
    assert_eq!(std::fs::read_to_string(&path).unwrap(), "external");
    assert_eq!(std::fs::read_dir(&dir.0).unwrap().count(), 1);
}
#[test]
fn windows_path_labels_hide_only_transport_prefixes() {
    use std::path::Path;
    for (path, label) in [
        (r"\\?\C:\Writing\中文.md", r"C:\Writing\中文.md"),
        (r"\\?\UNC\server\share\note.md", r"\\server\share\note.md"),
        (r"\\?\Volume{keep}\note.md", r"\\?\Volume{keep}\note.md"),
        ("/writing/notes.md", "/writing/notes.md"),
    ] {
        assert_eq!(storage::display_path(Path::new(path)), label);
    }
}
#[test]
fn restored_file_identity_is_canonical_without_losing_dirty_text() {
    use markgo::session::{Draft, restore};
    let dir = Dir::new();
    std::fs::create_dir(dir.0.join("nested")).unwrap();
    let path = dir.0.join("note.md");
    std::fs::write(&path, "base").unwrap();
    let legacy = dir.0.join("nested").join("..").join("note.md");
    let draft = Draft {
        path: legacy.to_string_lossy().into_owned(),
        source: "my unsaved writing".into(),
        original: "base".into(),
        dirty: true,
        ..Default::default()
    };
    let restored = restore(draft.clone());
    assert_eq!(
        restored.draft.path,
        storage::open(&path).unwrap().path.to_string_lossy()
    );
    assert_eq!(restored.draft.source, draft.source);
    assert!(!restored.conflict);
    std::fs::write(&path, "external edit").unwrap();
    let conflict = restore(draft);
    assert!(conflict.conflict);
    assert!(conflict.file.is_none());
    assert_eq!(conflict.draft.source, "my unsaved writing");
    assert_eq!(std::fs::read_to_string(&path).unwrap(), "external edit");
}
#[test]
fn save_as_never_overwrites_an_existing_destination() {
    let dir = Dir::new();
    let path = dir.0.join("writing.md");
    std::fs::write(&path, "keep me").unwrap();
    assert!(storage::create(&path, Arc::from("new")).is_err());
    assert_eq!(std::fs::read_to_string(&path).unwrap(), "keep me");
    assert_eq!(std::fs::read_dir(&dir.0).unwrap().count(), 1);
}
#[test]
fn invalid_utf8_and_nonregular_files_are_rejected() {
    let dir = Dir::new();
    let path = dir.0.join("binary.md");
    std::fs::write(&path, [0xff]).unwrap();
    assert!(storage::open(&path).is_err());
    assert!(storage::open(&dir.0).is_err());
}
#[cfg(unix)]
#[test]
fn files_and_folder_enumeration_do_not_follow_symlinks() {
    let dir = Dir::new();
    let path = dir.0.join("real.md");
    std::fs::write(&path, "source").unwrap();
    std::os::unix::fs::symlink(&path, dir.0.join("link.md")).unwrap();
    assert!(storage::open(&dir.0.join("link.md")).is_err());
    assert_eq!(storage::list_folder(&dir.0).unwrap().len(), 1);
}
