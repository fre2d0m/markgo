//! Parser-confirmed presentation ranges. Source bytes remain authoritative.
use pulldown_cmark::{CodeBlockKind, Event, Options, Parser, Tag, TagEnd};
use std::ops::Range;

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Style {
    pub bold: bool,
    pub italic: bool,
    pub strike: bool,
    pub code: bool,
    pub link: Option<String>,
}
#[derive(Clone, Debug)]
pub struct Inline {
    pub full: Range<usize>,
    pub body: Range<usize>,
    pub style: Style,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub enum Block {
    #[default]
    Paragraph,
    List {
        depth: usize,
        marker: String,
        task: Option<(Range<usize>, bool)>,
    },
    Quote {
        depth: usize,
    },
    Code {
        index: usize,
        language: String,
        first: bool,
    },
}
#[derive(Clone, Debug)]
pub struct Code {
    pub full: Range<usize>,
    pub body: Range<usize>,
    pub opening: Range<usize>,
    pub closing: Range<usize>,
    pub language: String,
    pub delimiter: char,
    pub count: usize,
}
#[derive(Clone, Debug, Default)]
pub struct Syntax {
    pub inlines: Vec<Inline>,
    pub codes: Vec<Code>,
    pub items: Vec<(Range<usize>, usize, String)>,
    pub quotes: Vec<Range<usize>>,
    pub tasks: Vec<(Range<usize>, bool)>,
    item_ends: Vec<usize>,
    quote_ends: Vec<usize>,
}
pub fn options() -> Options {
    Options::ENABLE_STRIKETHROUGH
        | Options::ENABLE_TASKLISTS
        | Options::ENABLE_TABLES
        | Options::ENABLE_YAML_STYLE_METADATA_BLOCKS
        | Options::ENABLE_PLUSES_DELIMITED_METADATA_BLOCKS
}
impl Syntax {
    pub fn parse(source: &str) -> Self {
        let mut result = Self::default();
        let mut lists = Vec::<Option<u64>>::new();
        let mut depth: usize = 0;
        for (event, range) in Parser::new_ext(source, options()).into_offset_iter() {
            match event {
                Event::Start(tag) => {
                    match tag {
                        Tag::List(number) => lists.push(number),
                        Tag::Item => {
                            let level = lists.len().saturating_sub(1);
                            let marker = match lists.last_mut() {
                                Some(Some(n)) => {
                                    let s = format!("{n}.");
                                    *n = n.saturating_add(1);
                                    s
                                }
                                _ => "•".into(),
                            };
                            result.items.push((range, level, marker));
                        }
                        Tag::BlockQuote(_) => result.quotes.push(range),
                        Tag::CodeBlock(CodeBlockKind::Fenced(info)) if depth == 0 => {
                            if let Some(code) = code_range(source, range, info.to_string()) {
                                result.codes.push(code);
                            }
                        }
                        Tag::Strong | Tag::Emphasis | Tag::Strikethrough => {
                            let raw = &source[range.clone()];
                            let count = if matches!(tag, Tag::Emphasis) { 1 } else { 2 };
                            if raw.len() >= 2 * count {
                                result.inlines.push(Inline {
                                    body: range.start + count..range.end - count,
                                    full: range,
                                    style: Style {
                                        bold: matches!(tag, Tag::Strong),
                                        italic: matches!(tag, Tag::Emphasis),
                                        strike: matches!(tag, Tag::Strikethrough),
                                        ..Style::default()
                                    },
                                });
                            }
                        }
                        Tag::Link { dest_url, .. } => {
                            let raw = &source[range.clone()];
                            if let Some(end) = label_end(raw) {
                                result.inlines.push(Inline {
                                    body: range.start + 1..range.start + end,
                                    full: range,
                                    style: Style {
                                        link: Some(dest_url.to_string()),
                                        ..Style::default()
                                    },
                                });
                            }
                        }
                        _ => {}
                    }
                    depth += 1;
                }
                Event::End(tag) => {
                    depth = depth.saturating_sub(1);
                    if matches!(tag, TagEnd::List(_)) {
                        lists.pop();
                    }
                }
                Event::Code(_) => {
                    let raw = &source[range.clone()];
                    let count = raw.bytes().take_while(|b| *b == b'`').count();
                    if count > 0 && raw.len() > count * 2 {
                        result.inlines.push(Inline {
                            body: range.start + count..range.end - count,
                            full: range,
                            style: Style {
                                code: true,
                                ..Style::default()
                            },
                        });
                    }
                }
                Event::TaskListMarker(checked) => result.tasks.push((range, checked)),
                _ => {}
            }
        }
        let mut end = 0;
        result.item_ends = result
            .items
            .iter()
            .map(|(r, _, _)| {
                end = end.max(r.end);
                end
            })
            .collect();
        let mut end = 0;
        result.quote_ends = result
            .quotes
            .iter()
            .map(|r| {
                end = end.max(r.end);
                end
            })
            .collect();
        result
    }
    pub fn block(&self, source: &str, start: usize, end: usize) -> (Block, usize) {
        for (index, code) in self.codes.iter().enumerate() {
            if start >= code.body.start
                && (start < code.closing.start.max(code.full.end) || start == code.body.start)
                && start <= code.body.end
            {
                return (
                    Block::Code {
                        index,
                        language: code.language.clone(),
                        first: start == code.body.start,
                    },
                    start,
                );
            }
        }
        let raw = &source[start..end];
        let first = self.item_ends.partition_point(|at| *at <= start);
        let last = self
            .items
            .partition_point(|(r, _, _)| r.start <= end)
            .max(first);
        if let Some((range, depth, marker)) = self.items[first..last]
            .iter()
            .rev()
            .find(|(r, _, _)| r.start <= end && start < r.end)
        {
            let trimmed = raw.trim_start_matches([' ', '\t']);
            let indent = raw.len() - trimmed.len();
            let first = range.start >= start && range.start <= end;
            let mut offset = start + indent;
            if first {
                let digits = trimmed.bytes().take_while(u8::is_ascii_digit).count();
                let n = if ["- ", "* ", "+ "].iter().any(|p| trimmed.starts_with(p)) {
                    2
                } else if digits > 0
                    && trimmed
                        .get(digits..)
                        .is_some_and(|s| s.starts_with(". ") || s.starts_with(") "))
                {
                    digits + 2
                } else {
                    0
                };
                offset += n;
            }
            let task = self
                .tasks
                .get(self.tasks.partition_point(|(r, _)| r.start < start))
                .filter(|(r, _)| r.end <= end)
                .cloned();
            if let Some((r, _)) = &task {
                offset = r.end;
                while offset < end && source.as_bytes()[offset] == b' ' {
                    offset += 1;
                }
            }
            return (
                Block::List {
                    depth: *depth,
                    marker: if first { marker.clone() } else { String::new() },
                    task,
                },
                offset,
            );
        }
        let first = self.quote_ends.partition_point(|at| *at <= start);
        if self
            .quotes
            .get(first)
            .is_some_and(|r| r.start <= end && start < r.end)
        {
            let mut offset = start;
            let mut depth = 0;
            loop {
                while offset < end && source.as_bytes()[offset] == b' ' {
                    offset += 1;
                }
                if offset >= end || source.as_bytes()[offset] != b'>' {
                    break;
                }
                depth += 1;
                offset += 1;
                if offset < end && source.as_bytes()[offset] == b' ' {
                    offset += 1;
                }
            }
            return (
                Block::Quote {
                    depth: depth.max(1),
                },
                offset,
            );
        }
        (Block::Paragraph, start)
    }
}
fn label_end(raw: &str) -> Option<usize> {
    if !raw.starts_with('[') {
        return None;
    }
    let mut depth = 0;
    let mut escape = false;
    for (i, ch) in raw.char_indices() {
        if escape {
            escape = false;
            continue;
        }
        if ch == '\\' {
            escape = true;
            continue;
        }
        if ch == '[' {
            depth += 1;
        }
        if ch == ']' {
            depth -= 1;
            if depth == 0 {
                return Some(i);
            }
        }
    }
    None
}
fn code_range(source: &str, full: Range<usize>, language: String) -> Option<Code> {
    let end = source[full.start..]
        .find('\n')
        .map(|i| full.start + i + 1)?;
    let opening = full.start..end;
    let raw = source[opening.clone()].trim_start();
    let delimiter = raw.chars().next()?;
    if delimiter != '`' && delimiter != '~' {
        return None;
    }
    let count = raw.chars().take_while(|c| *c == delimiter).count();
    let trimmed_end = source[..full.end].trim_end_matches(['\r', '\n']).len();
    let last_start = source[..trimmed_end]
        .rfind('\n')
        .map_or(end, |i| i + 1)
        .max(end);
    let last = source[last_start..trimmed_end].trim();
    let closed = last.len() >= count && last.chars().all(|c| c == delimiter);
    let closing = if closed {
        last_start..full.end
    } else {
        full.end..full.end
    };
    let body_end = if closed && last_start > end {
        last_start
            - if source[..last_start].ends_with("\r\n") {
                2
            } else {
                1
            }
    } else {
        closing.start
    };
    Some(Code {
        full,
        body: end..body_end.max(end),
        opening,
        closing,
        language,
        delimiter,
        count,
    })
}
