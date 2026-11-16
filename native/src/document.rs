use std::{fmt, ops::Range, sync::Arc};

/// UTF-8 byte offsets into the committed source, preserving selection direction.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Selection {
    pub anchor: usize,
    pub head: usize,
}

impl Selection {
    pub fn caret(at: usize) -> Self {
        Self {
            anchor: at,
            head: at,
        }
    }

    pub fn range(self) -> Range<usize> {
        self.anchor.min(self.head)..self.anchor.max(self.head)
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Snapshot {
    pub source: Arc<str>,
    pub revision: u64,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EditError {
    InvalidRange,
    CompositionActive,
    NoComposition,
}

impl fmt::Display for EditError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::InvalidRange => "range must be ordered and lie on UTF-8 boundaries",
            Self::CompositionActive => "finish or cancel composition before another edit",
            Self::NoComposition => "there is no active composition",
        })
    }
}

impl std::error::Error for EditError {}

#[derive(Clone, Debug)]
struct Change {
    start: usize,
    removed: String,
    inserted: String,
    before: Selection,
    after: Selection,
}

/// Preedit is deliberately separate from source and undo history.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Composition {
    pub range: Range<usize>,
    pub text: String,
    /// Selection in the staged text, in UTF-8 bytes.
    pub selection: Selection,
}

#[derive(Debug)]
pub struct Document {
    source: String,
    selection: Selection,
    revision: u64,
    undo: Vec<Change>,
    redo: Vec<Change>,
    composition: Option<Composition>,
}

impl Document {
    pub fn new(source: impl Into<String>) -> Self {
        Self {
            source: source.into(),
            selection: Selection::default(),
            revision: 0,
            undo: Vec::new(),
            redo: Vec::new(),
            composition: None,
        }
    }

    pub fn source(&self) -> &str {
        &self.source
    }
    pub fn selection(&self) -> Selection {
        self.selection
    }
    pub fn revision(&self) -> u64 {
        self.revision
    }
    pub fn composition(&self) -> Option<&Composition> {
        self.composition.as_ref()
    }
    pub fn can_undo(&self) -> bool {
        !self.undo.is_empty()
    }
    pub fn can_redo(&self) -> bool {
        !self.redo.is_empty()
    }

    /// The UI may group a continuous burst of typing. It supplies the timing
    /// policy; the model still verifies contiguous insertions and selections.
    pub fn coalesce_typing(&mut self) {
        if self.undo.len() < 2 || self.composition.is_some() {
            return;
        }
        let last = &self.undo[self.undo.len() - 1];
        let previous = &self.undo[self.undo.len() - 2];
        if !last.removed.is_empty()
            || last.inserted.is_empty()
            || previous.inserted.is_empty()
            || last.start != previous.start + previous.inserted.len()
            || last.before != previous.after
        {
            return;
        }
        let last = self.undo.pop().unwrap();
        let previous = self.undo.last_mut().unwrap();
        previous.inserted.push_str(&last.inserted);
        previous.after = last.after;
    }

    /// Safe for save/recovery workers even while native input is composing.
    pub fn snapshot(&self) -> Snapshot {
        Snapshot {
            source: Arc::from(self.source.as_str()),
            revision: self.revision,
        }
    }

    pub fn select(&mut self, selection: Selection) -> Result<(), EditError> {
        self.ensure_idle()?;
        validate_range(&self.source, &selection.range())?;
        self.selection = selection;
        Ok(())
    }

    pub fn replace(&mut self, range: Range<usize>, text: &str) -> Result<bool, EditError> {
        let after = Selection::caret(range.start.saturating_add(text.len()));
        self.replace_and_select(range, text, after)
    }

    /// Structured commands may retain text after the insertion caret. Store the
    /// final selection in the same history entry so Redo restores it precisely.
    pub fn replace_and_select(
        &mut self,
        range: Range<usize>,
        text: &str,
        after: Selection,
    ) -> Result<bool, EditError> {
        self.ensure_idle()?;
        validate_range(&self.source, &range)?;
        let inserted_end = range.start + text.len();
        let new_len = self.source.len() - range.len() + text.len();
        let valid_offset = |at| {
            if at < range.start {
                self.source.is_char_boundary(at)
            } else if at <= inserted_end {
                text.is_char_boundary(at - range.start)
            } else if at <= new_len {
                self.source.is_char_boundary(at - text.len() + range.len())
            } else {
                false
            }
        };
        if !valid_offset(after.anchor) || !valid_offset(after.head) {
            return Err(EditError::InvalidRange);
        }
        if self.source[range.clone()] == *text {
            self.selection = after;
            return Ok(false);
        }
        let change = Change {
            start: range.start,
            removed: self.source[range.clone()].to_owned(),
            inserted: text.to_owned(),
            before: self.selection,
            after,
        };
        self.source.replace_range(range, text);
        self.selection = after;
        self.revision += 1;
        self.undo.push(change);
        self.redo.clear();
        Ok(true)
    }

    pub fn replace_selection(&mut self, text: &str) -> Result<bool, EditError> {
        self.replace(self.selection.range(), text)
    }

    /// Undo cancels uncommitted input first, without undoing earlier writing.
    pub fn undo(&mut self) -> bool {
        if self.cancel_composition() {
            return true;
        }
        let Some(change) = self.undo.pop() else {
            return false;
        };
        self.source.replace_range(
            change.start..change.start + change.inserted.len(),
            &change.removed,
        );
        self.selection = change.before;
        self.revision += 1;
        self.redo.push(change);
        true
    }

    pub fn redo(&mut self) -> bool {
        if self.cancel_composition() {
            return true;
        }
        let Some(change) = self.redo.pop() else {
            return false;
        };
        self.source.replace_range(
            change.start..change.start + change.removed.len(),
            &change.inserted,
        );
        self.selection = change.after;
        self.revision += 1;
        self.undo.push(change);
        true
    }

    pub fn begin_composition(&mut self, range: Range<usize>) -> Result<(), EditError> {
        self.ensure_idle()?;
        validate_range(&self.source, &range)?;
        self.composition = Some(Composition {
            range,
            text: String::new(),
            selection: Selection::default(),
        });
        Ok(())
    }

    pub fn update_composition(
        &mut self,
        text: &str,
        selection: Selection,
    ) -> Result<(), EditError> {
        validate_range(text, &selection.range())?;
        let composition = self.composition.as_mut().ok_or(EditError::NoComposition)?;
        composition.text = text.to_owned();
        composition.selection = selection;
        Ok(())
    }

    /// A platform commit is one source transaction regardless of preedit count.
    pub fn commit_composition(&mut self, text: &str) -> Result<bool, EditError> {
        let composition = self.composition.take().ok_or(EditError::NoComposition)?;
        self.replace(composition.range, text)
    }

    pub fn cancel_composition(&mut self) -> bool {
        self.composition.take().is_some()
    }

    fn ensure_idle(&self) -> Result<(), EditError> {
        if self.composition.is_some() {
            Err(EditError::CompositionActive)
        } else {
            Ok(())
        }
    }
}

pub fn validate_range(text: &str, range: &Range<usize>) -> Result<(), EditError> {
    if range.start <= range.end
        && range.end <= text.len()
        && text.is_char_boundary(range.start)
        && text.is_char_boundary(range.end)
    {
        Ok(())
    } else {
        Err(EditError::InvalidRange)
    }
}

/// Platform UTF-16 offsets may fall inside a surrogate pair. Snap down for a
/// start/caret and up for a range end, so no operation splits an emoji.
pub fn utf16_to_byte(text: &str, offset: usize, round_up: bool) -> usize {
    let mut units = 0;
    for (byte, ch) in text.char_indices() {
        if units == offset {
            return byte;
        }
        let next = units + ch.len_utf16();
        if next > offset {
            return if round_up { byte + ch.len_utf8() } else { byte };
        }
        units = next;
    }
    text.len()
}

pub fn byte_to_utf16(text: &str, byte: usize) -> Result<usize, EditError> {
    validate_range(text, &(byte..byte))?;
    Ok(text[..byte].encode_utf16().count())
}
