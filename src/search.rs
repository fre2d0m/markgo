use regex::{Regex, RegexBuilder};
use std::{ops::Range, sync::Arc};

#[derive(Clone, Default)]
pub struct Search {
    pub case: bool,
    pub words: bool,
    pub regex: bool,
    pub scoped: bool,
    pub scope: Option<Range<usize>>,
    baseline: Arc<str>,
}
impl Search {
    pub fn clear_scope(&mut self) {
        self.scope = None;
        self.scoped = false;
        self.baseline = Arc::from("");
    }
    pub fn begin(&mut self, source: &str, selection: Range<usize>) {
        self.baseline = Arc::from(source);
        self.scope = (!selection.is_empty()).then_some(selection);
        self.scoped = false;
    }
    pub fn reconcile(&mut self, source: &str) {
        if source == self.baseline.as_ref() {
            return;
        }
        let old = self.baseline.as_ref();
        let mut start = old
            .bytes()
            .zip(source.bytes())
            .take_while(|(a, b)| a == b)
            .count();
        while !old.is_char_boundary(start) || !source.is_char_boundary(start) {
            start -= 1;
        }
        let mut suffix = old[start..]
            .bytes()
            .rev()
            .zip(source[start..].bytes().rev())
            .take_while(|(a, b)| a == b)
            .count();
        while !old.is_char_boundary(old.len() - suffix)
            || !source.is_char_boundary(source.len() - suffix)
        {
            suffix -= 1;
        }
        let old_end = old.len() - suffix;
        let new_end = source.len() - suffix;
        if let Some(scope) = self.scope.as_mut() {
            let delta = new_end as isize - old_end as isize;
            if old_end < scope.start {
                scope.start = scope.start.saturating_add_signed(delta);
                scope.end = scope.end.saturating_add_signed(delta);
            } else if start > scope.end {
            } else if start >= scope.start && old_end <= scope.end {
                scope.end = scope.end.saturating_add_signed(delta);
            } else {
                self.scope = None;
            }
        }
        self.baseline = Arc::from(source);
    }
    fn compiled(&self, query: &str) -> Result<Regex, String> {
        let pattern = if self.regex {
            query.to_owned()
        } else {
            regex::escape(query)
        };
        let pattern = if self.words {
            format!("\\b(?:{pattern})\\b")
        } else {
            pattern
        };
        RegexBuilder::new(&pattern)
            .case_insensitive(!self.case)
            .size_limit(4 * 1024 * 1024)
            .build()
            .map_err(|e| format!("表达式无效：{e}"))
    }
    fn range(&self, source: &str) -> Result<Range<usize>, String> {
        let range = if self.scoped {
            self.scope
                .clone()
                .ok_or("选区范围已失效，请重新选择文字后打开查找")?
        } else {
            0..source.len()
        };
        crate::document::validate_range(source, &range).map_err(|e| e.to_string())?;
        Ok(range)
    }
    pub fn matches(&self, source: &str, query: &str) -> Result<Vec<Range<usize>>, String> {
        if query.is_empty() {
            return Ok(vec![]);
        }
        let range = self.range(source)?;
        let regex = self.compiled(query)?;
        Ok(regex
            .find_iter(&source[range.clone()])
            .take(100_000)
            .map(|m| range.start + m.start()..range.start + m.end())
            .collect())
    }
    pub fn replacement(
        &self,
        source: &str,
        query: &str,
        value: &str,
        only: Option<usize>,
    ) -> Result<Option<(Range<usize>, String)>, String> {
        if query.is_empty() {
            return Ok(None);
        }
        let scope = self.range(source)?;
        let regex = self.compiled(query)?;
        let mut edits = vec![];
        for (i, captures) in regex
            .captures_iter(&source[scope.clone()])
            .take(100_000)
            .enumerate()
        {
            if only.is_some_and(|only| only != i) {
                continue;
            }
            let m = captures.get(0).unwrap();
            let mut value_out = String::new();
            if self.regex {
                captures.expand(value, &mut value_out);
            } else {
                value_out.push_str(value);
            }
            edits.push((scope.start + m.start()..scope.start + m.end(), value_out));
        }
        let Some(first) = edits.first() else {
            return Ok(None);
        };
        let range = first.0.start..edits.last().unwrap().0.end;
        let mut output = source[range.clone()].to_owned();
        for (r, text) in edits.iter().rev() {
            output.replace_range(r.start - range.start..r.end - range.start, text);
        }
        Ok(Some((range, output)))
    }
}
