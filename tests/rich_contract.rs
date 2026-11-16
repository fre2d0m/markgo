use markgo::{
    document::Selection,
    editing::Editor,
    markdown::Block,
    search::Search,
    session::{Draft, Journal, Session, restore},
};
use std::{
    path::PathBuf,
    sync::atomic::{AtomicU64, Ordering},
};
static ID: AtomicU64 = AtomicU64::new(0);
struct Dir(PathBuf);
impl Dir {
    fn new() -> Self {
        let p = std::env::temp_dir().join(format!(
            "markgo-rust-contract-{}-{}",
            std::process::id(),
            ID.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir_all(&p).unwrap();
        Self(p)
    }
}
impl Drop for Dir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

#[test]
fn rich_ranges_preserve_source_and_unicode_mappings() {
    let source = "# __标题__ ###\r\n\r\na **bold _中文🙂_** ~~gone~~ `a+b` [link][id]\r\n\r\n- [ ] task\r\n  - nested\r\n\r\n> quote\r\n\r\n```rust\r\nlet x = 1;\r\n```\r\n\r\n[id]: <note.md> \"title\"\r\n";
    let mut e = Editor::new(source);
    let text = e.text();
    assert!(text.contains("标题"));
    assert!(!text.contains("__标题__"));
    assert!(text.contains("a bold 中文🙂 gone a+b link"));
    assert!(text.contains("task\nnested"));
    assert!(!text.contains("```"));
    assert!(text.contains("[id]: <note.md> \"title\""));
    for at in text.char_indices().map(|(i, _)| i).chain([text.len()]) {
        assert_eq!(
            e.projection()
                .visible_offset(e.projection().source_offset(at).unwrap())
                .unwrap(),
            at
        );
    }
    let at = text.find("中文").unwrap();
    e.replace(Selection::caret(at), "新").unwrap();
    assert_eq!(e.document.source(), source.replacen("中文", "新中文", 1));
    e.undo();
    assert_eq!(e.document.source(), source);
}
#[test]
fn inline_cross_boundary_edits_repair_pairs_and_undo_once() {
    for (source, start, end, expected) in [
        ("a **bold** z", 0, 4, "!ld z"),
        ("a **bold** z", 4, 8, "a bo!"),
        ("a [text](target) z", 4, 8, "a te!"),
    ] {
        let mut e = Editor::new(source);
        e.replace(
            Selection {
                anchor: end,
                head: start,
            },
            "!",
        )
        .unwrap();
        assert_eq!(e.document.source(), expected);
        e.undo();
        assert_eq!(e.document.source(), source);
        assert!(!e.document.can_undo());
    }
}
#[test]
fn tasks_numbering_and_indentation_share_history() {
    let mut e = Editor::new("3. one\n9. two\n   - [X] 中文");
    let markers = e
        .projection()
        .lines()
        .iter()
        .filter_map(|l| {
            if let Block::List { marker, .. } = &l.block {
                Some(marker.as_str())
            } else {
                None
            }
        })
        .collect::<Vec<_>>();
    assert_eq!(markers, vec!["3.", "4.", "•"]);
    let line = e.projection().lines().last().unwrap();
    let task = if let Block::List {
        task: Some((r, _)), ..
    } = &line.block
    {
        r.clone()
    } else {
        panic!("task")
    };
    e.select(Selection::caret(e.text().len())).unwrap();
    let selection = e.document.selection();
    e.toggle_task(task).unwrap();
    assert!(e.document.source().contains("[ ]"));
    assert_eq!(e.document.selection(), selection);
    e.undo();
    assert!(e.document.source().contains("[X]"));
    e.indent(false);
    assert!(e.document.source().ends_with("     - [X] 中文"));
    e.undo();
    assert_eq!(e.document.source(), "3. one\n9. two\n   - [X] 中文");
}
#[test]
fn empty_code_has_an_editable_caret_and_language_changes_are_undoable() {
    let mut e = Editor::new("```rust\n```\n");
    assert_eq!(e.text(), "\n");
    assert!(matches!(e.block_at(0), Block::Code { .. }));
    e.insert("let answer = 42;").unwrap();
    assert_eq!(e.document.source(), "```rust\nlet answer = 42;\n```\n");
    e.code_language(0, "python").unwrap();
    assert!(e.document.source().starts_with("```python\n"));
    e.undo();
    assert!(e.document.source().starts_with("```rust\n"));
    e.continue_code(0);
    assert!(e.document.source().ends_with("```\n\n"));
    e.undo();
    e.undo();
    assert_eq!(e.document.source(), "```rust\n```\n");
}
#[test]
fn search_capture_replacement_scope_and_invalid_patterns() {
    let source = "A1 a2 outside a3";
    let mut s = Search::default();
    s.regex = true;
    s.begin(source, 0..5);
    s.scoped = true;
    assert_eq!(s.matches(source, "a(\\d)").unwrap(), vec![0..2, 3..5]);
    let (range, text) = s
        .replacement(source, "a(\\d)", "b$1", None)
        .unwrap()
        .unwrap();
    assert_eq!(range, 0..5);
    assert_eq!(text, "b1 b2");
    assert!(s.matches(source, "[").is_err());
    s.reconcile("A1 new a2 outside a3");
    assert_eq!(s.scope, Some(0..9));
    s.reconcile("replacement");
    assert!(s.scope.is_none());
    assert!(s.matches("replacement", ".").is_err());
}
#[test]
fn journal_round_trip_conflicts_and_corrupt_files_never_touch_documents() {
    let dir = Dir::new();
    let path = dir.0.join("session.json");
    let file = dir.0.join("article.md");
    std::fs::write(&file, "original").unwrap();
    let (journal, _) = Journal::load(path.clone()).unwrap();
    let state = Session {
        documents: vec![Draft {
            id: "1".into(),
            name: "article.md".into(),
            path: file.to_string_lossy().into_owned(),
            dirty: true,
            source: "draft 中文".into(),
            original: "original".into(),
            head: 6,
            anchor: 6,
            ..Draft::default()
        }],
        ..Session::default()
    };
    let journal = journal.write(state.clone()).map_err(|(_, e)| e).unwrap();
    assert_eq!(std::fs::read_to_string(&file).unwrap(), "original");
    let (_, loaded) = Journal::load(path.clone()).unwrap();
    assert_eq!(loaded.documents[0].source, "draft 中文");
    std::fs::write(&file, "external").unwrap();
    let recovered = restore(loaded.documents[0].clone());
    assert!(recovered.conflict);
    assert!(recovered.file.is_none());
    assert_eq!(recovered.draft.source, "draft 中文");
    std::fs::write(&path, "unsupported").unwrap();
    assert!(journal.write(state).is_err());
    assert_eq!(std::fs::read_to_string(&path).unwrap(), "unsupported");
    assert!(Journal::load(path).is_err());
}
#[test]
fn previous_journal_schema_remains_readable_without_an_old_runtime() {
    let s=Session::parse(r##"{"schema":1,"documents":[{"id":"a","name":"Draft","dirty":true,"source":"字","head":3,"anchor":3}],"sourceMode":true,"sidebar":false}"##).unwrap();
    assert_eq!(s.schema, 2);
    assert!(s.documents[0].source_mode);
    assert!(!s.sidebar);
    assert!(Session::parse(r#"{"schema":99,"documents":[]}"#).is_err());
}

#[test]
fn literal_fences_extend_pairs_and_keep_caret_and_original_bytes() {
    for (source, insert, expected) in [
        (
            "before\n\n```rust\nbody\n```\n\nafter",
            "\n```\n",
            "before\n\n````rust\nbody\n```\n\n````\n\nafter",
        ),
        (
            "~~~text\r\n~~~\r\n",
            "~~~~",
            "~~~~~text\r\n~~~~\r\n~~~~~\r\n",
        ),
        ("```\na", "\n```", "````\na\n```"),
    ] {
        let mut e = Editor::new(source);
        let at = e.projection().syntax.codes[0].body.end;
        let visible = e.projection().visible_offset(at).unwrap();
        e.select(Selection::caret(visible)).unwrap();
        e.insert(insert).unwrap();
        assert_eq!(e.document.source(), expected);
        assert!(e.document.source()[..e.document.selection().head].ends_with(insert));
        e.undo();
        assert_eq!(e.document.source(), source);
        assert!(!e.document.can_undo());
    }
}

#[test]
fn structural_enter_and_block_conversion_preserve_context() {
    let mut e = Editor::new("- parent\n  - ");
    e.select(Selection::caret(e.text().len())).unwrap();
    e.enter().unwrap();
    assert_eq!(e.document.source(), "- parent\n- ");
    e.undo();
    let mut e = Editor::new("```rust\n    let n = 1;\n```\n");
    let at = e.projection().syntax.codes[0].body.end;
    e.select(Selection::caret(e.projection().visible_offset(at).unwrap()))
        .unwrap();
    e.enter().unwrap();
    assert_eq!(e.document.source(), "```rust\n    let n = 1;\n    \n```\n");
    e.undo();
    let mut e = Editor::new("- [ ] task\n");
    e.block("## ", "");
    assert_eq!(e.document.source(), "## task\n");
    e.undo();
    assert_eq!(e.document.source(), "- [ ] task\n");
}

#[test]
fn out_of_order_checkpoints_cannot_resurrect_discarded_drafts() {
    let dir = Dir::new();
    let path = dir.0.join("session.json");
    let (journal, _) = Journal::load(path.clone()).unwrap();
    let mut ordered = markgo::session::OrderedJournal::new(journal);
    ordered.write(2, Session::default()).unwrap();
    let obsolete = Session {
        documents: vec![Draft {
            id: "old".into(),
            name: "draft".into(),
            dirty: true,
            source: "discarded".into(),
            ..Draft::default()
        }],
        ..Session::default()
    };
    ordered.write(1, obsolete).unwrap();
    assert!(Journal::load(path).unwrap().1.documents.is_empty());
}

#[test]
fn missing_clean_file_keeps_checkpoint_text_without_overwrite_permission() {
    let dir = Dir::new();
    let path = dir.0.join("missing.md");
    let recovered = restore(Draft {
        id: "1".into(),
        name: "missing.md".into(),
        path: path.to_string_lossy().into_owned(),
        source: "last known text".into(),
        original: "last known text".into(),
        ..Draft::default()
    });
    assert!(recovered.conflict);
    assert!(recovered.file.is_none());
    assert_eq!(recovered.draft.source, "last known text");
    assert!(!path.exists());
}

#[test]
fn dense_document_projection_matches_fresh_parse_after_edits() {
    let source = "段落 **bold 中文** _italic_ ~~strike~~ [link](note.md) `code`\n\n".repeat(1600);
    let mut e = Editor::new(source.clone());
    for needle in [0, e.text().len() / 2, e.text().len()] {
        let mut at = needle.min(e.text().len());
        while !e.text().is_char_boundary(at) {
            at -= 1;
        }
        e.replace(Selection::caret(at), "新增").unwrap();
        let fresh = Editor::new(e.document.source());
        assert_eq!(e.text(), fresh.text());
        assert_eq!(e.projection().lines(), fresh.projection().lines());
        e.undo();
        assert_eq!(e.document.source(), source);
    }
}

#[test]
fn migration_reads_previous_settings_and_drafts_without_overwriting_them() {
    let dir = Dir::new();
    let old = dir.0.join("session.json");
    let new = dir.0.join("native/session.json");
    let original = r#"{"schema":1,"documents":[{"id":"a","name":"Draft","source":"未完成","dirty":true}],"recent":null}"#;
    std::fs::write(&old, original).unwrap();
    let (journal, state) = Journal::load_with_fallback(new.clone(), Some(old.clone())).unwrap();
    assert_eq!(state.documents[0].source, "未完成");
    let _ = journal.write(state).map_err(|(_, e)| e).unwrap();
    assert_eq!(std::fs::read_to_string(&old).unwrap(), original);
    std::fs::write(&old, "now invalid").unwrap();
    assert!(
        Journal::load_with_fallback(new, Some(old)).is_ok(),
        "new journal takes precedence"
    );
    let old = dir.0.join("appearance.json");
    let new = dir.0.join("native/appearance.conf");
    let original = r#"{"theme":"sakura","fontSize":20,"width":840,"serif":true}"#;
    std::fs::write(&old, original).unwrap();
    let (store, prefs) =
        markgo::preferences::Store::load_with_fallback(new, Some(old.clone())).unwrap();
    assert_eq!(prefs.theme, "sakura");
    assert!(prefs.serif);
    assert_eq!(prefs.column, 840);
    store.write(prefs).unwrap();
    assert_eq!(std::fs::read_to_string(&old).unwrap(), original);
}

#[test]
fn corrupt_previous_journal_does_not_create_a_new_empty_journal() {
    let dir = Dir::new();
    let old = dir.0.join("old.json");
    let new = dir.0.join("native/session.json");
    std::fs::write(&old, r#"{"schema":99,"documents":[]}"#).unwrap();
    assert!(Journal::load_with_fallback(new.clone(), Some(old.clone())).is_err());
    assert!(!new.exists());
    assert!(old.exists());
}

#[test]
fn block_conversion_and_indentation_preserve_crlf_and_selection_endpoints() {
    let mut e = Editor::new("- [ ] 第一行\r\n第二行\r\n");
    e.block("## ", "");
    assert_eq!(e.document.source(), "## 第一行\r\n第二行\r\n");
    e.undo();
    let mut e = Editor::new("one\ntwo\nthree");
    e.document.select(Selection { anchor: 4, head: 0 }).unwrap();
    e.indent(false);
    assert_eq!(e.document.source(), "  one\ntwo\nthree");
    assert_eq!(e.document.selection(), Selection { anchor: 6, head: 2 });
    e.undo();
    assert_eq!(e.document.selection(), Selection { anchor: 4, head: 0 });
    let mut e = Editor::new("> > ");
    e.select(Selection::caret(0)).unwrap();
    e.enter().unwrap();
    assert_eq!(e.document.source(), "> ");
}
