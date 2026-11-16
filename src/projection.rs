//! Source-preserving rich projection. Parsing discovers ranges; it never serializes.

use crate::document::{Document, EditError, Selection, Snapshot, validate_range};
use crate::markdown::{Block, Style, Syntax};
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
    pub block: Block,
}

#[derive(Clone, Debug)]
struct Segment {
    source: Range<usize>,
    visible: Range<usize>,
}
#[derive(Clone, Debug)]
pub struct StyledSpan {
    pub visible: Range<usize>,
    pub style: Style,
}

#[derive(Clone, Debug)]
pub struct Projection {
    snapshot: Snapshot,
    text: String,
    lines: Vec<Line>,
    segments: Vec<Segment>,
    pub syntax: Syntax,
    pub styles: Vec<StyledSpan>,
    style_ends: Vec<usize>,
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
        let syntax = Syntax::parse(source);
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
        let mut segments = Vec::new();
        let mut hidden: Vec<_> = syntax
            .inlines
            .iter()
            .flat_map(|inline| {
                [
                    inline.full.start..inline.body.start,
                    inline.body.end..inline.full.end,
                ]
            })
            .filter(|r| !r.is_empty())
            .collect();
        hidden.sort_by_key(|r| r.start);
        let mut merged: Vec<Range<usize>> = Vec::new();
        for range in hidden {
            if let Some(last) = merged.last_mut().filter(|r| r.end >= range.start) {
                last.end = last.end.max(range.end);
            } else {
                merged.push(range);
            }
        }
        let mut start = 0;
        for raw in source.split('\n') {
            // Only CR belonging to CRLF is hidden. Lone CR is retained literally.
            let raw_end = start + raw.len();
            let end = raw_end - usize::from(raw_end < source.len() && raw.ends_with('\r'));
            if syntax.codes.iter().any(|code| {
                (code.opening.start <= start && start < code.opening.end)
                    || (code.closing.start <= start
                        && start < code.closing.end
                        && !code.body.is_empty())
            }) {
                start = raw_end + 1;
                continue;
            }
            if !lines.is_empty() {
                text.push('\n');
            }
            let mut content = start..end;
            let mut heading = 0;
            let (block, body_start) = syntax.block(source, start, end);
            content.start = body_start;
            if matches!(block, Block::Code { .. })
                && syntax
                    .codes
                    .iter()
                    .any(|c| c.body.is_empty() && c.body.start == start)
            {
                content.end = content.start;
            }
            if let Some(&level) = headings.get(&start)
                && let Some(body) = atx_body(&source[start..end], level)
            {
                content = start + body.start..start + body.end;
                heading = level;
            }
            let visible_start = text.len();
            let first = merged.partition_point(|r| r.end <= content.start);
            let hidden = merged[first..]
                .iter()
                .take_while(|r| r.start < content.end)
                .filter(|_| !matches!(block, Block::Code { .. }))
                .map(|r| r.start.max(content.start)..r.end.min(content.end));
            let mut cursor = content.start;
            for range in hidden {
                if cursor < range.start {
                    let visible = text.len()..text.len() + range.start - cursor;
                    text.push_str(&source[cursor..range.start]);
                    segments.push(Segment {
                        source: cursor..range.start,
                        visible,
                    });
                }
                cursor = cursor.max(range.end);
            }
            if cursor < content.end {
                let visible = text.len()..text.len() + content.end - cursor;
                text.push_str(&source[cursor..content.end]);
                segments.push(Segment {
                    source: cursor..content.end,
                    visible,
                });
            }
            lines.push(Line {
                source_start: start,
                source_end: end,
                content,
                visible: visible_start..text.len(),
                heading,
                block,
            });
            start = raw_end + 1;
        }
        let mut result = Self {
            snapshot,
            text,
            lines,
            segments,
            syntax,
            styles: vec![],
            style_ends: vec![],
        };
        result.styles = result
            .syntax
            .inlines
            .iter()
            .map(|inline| StyledSpan {
                visible: result.visible_offset(inline.body.start).unwrap_or(0)
                    ..result.visible_offset(inline.body.end).unwrap_or(0),
                style: inline.style.clone(),
            })
            .collect();
        result.styles.sort_by_key(|s| s.visible.start);
        let mut end = 0;
        result.style_ends = result
            .styles
            .iter()
            .map(|s| {
                end = end.max(s.visible.end);
                end
            })
            .collect();
        result
    }

    pub fn text(&self) -> &str {
        &self.text
    }
    pub fn lines(&self) -> &[Line] {
        &self.lines
    }
    pub fn styles_for(&self, range: Range<usize>) -> impl Iterator<Item = &StyledSpan> {
        let start = self.style_ends.partition_point(|end| *end <= range.start);
        self.styles[start..]
            .iter()
            .take_while(move |s| s.visible.start < range.end)
    }

    pub fn source_offset(&self, visible: usize) -> Result<usize, EditError> {
        self.source_offset_affinity(visible, false)
    }
    fn source_offset_affinity(&self, visible: usize, left: bool) -> Result<usize, EditError> {
        validate_range(&self.text, &(visible..visible))?;
        let index = self
            .lines
            .partition_point(|line| line.visible.start <= visible)
            .saturating_sub(1);
        let line = &self.lines[index];
        let first = self
            .segments
            .partition_point(|s| s.visible.end < line.visible.start);
        let spans = self.segments[first..]
            .iter()
            .take_while(|s| s.visible.start <= line.visible.end)
            .filter(|s| s.visible.start >= line.visible.start && s.visible.end <= line.visible.end);
        let mut previous = None;
        for segment in spans {
            if segment.visible.start <= visible && visible < segment.visible.end {
                if left
                    && segment.visible.start == visible
                    && let Some(end) = previous
                {
                    return Ok(end);
                }
                return Ok(segment.source.start + visible - segment.visible.start);
            }
            if segment.visible.end == visible {
                previous = Some(segment.source.end);
            }
        }
        Ok(previous.unwrap_or(line.content.start))
    }

    pub fn visible_offset(&self, source: usize) -> Result<usize, EditError> {
        validate_range(&self.snapshot.source, &(source..source))?;
        let index = self
            .lines
            .partition_point(|line| line.source_start <= source)
            .saturating_sub(1);
        let line = &self.lines[index];
        let first = self
            .segments
            .partition_point(|s| s.visible.end < line.visible.start);
        for segment in self.segments[first..]
            .iter()
            .take_while(|s| s.visible.start <= line.visible.end)
            .filter(|s| s.visible.start >= line.visible.start && s.visible.end <= line.visible.end)
        {
            if source <= segment.source.start {
                return Ok(segment.visible.start);
            }
            if source <= segment.source.end {
                return Ok(segment.visible.start + source - segment.source.start);
            }
        }
        Ok(line.visible.end)
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
            anchor: self.source_offset_affinity(visible.anchor, visible.anchor > visible.head)?,
            head: self.source_offset_affinity(visible.head, visible.head > visible.anchor)?,
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
        if let Some(code) = self
            .syntax
            .codes
            .iter()
            .find(|code| code.body.start <= range.start && range.end <= code.body.end)
        {
            let source = self.snapshot.source.as_ref();
            let ending = if source[code.opening.clone()].ends_with("\r\n") {
                "\r\n"
            } else {
                "\n"
            };
            let body = format!(
                "{}{}{}",
                &source[code.body.start..range.start],
                text,
                &source[range.end..code.body.end]
            );
            // A literal fence entered into code must stay code. Extend the paired
            // delimiters, preserving all info text, whitespace and line endings.
            let longest = body
                .lines()
                .filter_map(|line| {
                    let trimmed = line.trim_start_matches(' ');
                    if line.len() - trimmed.len() > 3 {
                        return None;
                    }
                    let n = trimmed
                        .chars()
                        .take_while(|ch| *ch == code.delimiter)
                        .count();
                    (n >= code.count && trimmed[n..].trim().is_empty()).then_some(n)
                })
                .max()
                .unwrap_or(code.count.saturating_sub(1));
            let count = code.count.max(longest + 1);
            let opening = &source[code.opening.clone()];
            let offset = opening.find(code.delimiter).unwrap_or(0);
            let mut replacement = format!(
                "{}{}{}",
                &opening[..offset],
                code.delimiter.to_string().repeat(count),
                &opening[offset + code.count..]
            );
            let caret =
                code.full.start + replacement.len() + range.start - code.body.start + text.len();
            replacement.push_str(&body);
            let tail = &source[code.body.end..code.full.end];
            if code.body.is_empty()
                && !body.is_empty()
                && !code.closing.is_empty()
                && !body.ends_with('\n')
            {
                replacement.push_str(ending);
            }
            if !code.closing.is_empty() && count > code.count {
                let relative = code.closing.start - code.body.end;
                let at = relative + tail[relative..].find(code.delimiter).unwrap_or(0);
                let existing = tail[at..]
                    .chars()
                    .take_while(|ch| *ch == code.delimiter)
                    .count();
                replacement.push_str(&tail[..at]);
                replacement.push_str(&code.delimiter.to_string().repeat(count.max(existing)));
                replacement.push_str(&tail[at + existing..]);
            } else {
                replacement.push_str(tail);
            }
            return Ok(document.replace_and_select(
                code.full.clone(),
                &replacement,
                Selection::caret(caret),
            )?);
        }
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
        let mut repairs = Vec::<Range<usize>>::new();
        if !range.is_empty() {
            for inline in &self.syntax.inlines {
                let open = inline.full.start..inline.body.start;
                let close = inline.body.end..inline.full.end;
                let crosses_open = range.start < open.end && open.start < range.end;
                let crosses_close = range.start < close.end && close.start < range.end;
                let whole_body = range.start <= inline.body.start && range.end >= inline.body.end;
                if crosses_open != crosses_close || whole_body {
                    repairs.push(open);
                    repairs.push(close);
                }
            }
            for code in &self.syntax.codes {
                let crosses_open = range.start < code.opening.end && code.opening.start < range.end;
                let crosses_close =
                    range.start < code.closing.end && code.closing.start < range.end;
                if crosses_open != crosses_close {
                    repairs.push(code.opening.clone());
                    repairs.push(code.closing.clone());
                }
            }
        }
        let start = repairs.iter().fold(range.start, |at, r| at.min(r.start));
        let end = repairs.iter().fold(range.end, |at, r| at.max(r.end));
        let source = document.source();
        let mut replacement = String::new();
        let mut inserted = false;
        let mut caret = start;
        for (offset, ch) in source[start..end].char_indices() {
            let at = start + offset;
            if !inserted && at >= range.start {
                replacement.push_str(text);
                caret = start + replacement.len();
                inserted = true;
            }
            if !(range.contains(&at) || repairs.iter().any(|r| r.contains(&at))) {
                replacement.push(ch);
            }
        }
        if !inserted {
            replacement.push_str(text);
            caret = start + replacement.len();
        }
        Ok(document.replace_and_select(start..end, &replacement, Selection::caret(caret))?)
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
    // A bare run of hashes is still being typed. Convert to an empty visual
    // heading only once its separating space/tab has actually been entered.
    if marker_end == bytes.len() {
        return None;
    }
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
