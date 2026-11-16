//! Deliberately narrow visual projection: top-level ATX headings and paragraphs.
//! Other Markdown remains literal. Parsing discovers ranges; it never serializes.

use crate::document::{Document, EditError, Selection, Snapshot, validate_range};
use pulldown_cmark::{Event, Options, Parser, Tag};
use std::{collections::BTreeMap, ops::Range};

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Line {
    pub source_start: usize,
    pub source_end: usize,
    /// Source bytes actually displayed, excluding heading markers and CRLF.
    pub content: Range<usize>,
    pub visible: Range<usize>,
    pub heading: u8,
}

#[derive(Clone, Debug)]
pub struct Projection {
    snapshot: Snapshot,
    text: String,
    lines: Vec<Line>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ProjectionError {
    Stale,
    Edit(EditError),
}

impl From<EditError> for ProjectionError {
    fn from(error: EditError) -> Self {
        Self::Edit(error)
    }
}

impl Projection {
    pub fn new(document: &Document) -> Self {
        let snapshot = document.snapshot();
        let source = snapshot.source.as_ref();
        let mut headings = BTreeMap::new();
        let mut depth = 0usize;
        let options = Options::ENABLE_YAML_STYLE_METADATA_BLOCKS
            | Options::ENABLE_PLUSES_DELIMITED_METADATA_BLOCKS;
        for (event, range) in Parser::new_ext(source, options).into_offset_iter() {
            match event {
                Event::Start(tag) => {
                    if depth == 0
                        && let Tag::Heading { level, .. } = tag
                    {
                        let line_start = source[..range.start].rfind('\n').map_or(0, |i| i + 1);
                        headings.insert(line_start, level as u8);
                    }
                    depth += 1;
                }
                Event::End(_) => depth -= 1,
                _ => {}
            }
        }
        let mut text = String::with_capacity(source.len());
        let mut lines = Vec::new();
        let mut start = 0;
        for raw in source.split('\n') {
            // Only CR belonging to CRLF is hidden. Lone CR is retained literally.
            let raw_end = start + raw.len();
            let end = raw_end - usize::from(raw_end < source.len() && raw.ends_with('\r'));
            let mut content = start..end;
            let mut heading = 0;
            if let Some(&level) = headings.get(&start)
                && let Some(body) = atx_body(&source[start..end], level)
            {
                content = start + body.start..start + body.end;
                heading = level;
            }
            let visible_start = text.len();
            text.push_str(&source[content.clone()]);
            lines.push(Line {
                source_start: start,
                source_end: end,
                content,
                visible: visible_start..text.len(),
                heading,
            });
            if raw_end < source.len() {
                text.push('\n');
            }
            start = raw_end + 1;
        }
        Self {
            snapshot,
            text,
            lines,
        }
    }

    pub fn text(&self) -> &str {
        &self.text
    }
    pub fn lines(&self) -> &[Line] {
        &self.lines
    }

    pub fn source_offset(&self, visible: usize) -> Result<usize, EditError> {
        validate_range(&self.text, &(visible..visible))?;
        let index = self
            .lines
            .partition_point(|line| line.visible.start <= visible)
            .saturating_sub(1);
        let line = &self.lines[index];
        Ok(line.content.start + (visible - line.visible.start).min(line.content.len()))
    }

    pub fn visible_offset(&self, source: usize) -> Result<usize, EditError> {
        validate_range(&self.snapshot.source, &(source..source))?;
        let index = self
            .lines
            .partition_point(|line| line.source_start <= source)
            .saturating_sub(1);
        let line = &self.lines[index];
        Ok(line.visible.start
            + source
                .saturating_sub(line.content.start)
                .min(line.content.len()))
    }

    pub fn source_selection(&self, visible: Selection) -> Result<Selection, EditError> {
        validate_range(&self.text, &visible.range())?;
        // A visible Select All removes the entire Markdown structure as well.
        if !visible.range().is_empty() && visible.range() == (0..self.text.len()) {
            let end = self.snapshot.source.len();
            return Ok(if visible.anchor <= visible.head {
                Selection {
                    anchor: 0,
                    head: end,
                }
            } else {
                Selection {
                    anchor: end,
                    head: 0,
                }
            });
        }
        Ok(Selection {
            anchor: self.source_offset(visible.anchor)?,
            head: self.source_offset(visible.head)?,
        })
    }

    pub fn replace(
        &self,
        document: &mut Document,
        visible: Selection,
        text: &str,
    ) -> Result<bool, ProjectionError> {
        // A mapping from a previous parse must never edit a newer document.
        if document.revision() != self.snapshot.revision
            || document.source() != self.snapshot.source.as_ref()
        {
            return Err(ProjectionError::Stale);
        }
        let selection = self.source_selection(visible)?;
        document.select(selection)?;
        let mut range = selection.range();
        // Empty ATX headings may have no separator after the opening hashes,
        // or their closing hashes may immediately follow the empty body. Keep
        // new prose inside the heading and retain the existing marker spelling.
        if !text.is_empty()
            && !text.contains(['\r', '\n'])
            && let Some(line) = self
                .lines
                .iter()
                .find(|line| line.heading > 0 && line.content.is_empty() && range == line.content)
        {
            let prefix = if self.snapshot.source[..range.start].ends_with('#')
                && !text.starts_with([' ', '\t'])
            {
                " "
            } else {
                ""
            };
            let suffix = if self.snapshot.source[range.end..line.source_end].starts_with('#')
                && !text.ends_with([' ', '\t'])
            {
                " "
            } else {
                ""
            };
            let replacement = format!("{prefix}{text}{suffix}");
            let caret = range.start + prefix.len() + text.len();
            return Ok(document.replace_and_select(
                range,
                &replacement,
                Selection::caret(caret),
            )?);
        }
        // Crossing into a heading consumes its prefix. Its now-orphaned closing
        // hashes must disappear in the same transaction, retaining the suffix
        // of the heading's text byte for byte.
        if let Some(line) = self.lines.iter().find(|line| {
            line.heading > 0
                && range.start <= line.source_start
                && range.end >= line.content.start
                && range.end <= line.content.end
                && self.snapshot.source[line.content.end..line.source_end].contains('#')
        }) {
            let replacement = format!(
                "{}{}",
                text,
                &self.snapshot.source[range.end..line.content.end]
            );
            let caret = range.start + text.len();
            range.end = line.source_end;
            return Ok(document.replace_and_select(
                range,
                &replacement,
                Selection::caret(caret),
            )?);
        }
        Ok(document.replace_selection(text)?)
    }
}

fn atx_body(line: &str, level: u8) -> Option<Range<usize>> {
    let bytes = line.as_bytes();
    let indent = bytes.iter().take_while(|&&b| b == b' ').count();
    if indent > 3 {
        return None;
    }
    let count = bytes[indent..].iter().take_while(|&&b| b == b'#').count();
    if count != usize::from(level) {
        return None;
    }
    let marker_end = indent + count;
    if marker_end < bytes.len() && !matches!(bytes[marker_end], b' ' | b'\t') {
        return None;
    }
    let padding = bytes[marker_end..]
        .iter()
        .take_while(|&&b| matches!(b, b' ' | b'\t'))
        .count();
    let start = marker_end + padding;
    let end = line.trim_end_matches([' ', '\t']).len().max(start);
    let hashes = bytes[start..end]
        .iter()
        .rev()
        .take_while(|&&b| b == b'#')
        .count();
    let closing = end - hashes;
    let end = if hashes > 0 && (closing == start || matches!(bytes[closing - 1], b' ' | b'\t')) {
        line[start..closing].trim_end_matches([' ', '\t']).len() + start
    } else {
        end
    };
    Some(start..end)
}
