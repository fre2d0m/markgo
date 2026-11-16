//! Versioned, bounded settings; native migration uses a separate settings file.
use crate::storage::{self, FileSnapshot};
use std::{io, path::PathBuf, sync::Arc};

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Preferences {
    pub theme: String,
    pub prose: u8,
    pub column: u16,
}
impl Default for Preferences {
    fn default() -> Self {
        Self {
            theme: "paper".into(),
            prose: 18,
            column: 720,
        }
    }
}
pub const THEMES: [&str; 12] = [
    "paper", "warm", "sakura", "sage", "mist", "lavender", "sand", "pearl", "graphite", "ink",
    "pine", "plum",
];
impl Preferences {
    pub fn parse(text: &str) -> io::Result<Self> {
        if text.len() > 4096 {
            return Err(io::Error::other("外观配置过大，保留原文件"));
        }
        let mut lines = text.lines();
        if lines.next() != Some("markgo-appearance-v1") {
            return Err(io::Error::other("外观配置版本不受支持，保留原文件"));
        }
        let mut result = Self::default();
        for line in lines {
            let Some((key, value)) = line.split_once('=') else {
                return Err(io::Error::other("外观配置格式不正确"));
            };
            match key {
                "theme" if THEMES.contains(&value) => result.theme = value.into(),
                "prose" => {
                    result.prose = value.parse().map_err(io::Error::other)?;
                    if !(14..=24).contains(&result.prose) {
                        return Err(io::Error::other("字号超出范围"));
                    }
                }
                "column" => {
                    result.column = value.parse().map_err(io::Error::other)?;
                    if ![640, 720, 840].contains(&result.column) {
                        return Err(io::Error::other("阅读宽度超出范围"));
                    }
                }
                _ => return Err(io::Error::other("未知外观配置，保留原文件")),
            }
        }
        Ok(result)
    }
    pub fn encode(&self) -> Arc<str> {
        format!(
            "markgo-appearance-v1\ntheme={}\nprose={}\ncolumn={}\n",
            self.theme, self.prose, self.column
        )
        .into()
    }
}
pub struct Store {
    path: PathBuf,
    snapshot: Option<FileSnapshot>,
}
impl Store {
    pub fn load(path: PathBuf) -> io::Result<(Self, Preferences)> {
        match storage::open(&path) {
            Ok(snapshot) => {
                let preferences = Preferences::parse(&snapshot.source)?;
                Ok((
                    Self {
                        path,
                        snapshot: Some(snapshot),
                    },
                    preferences,
                ))
            }
            Err(error) if error.kind() == io::ErrorKind::NotFound => Ok((
                Self {
                    path,
                    snapshot: None,
                },
                Preferences::default(),
            )),
            Err(error) => Err(error),
        }
    }
    pub fn write(mut self, preferences: Preferences) -> io::Result<Self> {
        if let Some(parent) = self.path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        let source = preferences.encode();
        self.snapshot = Some(if let Some(snapshot) = &self.snapshot {
            storage::save(snapshot, source)?
        } else {
            storage::create(&self.path, source)?
        });
        Ok(self)
    }
}
pub fn path() -> Option<PathBuf> {
    #[cfg(target_os = "macos")]
    let base = std::env::var_os("HOME")
        .map(PathBuf::from)
        .map(|p| p.join("Library/Application Support"));
    #[cfg(target_os = "windows")]
    let base = std::env::var_os("APPDATA").map(PathBuf::from);
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    let base = std::env::var_os("XDG_CONFIG_HOME")
        .map(PathBuf::from)
        .or_else(|| {
            std::env::var_os("HOME")
                .map(PathBuf::from)
                .map(|p| p.join(".config"))
        });
    base.map(|p| p.join("MarkGo/native/appearance.conf"))
}
