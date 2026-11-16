use markgo::{
    document::{Document, EditError, Selection, byte_to_utf16, utf16_to_byte},
    projection::{Projection, ProjectionError},
};

const FIXTURE: &str = "---\n# metadata: literal\n---\n\n# 标题 ###\n\n正文🙂\n\n```go\n# not a heading\nfmt.Println(1)\n```\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n[link][ref]\n\n[ref]: <https://example.test> \"title\"\n\nSetext\n======\n";

#[test]
fn visual_edit_retains_untouched_source_bytes_and_line_endings() {
    for ending in ["\n", "\r\n"] {
        let original = FIXTURE.replace('\n', ending);
        let mut doc = Document::new(&original);
        let view = Projection::new(&doc);
        assert!(view.text().contains("# metadata: literal"));
        assert!(view.text().contains("# not a heading"));
        assert!(view.text().contains("Setext\n======"));
        assert!(!view.text().contains("# 标题"));
        assert!(!view.text().contains("标题 ###"));
        let at = view.text().find("正文").unwrap() + "正文".len();
        view.replace(&mut doc, Selection::caret(at), "，继续")
            .unwrap();
        assert_eq!(doc.source(), original.replacen("正文", "正文，继续", 1));
        assert!(doc.undo());
        assert_eq!(doc.source(), original);
        assert!(doc.redo());
        assert_eq!(doc.source(), original.replacen("正文", "正文，继续", 1));
    }
}

#[test]
fn cross_block_replacement_is_one_undo_and_retains_reversed_selection() {
    for (anchor, head) in [(1, 9), (9, 1)] {
        let original = "# One\n\n## Two\n\nlast";
        let mut doc = Document::new(original);
        let view = Projection::new(&doc);
        let selection = Selection { anchor, head };
        let mapped = view.source_selection(selection).unwrap();
        view.replace(&mut doc, selection, "中🙂").unwrap();
        assert_eq!(doc.source(), "# O中🙂\nlast");
        assert!(doc.undo());
        assert_eq!(doc.source(), original);
        assert_eq!(doc.selection(), mapped);
        assert!(!doc.can_undo());
    }
}

#[test]
fn composition_never_enters_save_snapshot_until_one_commit() {
    let mut doc = Document::new("# 标题\n\n原文🙂\r\n结尾");
    let start = doc.source().find("原文").unwrap();
    let end = start + "原文🙂".len();
    let selection = Selection {
        anchor: end,
        head: start,
    };
    doc.select(selection).unwrap();
    let original = doc.snapshot();
    doc.begin_composition(selection.range()).unwrap();
    for candidate in ["n", "ni", "nihao", "你好🙂"] {
        doc.update_composition(candidate, Selection::caret(candidate.len()))
            .unwrap();
        assert_eq!(doc.snapshot(), original);
        assert!(!doc.can_undo());
    }
    assert_eq!(doc.replace(0..0, "bad"), Err(EditError::CompositionActive));
    doc.commit_composition("你好").unwrap();
    assert_eq!(doc.source(), "# 标题\n\n你好\r\n结尾");
    assert_eq!(doc.revision(), 1);
    assert!(doc.undo());
    assert_eq!(doc.source(), original.source.as_ref());
    assert_eq!(doc.selection(), selection);
    assert!(!doc.can_undo());
}

#[test]
fn cancelling_composition_restores_selection_and_preserves_redo() {
    let mut doc = Document::new("正文");
    doc.replace(0..0, "前言").unwrap();
    doc.undo();
    let original = doc.snapshot();
    let selection = doc.selection();
    doc.begin_composition(0.."正文".len()).unwrap();
    doc.update_composition("候选", Selection::caret("候选".len()))
        .unwrap();
    assert!(doc.undo()); // Cancel preedit; do not consume source history.
    assert_eq!(doc.snapshot(), original);
    assert_eq!(doc.selection(), selection);
    assert!(doc.can_redo());
    doc.redo();
    assert_eq!(doc.source(), "前言正文");
}

#[test]
fn stale_projection_cannot_edit_another_revision() {
    let mut doc = Document::new("# 标题");
    let view = Projection::new(&doc);
    doc.replace(doc.source().len()..doc.source().len(), "正文")
        .unwrap();
    let snapshot = doc.snapshot();
    assert_eq!(
        view.replace(&mut doc, Selection::caret(0), "bad"),
        Err(ProjectionError::Stale)
    );
    assert_eq!(doc.snapshot(), snapshot);
    doc.undo(); // Equal bytes are not enough: revision is different.
    assert_eq!(
        view.replace(&mut doc, Selection::caret(0), "bad"),
        Err(ProjectionError::Stale)
    );
}

#[test]
fn utf16_offsets_and_invalid_ranges_never_split_unicode() {
    let text = "中🙂e\u{301}";
    for (byte, _) in text.char_indices() {
        let units = byte_to_utf16(text, byte).unwrap();
        assert_eq!(utf16_to_byte(text, units, false), byte);
    }
    assert_eq!(utf16_to_byte(text, 2, false), "中".len());
    assert_eq!(utf16_to_byte(text, 2, true), "中🙂".len());
    assert_eq!(utf16_to_byte(text, usize::MAX, true), text.len());
    let mut doc = Document::new(text);
    for range in [1..2, 0..999, std::ops::Range { start: 4, end: 3 }] {
        assert_eq!(doc.replace(range, "bad"), Err(EditError::InvalidRange));
    }
    assert_eq!(doc.source(), text);
    assert!(!doc.can_undo());
}

#[test]
fn every_visible_character_roundtrips_through_compact_line_maps() {
    for source in [
        "",
        "#",
        "## 标题 ###\r\n\r\n中🙂\r\n",
        FIXTURE,
        "   ### 多行\n> # 引用\n- # 列表\n",
    ] {
        let doc = Document::new(source);
        let view = Projection::new(&doc);
        for at in view
            .text()
            .char_indices()
            .map(|(at, _)| at)
            .chain([view.text().len()])
        {
            assert_eq!(
                view.visible_offset(view.source_offset(at).unwrap())
                    .unwrap(),
                at,
                "{source:?} at {at}"
            );
        }
    }
}

#[test]
fn closing_heading_markers_and_mixed_endings_stay_intact() {
    let source = "# 标题 ###\r\n下一行\n最后\r\n";
    let mut doc = Document::new(source);
    let view = Projection::new(&doc);
    view.replace(&mut doc, Selection::caret("标题".len()), "新")
        .unwrap();
    assert_eq!(doc.source(), "# 标题新 ###\r\n下一行\n最后\r\n");
    doc.undo();
    assert_eq!(doc.source(), source);
}

#[test]
fn select_all_removes_hidden_structure_in_one_transaction() {
    let mut doc = Document::new("# One\n\n## Two ###");
    let view = Projection::new(&doc);
    view.replace(
        &mut doc,
        Selection {
            anchor: view.text().len(),
            head: 0,
        },
        "",
    )
    .unwrap();
    assert_eq!(doc.source(), "");
    doc.undo();
    assert_eq!(doc.source(), "# One\n\n## Two ###");
}

#[test]
fn no_op_does_not_create_history_or_invalidate_redo() {
    let mut doc = Document::new("原文");
    doc.replace(0..0, "前言").unwrap();
    doc.undo();
    let revision = doc.revision();
    assert!(!doc.replace(0..doc.source().len(), "原文").unwrap());
    assert_eq!(doc.revision(), revision);
    assert!(doc.can_redo());
}

#[test]
fn deleting_a_visible_newline_consumes_the_next_heading_prefix() {
    let mut doc = Document::new("one\r\n## two ###");
    let view = Projection::new(&doc);
    view.replace(&mut doc, Selection { anchor: 3, head: 4 }, "")
        .unwrap();
    assert_eq!(doc.source(), "onetwo");
    assert_eq!(doc.selection(), Selection::caret(3));
    doc.undo();
    assert_eq!(doc.source(), "one\r\n## two ###");
    doc.redo();
    assert_eq!(doc.source(), "onetwo");
    assert_eq!(doc.selection(), Selection::caret(3));
}

#[test]
fn parser_context_prevents_hiding_literal_hashes() {
    for source in [
        "---\n# metadata\n---\n",
        "    # indented code\n",
        "<!--\n# comment\n-->\n",
        "#not-heading\n",
        "Setext\n======\n",
    ] {
        let doc = Document::new(source);
        let view = Projection::new(&doc);
        assert_eq!(view.text(), source);
        assert!(view.lines().iter().all(|line| line.heading == 0));
    }
    let view = Projection::new(&Document::new("   ### 标题 ###\n"));
    assert_eq!(view.text(), "标题\n");
    assert_eq!(view.lines()[0].heading, 3);
    for (source, expected) in [
        ("```md\n# code\n```\n", "# code\n"),
        ("~~~\n# unclosed code\n", "# unclosed code\n"),
        ("> # quote\n", "# quote\n"),
        ("- # list\n", "# list\n"),
    ] {
        let view = Projection::new(&Document::new(source));
        assert_eq!(view.text(), expected);
        assert!(view.lines().iter().all(|l| l.heading == 0));
    }
}

#[test]
fn failed_composition_updates_are_atomic_and_do_not_change_committed_text() {
    let mut doc = Document::new("原文");
    assert_eq!(doc.begin_composition(1..2), Err(EditError::InvalidRange));
    assert!(doc.composition().is_none());
    doc.begin_composition(0..doc.source().len()).unwrap();
    doc.update_composition("你好", Selection::caret("你好".len()))
        .unwrap();
    let before = doc.composition().cloned();
    assert_eq!(
        doc.update_composition("🙂", Selection::caret(1)),
        Err(EditError::InvalidRange)
    );
    assert_eq!(doc.composition(), before.as_ref());
    assert_eq!(doc.source(), "原文");
    assert_eq!(doc.revision(), 0);
    doc.cancel_composition();
    assert_eq!(
        doc.commit_composition("不应插入"),
        Err(EditError::NoComposition)
    );
}

#[test]
fn separate_documents_retain_independent_history_selection_and_composition() {
    let mut first = Document::new("# 第一篇");
    let mut second = Document::new("## 第二篇");
    first
        .replace(first.source().len()..first.source().len(), "正文")
        .unwrap();
    first.begin_composition(first.selection().range()).unwrap();
    first
        .update_composition("候选", Selection::caret("候选".len()))
        .unwrap();
    let snapshot = first.snapshot();
    let selection = first.selection();
    second
        .replace(0..second.source().len(), "第二篇已修改")
        .unwrap();
    assert_eq!(first.snapshot(), snapshot);
    assert_eq!(first.selection(), selection);
    assert_eq!(first.composition().unwrap().text, "候选");
    second.undo();
    assert_eq!(second.source(), "## 第二篇");
    first.cancel_composition();
    first.undo();
    assert_eq!(first.source(), "# 第一篇");
}

#[test]
fn foreign_projection_with_same_revision_is_rejected() {
    let first = Document::new("# First");
    let mut second = Document::new("# Second");
    let view = Projection::new(&first);
    assert_eq!(
        view.replace(&mut second, Selection::caret(0), "bad"),
        Err(ProjectionError::Stale)
    );
    assert_eq!(second.source(), "# Second");
    assert_eq!(second.revision(), 0);
}

#[test]
fn invalid_post_edit_selection_does_not_partially_apply_a_transaction() {
    let mut doc = Document::new("正文");
    let before = doc.snapshot();
    for after in [Selection::caret(1), Selection::caret(999)] {
        assert_eq!(
            doc.replace_and_select(0..doc.source().len(), "🙂", after),
            Err(EditError::InvalidRange)
        );
        assert_eq!(doc.snapshot(), before);
        assert!(!doc.can_undo());
    }
}

#[test]
fn typing_into_empty_headings_keeps_both_marker_boundaries() {
    for (source, expected) in [
        ("# ", "# 字"),
        ("## ", "## 字"),
        ("# ###", "# 字 ###"),
        ("## ###", "## 字 ###"),
    ] {
        let mut doc = Document::new(source);
        let view = Projection::new(&doc);
        assert_eq!(view.text(), "");
        view.replace(&mut doc, Selection::caret(0), "字").unwrap();
        let selection = doc.selection();
        assert_eq!(doc.source(), expected);
        assert_eq!(Projection::new(&doc).text(), "字");
        doc.undo();
        assert_eq!(doc.source(), source);
        doc.redo();
        assert_eq!(doc.source(), expected);
        assert_eq!(doc.selection(), selection);
    }
}
