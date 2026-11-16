//! The native text-service adapter. UTF-16 belongs at the platform boundary;
//! everything below it uses validated UTF-8 source offsets.
use crate::{
    document::{Document, EditError, Selection, Snapshot, byte_to_utf16, utf16_to_byte},
    projection::Projection,
};
use std::ops::Range;
use unicode_segmentation::UnicodeSegmentation;

pub struct Editor {
    pub document: Document,
    projection: Projection,
    source_mode: bool,
    staged: Option<Stage>,
}

struct Stage {
    original: Selection,
    text: String,
    selection: Selection,
}

impl Editor {
    pub fn new(source: impl Into<String>) -> Self {
        let document = Document::new(source);
        let projection = Projection::new(&document);
        Self {
            document,
            projection,
            source_mode: false,
            staged: None,
        }
    }
    pub fn snapshot(&self) -> Snapshot {
        self.document.snapshot()
    }
    pub fn source_mode(&self) -> bool {
        self.source_mode
    }
    pub fn can_open_slash(&self) -> bool {
        if self.source_mode || self.staged.is_some() || !self.selection().range().is_empty() {
            return false;
        }
        let source = self.document.source();
        let at = self.document.selection().head;
        let start = source[..at].rfind('\n').map_or(0, |i| i + 1);
        if !source[start..at].is_empty() {
            return false;
        }
        let options = pulldown_cmark::Options::ENABLE_YAML_STYLE_METADATA_BLOCKS
            | pulldown_cmark::Options::ENABLE_PLUSES_DELIMITED_METADATA_BLOCKS;
        !pulldown_cmark::Parser::new_ext(source, options)
            .into_offset_iter()
            .any(|(event, range)| {
                matches!(
                    event,
                    pulldown_cmark::Event::Start(
                        pulldown_cmark::Tag::CodeBlock(_) | pulldown_cmark::Tag::MetadataBlock(_)
                    )
                ) && range.start <= at
                    && at < range.end
            })
    }
    pub fn projection(&self) -> &Projection {
        &self.projection
    }
    pub fn committed_display_offset(&self, at: usize, end: bool) -> usize {
        if let Some(stage) = &self.staged {
            let marked = self.marked_range().unwrap();
            let original = stage.original.range();
            if at <= marked.start {
                at
            } else if at >= marked.end {
                at - marked.len() + original.len()
            } else if end {
                original.end
            } else {
                original.start
            }
        } else {
            at
        }
    }
    pub fn heading_at(&self, at: usize) -> u8 {
        if self.source_mode {
            return 0;
        }
        let at = self.committed_display_offset(at, false);
        let lines = self.projection.lines();
        let index = lines
            .partition_point(|l| l.visible.start <= at)
            .saturating_sub(1);
        lines.get(index).map_or(0, |l| l.heading)
    }
    pub fn select_native(&mut self, selection: Selection) -> Result<(), EditError> {
        crate::document::validate_range(&self.text(), &selection.range())?;
        if let Some(marked) = self.marked_range()
            && selection.range().start >= marked.start
            && selection.range().end <= marked.end
        {
            let stage = self.staged.as_mut().unwrap();
            stage.selection = Selection {
                anchor: selection.anchor - marked.start,
                head: selection.head - marked.start,
            };
            self.document
                .update_composition(&stage.text, stage.selection)?;
            return Ok(());
        }
        let original = Selection {
            anchor: self.committed_display_offset(selection.anchor, false),
            head: self.committed_display_offset(selection.head, true),
        };
        self.select(original)
    }
    pub fn committed_text(&self) -> &str {
        if self.source_mode {
            self.document.source()
        } else {
            self.projection.text()
        }
    }
    pub fn text(&self) -> String {
        let mut text = self.committed_text().to_owned();
        if let Some(stage) = &self.staged {
            text.replace_range(stage.original.range(), &stage.text);
        }
        text
    }
    pub fn selection(&self) -> Selection {
        if let Some(stage) = &self.staged {
            let start = stage.original.range().start;
            return Selection {
                anchor: start + stage.selection.anchor,
                head: start + stage.selection.head,
            };
        }
        let source = self.document.selection();
        if self.source_mode {
            source
        } else {
            Selection {
                anchor: self.projection.visible_offset(source.anchor).unwrap_or(0),
                head: self.projection.visible_offset(source.head).unwrap_or(0),
            }
        }
    }
    pub fn marked_range(&self) -> Option<Range<usize>> {
        self.staged
            .as_ref()
            .map(|s| s.original.range().start..s.original.range().start + s.text.len())
    }
    pub fn utf16_range(&self, range: Range<usize>) -> Range<usize> {
        let text = self.text();
        byte_to_utf16(&text, range.start).unwrap_or(0)..byte_to_utf16(&text, range.end).unwrap_or(0)
    }
    pub fn byte_range(&self, range: Range<usize>) -> Range<usize> {
        let text = self.text();
        utf16_to_byte(&text, range.start, false)..utf16_to_byte(&text, range.end, true)
    }
    pub fn select(&mut self, selection: Selection) -> Result<(), EditError> {
        self.cancel_composition();
        self.document.select(if self.source_mode {
            selection
        } else {
            self.projection.source_selection(selection)?
        })
    }
    pub fn replace(&mut self, selection: Selection, text: &str) -> Result<bool, EditError> {
        self.cancel_composition();
        let result = if self.source_mode {
            self.document.select(selection)?;
            self.document.replace_selection(text)
        } else {
            self.projection
                .replace(&mut self.document, selection, text)
                .map_err(|e| match e {
                    crate::projection::ProjectionError::Edit(e) => e,
                    crate::projection::ProjectionError::Stale => EditError::InvalidRange,
                })
        };
        self.reproject();
        result
    }
    pub fn insert(&mut self, text: &str) -> Result<bool, EditError> {
        self.replace(self.selection(), text)
    }
    pub fn set_source_mode(&mut self, source: bool) {
        self.cancel_composition();
        self.source_mode = source;
    }
    pub fn cancel_composition(&mut self) -> bool {
        self.staged.take().is_some() | self.document.cancel_composition()
    }
    pub fn commit_composition(&mut self, text: Option<&str>) -> Result<bool, EditError> {
        let Some(stage) = self.staged.take() else {
            return Ok(false);
        };
        self.document.cancel_composition();
        self.replace(stage.original, text.unwrap_or(&stage.text))
    }
    /// Native ranges refer to the current displayed text, including preedit.
    /// Replacements inside preedit remain staged; expansion also stages the
    /// surrounding committed bytes, leaving save/recovery snapshots untouched.
    pub fn mark(
        &mut self,
        range: Option<Range<usize>>,
        text: &str,
        selection: Option<Range<usize>>,
    ) -> Result<(), EditError> {
        let shown = self.text();
        let range = range.unwrap_or_else(|| {
            self.marked_range()
                .unwrap_or_else(|| self.selection().range())
        });
        crate::document::validate_range(&shown, &range)?;
        let mut original = Selection {
            anchor: range.start,
            head: range.end,
        };
        let mut staged_text = text.to_owned();
        let mut inserted_at = 0;
        if let Some(stage) = &self.staged {
            let marked = self.marked_range().unwrap();
            let start = range.start.min(marked.start);
            let end = range.end.max(marked.end);
            let map = |at: usize, end_affinity: bool| {
                if at <= marked.start {
                    at
                } else if at >= marked.end {
                    at - marked.len() + stage.original.range().len()
                } else if end_affinity {
                    stage.original.range().end
                } else {
                    stage.original.range().start
                }
            };
            original = Selection {
                anchor: map(start, false),
                head: map(end, true),
            };
            staged_text = shown[start..end].to_owned();
            staged_text.replace_range(range.start - start..range.end - start, text);
            inserted_at = range.start - start;
        }
        let local = selection.unwrap_or(text.len()..text.len());
        crate::document::validate_range(text, &local)?;
        let selection = Selection {
            anchor: inserted_at + local.start,
            head: inserted_at + local.end,
        };
        let source = if self.source_mode {
            original
        } else {
            self.projection.source_selection(original)?
        };
        self.document.cancel_composition();
        self.document.begin_composition(source.range())?;
        self.document.update_composition(&staged_text, selection)?;
        self.staged = Some(Stage {
            original,
            text: staged_text,
            selection,
        });
        Ok(())
    }
    pub fn replace_native(
        &mut self,
        range: Option<Range<usize>>,
        text: &str,
    ) -> Result<bool, EditError> {
        if self.staged.is_some() {
            self.mark(range, text, None)?;
            self.commit_composition(None)
        } else {
            let range = range.unwrap_or_else(|| self.selection().range());
            self.replace(
                Selection {
                    anchor: range.start,
                    head: range.end,
                },
                text,
            )
        }
    }
    pub fn undo(&mut self) {
        if !self.cancel_composition() {
            self.document.undo();
        }
        self.reproject();
    }
    pub fn redo(&mut self) {
        if !self.cancel_composition() {
            self.document.redo();
        }
        self.reproject();
    }
    pub fn reproject(&mut self) {
        self.projection = Projection::new(&self.document);
    }
    pub fn horizontal(&mut self, forward: bool, extend: bool) {
        self.cancel_composition();
        let selection = self.selection();
        let text = self.committed_text();
        let next = if !extend && !selection.range().is_empty() {
            if forward {
                selection.range().end
            } else {
                selection.range().start
            }
        } else if forward {
            text.grapheme_indices(true)
                .map(|(i, _)| i)
                .find(|&i| i > selection.head)
                .unwrap_or(text.len())
        } else {
            text.grapheme_indices(true)
                .map(|(i, _)| i)
                .take_while(|&i| i < selection.head)
                .last()
                .unwrap_or(0)
        };
        let _ = self.select(Selection {
            anchor: if extend { selection.anchor } else { next },
            head: next,
        });
    }
    pub fn delete(&mut self, forward: bool) {
        if self.cancel_composition() {
            return;
        }
        if self.selection().range().is_empty() {
            self.horizontal(forward, true);
        }
        let _ = self.insert("");
    }
    pub fn format_inline(&mut self, marker: &str) {
        self.cancel_composition();
        let range = self.document.selection().range();
        let selected = &self.document.source()[range.clone()];
        if selected.len() >= 2 * marker.len()
            && selected.starts_with(marker)
            && selected.ends_with(marker)
        {
            let body = selected[marker.len()..selected.len() - marker.len()].to_owned();
            let selection = Selection {
                anchor: range.start,
                head: range.start + body.len(),
            };
            let _ = self.document.replace_and_select(range, &body, selection);
            self.reproject();
            return;
        }
        let source = self.document.source();
        if !range.is_empty()
            && range.start >= marker.len()
            && source[..range.start].ends_with(marker)
            && source[range.end..].starts_with(marker)
        {
            let body = selected.to_owned();
            let start = range.start - marker.len();
            let end = range.end + marker.len();
            let _ = self.document.replace_and_select(
                start..end,
                &body,
                Selection {
                    anchor: start,
                    head: start + body.len(),
                },
            );
            self.reproject();
            return;
        }
        let replacement = format!("{marker}{selected}{marker}");
        let selection = Selection {
            anchor: range.start + marker.len(),
            head: range.end + marker.len(),
        };
        let _ = self
            .document
            .replace_and_select(range, &replacement, selection);
        self.reproject();
    }
    pub fn block(&mut self, prefix: &str, suffix: &str) {
        self.cancel_composition();
        let source = self.document.source();
        let head = self.document.selection().head;
        let start = source[..head].rfind('\n').map_or(0, |at| at + 1);
        let end = source[head..]
            .find('\n')
            .map_or(source.len(), |at| head + at);
        let body = self
            .projection
            .lines()
            .iter()
            .find(|l| l.source_start == start && l.heading > 0)
            .map_or(&source[start..end], |l| &source[l.content.clone()]);
        let body = if body.starts_with('/') { "" } else { body };
        let replacement = format!("{prefix}{body}{suffix}");
        let _ = self.document.replace_and_select(
            start..end,
            &replacement,
            Selection::caret(start + prefix.len() + body.len()),
        );
        self.reproject();
    }

    pub fn enter(&mut self) -> Result<bool, EditError> {
        self.cancel_composition();
        let source = self.document.source();
        let eol = if source.contains("\r\n") {
            "\r\n"
        } else {
            "\n"
        };
        if !self.selection().range().is_empty() {
            return self.insert(eol);
        }
        let head = self.document.selection().head;
        let line = self
            .projection
            .lines()
            .iter()
            .find(|l| l.source_start <= head && head <= l.source_end)
            .cloned();
        if !self.source_mode
            && let Some(line) = line
            && line.heading > 0
        {
            let tail = &source[head.min(line.content.end)..line.content.end];
            let closing = &source[line.content.end..line.source_end];
            let replacement = format!("{closing}{eol}{tail}");
            let caret = head + closing.len() + eol.len();
            let result = self.document.replace_and_select(
                head..line.source_end,
                &replacement,
                Selection::caret(caret),
            );
            self.reproject();
            return result;
        }
        let start = source[..head].rfind('\n').map_or(0, |i| i + 1);
        let end = source[head..].find('\n').map_or(source.len(), |i| head + i);
        let raw = &source[start..end].trim_end_matches('\r');
        let options = pulldown_cmark::Options::ENABLE_TASKLISTS
            | pulldown_cmark::Options::ENABLE_YAML_STYLE_METADATA_BLOCKS;
        let mut literal = false;
        let mut structured = false;
        for (event, range) in pulldown_cmark::Parser::new_ext(source, options).into_offset_iter() {
            if range.start <= head && head <= range.end {
                match event {
                    pulldown_cmark::Event::Start(
                        pulldown_cmark::Tag::CodeBlock(_) | pulldown_cmark::Tag::MetadataBlock(_),
                    ) => literal = true,
                    pulldown_cmark::Event::Start(
                        pulldown_cmark::Tag::Item | pulldown_cmark::Tag::BlockQuote(_),
                    ) => structured = true,
                    _ => {}
                }
            }
        }
        if structured
            && !literal
            && let Some((prefix, body)) = continuation(raw)
        {
            if body.trim().is_empty() {
                let result = self.document.replace(start..start + raw.len(), "");
                self.reproject();
                return result;
            }
            return self.insert(&format!("{eol}{prefix}"));
        }
        self.insert(eol)
    }
}

fn continuation(line: &str) -> Option<(String, &str)> {
    let trimmed = line.trim_start_matches([' ', '\t']);
    let indent = &line[..line.len() - trimmed.len()];
    let (marker, body) = if let Some(body) = trimmed.strip_prefix("> ") {
        ("> ".to_owned(), body)
    } else if ["- ", "* ", "+ "].iter().any(|p| trimmed.starts_with(p)) {
        (trimmed[..2].to_owned(), &trimmed[2..])
    } else {
        let digits = trimmed.bytes().take_while(u8::is_ascii_digit).count();
        let number = trimmed.get(..digits)?.parse::<u64>().ok()?;
        let rest = trimmed.get(digits..)?;
        if !rest.starts_with(". ") && !rest.starts_with(") ") {
            return None;
        }
        (
            format!("{}{} ", number.checked_add(1)?, &rest[..1]),
            &rest[2..],
        )
    };
    let (task, body) =
        if body.starts_with("[ ] ") || body.starts_with("[x] ") || body.starts_with("[X] ") {
            ("[ ] ", &body[4..])
        } else {
            ("", body)
        };
    Some((format!("{indent}{marker}{task}"), body))
}
