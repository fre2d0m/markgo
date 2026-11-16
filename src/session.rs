//! Versioned recovery journal. Never writes the user's document paths.
use crate::storage::{self, FileSnapshot};
use serde::{Deserialize, Serialize};
use std::{
    collections::HashSet,
    io,
    path::{Path, PathBuf},
    sync::Arc,
};

pub const MAX_BYTES: usize = 64 * 1024 * 1024;
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Draft {
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub dirty: bool,
    #[serde(default)]
    pub source: String,
    #[serde(default)]
    pub original: String,
    #[serde(default, rename = "bufferVersion")]
    pub revision: u64,
    #[serde(default)]
    pub anchor: usize,
    #[serde(default)]
    pub head: usize,
    #[serde(default, rename = "scrollY")]
    pub scroll: f32,
    #[serde(default, rename = "sourceMode")]
    pub source_mode: bool,
    #[serde(default)]
    pub reading: bool,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Session {
    pub schema: u8,
    #[serde(default, deserialize_with = "null_vec")]
    pub documents: Vec<Draft>,
    #[serde(default)]
    pub active: String,
    #[serde(default, deserialize_with = "null_vec")]
    pub recent: Vec<String>,
    #[serde(default)]
    pub folder: String,
    #[serde(default = "yes")]
    pub sidebar: bool,
    #[serde(default)]
    pub outline: bool,
    #[serde(default)]
    pub reading: bool,
    #[serde(default, rename = "sourceMode")]
    pub source_mode: bool,
}
fn yes() -> bool {
    true
}
fn null_vec<'de, D, T>(de: D) -> Result<Vec<T>, D::Error>
where
    D: serde::Deserializer<'de>,
    T: Deserialize<'de>,
{
    Ok(Option::<Vec<T>>::deserialize(de)?.unwrap_or_default())
}
impl Default for Session {
    fn default() -> Self {
        Self {
            schema: 2,
            documents: vec![],
            active: String::new(),
            recent: vec![],
            folder: String::new(),
            sidebar: true,
            outline: false,
            reading: false,
            source_mode: false,
        }
    }
}
impl Session {
    pub fn parse(source: &str) -> io::Result<Self> {
        if source.len() > MAX_BYTES {
            return Err(io::Error::other("恢复文件超过 64 MiB，已保留原文件"));
        }
        let mut state: Self = serde_json::from_str(source).map_err(io::Error::other)?;
        if state.schema == 1 {
            for doc in &mut state.documents {
                doc.source_mode = state.source_mode;
                doc.reading = state.reading;
            }
            state.schema = 2;
        }
        state.validate()?;
        Ok(state)
    }
    pub fn validate(&self) -> io::Result<()> {
        if self.schema != 2 {
            return Err(io::Error::other("不支持的恢复版本，已保留原文件"));
        }
        if self.documents.len() > 100 || self.recent.len() > 20 {
            return Err(io::Error::other("恢复最多支持 100 篇文档和 20 个最近路径"));
        }
        if self.folder.contains('\0')
            || self.recent.iter().any(|p| p.is_empty() || p.contains('\0'))
        {
            return Err(io::Error::other("恢复路径无效，已保留原文件"));
        }
        let mut ids = HashSet::new();
        let mut paths = HashSet::new();
        let mut bytes = 0usize;
        for doc in &self.documents {
            bytes = bytes
                .saturating_add(doc.source.len())
                .saturating_add(doc.original.len());
            if doc.id.is_empty()
                || doc.name.is_empty()
                || !ids.insert(&doc.id)
                || (!doc.path.is_empty() && !paths.insert(&doc.path))
                || !doc.scroll.is_finite()
                || doc.scroll < 0.
                || doc.path.contains('\0')
            {
                return Err(io::Error::other("恢复数据无效，已保留原文件"));
            }
            if doc.dirty
                && (!doc.source.is_char_boundary(doc.anchor)
                    || !doc.source.is_char_boundary(doc.head))
            {
                return Err(io::Error::other("恢复光标不是有效文字边界"));
            }
        }
        if bytes > MAX_BYTES {
            return Err(io::Error::other("草稿超过 64 MiB，请先保存文档"));
        }
        Ok(())
    }
    pub fn encode(&self) -> io::Result<Arc<str>> {
        self.validate()?;
        let data = serde_json::to_string(self).map_err(io::Error::other)?;
        if data.len() > MAX_BYTES {
            return Err(io::Error::other("恢复文件超过 64 MiB，旧草稿未覆盖"));
        }
        Ok(data.into())
    }
}
pub struct Journal {
    path: PathBuf,
    snapshot: Option<FileSnapshot>,
}
/// Serializes idle and shutdown checkpoints. An older worker can never restore
/// an obsolete snapshot after a newer checkpoint has reached disk.
pub struct OrderedJournal {
    journal: Option<Journal>,
    generation: Option<u64>,
}
impl OrderedJournal {
    pub fn new(journal: Journal) -> Self {
        Self {
            journal: Some(journal),
            generation: None,
        }
    }
    pub fn write(&mut self, generation: u64, state: Session) -> io::Result<()> {
        if self.generation.is_some_and(|saved| saved >= generation) {
            return Ok(());
        }
        let journal = self
            .journal
            .take()
            .expect("journal returned after every write");
        match journal.write(state) {
            Ok(journal) => {
                self.journal = Some(journal);
                self.generation = Some(generation);
                Ok(())
            }
            Err((journal, error)) => {
                self.journal = Some(journal);
                Err(error)
            }
        }
    }
}
impl Journal {
    /// Read the previous known data location only on first use. Subsequent
    /// checkpoints target the new path; the original is never renamed/deleted.
    pub fn load_with_fallback(
        path: PathBuf,
        previous: Option<PathBuf>,
    ) -> io::Result<(Self, Session)> {
        let (journal, state) = Self::load(path)?;
        if journal.snapshot.is_none()
            && let Some(previous) = previous
        {
            match storage::open_bounded(&previous, MAX_BYTES as u64) {
                Ok(snapshot) => return Ok((journal, Session::parse(&snapshot.source)?)),
                Err(error) if error.kind() == io::ErrorKind::NotFound => {}
                Err(error) => return Err(error),
            }
        }
        Ok((journal, state))
    }
    pub fn load(path: PathBuf) -> io::Result<(Self, Session)> {
        match storage::open_bounded(&path, MAX_BYTES as u64) {
            Ok(snapshot) => {
                let state = Session::parse(&snapshot.source)?;
                Ok((
                    Self {
                        path,
                        snapshot: Some(snapshot),
                    },
                    state,
                ))
            }
            Err(e) if e.kind() == io::ErrorKind::NotFound => Ok((
                Self {
                    path,
                    snapshot: None,
                },
                Session::default(),
            )),
            Err(e) => Err(e),
        }
    }
    pub fn write(mut self, state: Session) -> Result<Self, (Self, io::Error)> {
        let result = (|| {
            let data = state.encode()?;
            if let Some(parent) = self.path.parent() {
                std::fs::create_dir_all(parent)?;
            }
            if let Some(snapshot) = &self.snapshot {
                storage::save(snapshot, data)?;
            } else {
                storage::create(&self.path, data)?;
            }
            storage::open_bounded(&self.path, MAX_BYTES as u64)
        })();
        match result {
            Ok(snapshot) => {
                self.snapshot = Some(snapshot);
                Ok(self)
            }
            Err(error) => Err((self, error)),
        }
    }
}
pub struct Restored {
    pub draft: Draft,
    pub file: Option<FileSnapshot>,
    pub conflict: bool,
}
pub fn restore(draft: Draft) -> Restored {
    let file = if draft.path.is_empty() {
        None
    } else {
        storage::open(Path::new(&draft.path)).ok()
    };
    let conflict = !draft.path.is_empty()
        && file
            .as_ref()
            .is_none_or(|f| draft.dirty && f.source.as_ref() != draft.original);
    let mut draft = draft;
    if !draft.dirty
        && let Some(file) = &file
    {
        draft.source = file.source.to_string();
        draft.original = draft.source.clone();
    }
    // A changed/missing destination cannot be saved until the user explicitly
    // chooses a new path. Retaining the buffer never grants overwrite consent.
    Restored {
        draft,
        file: if conflict { None } else { file },
        conflict,
    }
}
pub fn path() -> Option<PathBuf> {
    crate::preferences::path().and_then(|p| p.parent().map(|p| p.join("session.json")))
}
