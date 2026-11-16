//! Guarded local writes. All calls run on a worker, never during UI rendering.
use std::{
    fs::{self, File, OpenOptions, Permissions},
    io::{self, Read, Write},
    path::{Path, PathBuf},
    sync::{
        Arc,
        atomic::{AtomicU64, Ordering},
    },
};

const MAX_FILE_BYTES: u64 = 32 * 1024 * 1024;
static TEMP_ID: AtomicU64 = AtomicU64::new(1);

/// Keep native filesystem paths intact; remove Windows transport prefixes only
/// in labels and tooltips.
pub fn display_path(path: &Path) -> String {
    let value = path.to_string_lossy();
    if let Some(unc) = value.strip_prefix("\\\\?\\UNC\\") {
        return format!("\\\\{unc}");
    }
    if let Some(disk) = value.strip_prefix("\\\\?\\")
        && disk.as_bytes().get(1..3) == Some(b":\\")
    {
        return disk.to_owned();
    }
    value.into_owned()
}

#[derive(Clone, Debug)]
pub struct FileSnapshot {
    pub path: PathBuf,
    pub source: Arc<str>,
    permissions: Permissions,
    read_limit: u64,
}

pub fn open(path: &Path) -> io::Result<FileSnapshot> {
    open_bounded(path, MAX_FILE_BYTES)
}
pub(crate) fn open_bounded(path: &Path, limit: u64) -> io::Result<FileSnapshot> {
    let metadata = fs::symlink_metadata(path)?;
    if !metadata.is_file() {
        return Err(io::Error::other("请选择普通 Markdown 文件；不跟随符号链接"));
    }
    if metadata.len() > limit {
        return Err(io::Error::other("文件超过 32 MB，未打开，原文件没有改动"));
    }
    let mut source = String::new();
    File::open(path)?
        .take(limit + 1)
        .read_to_string(&mut source)?;
    if source.len() as u64 > limit {
        return Err(io::Error::other("文件读取时超过 32 MB"));
    }
    Ok(FileSnapshot {
        path: fs::canonicalize(path)?,
        source: source.into(),
        permissions: metadata.permissions(),
        read_limit: limit,
    })
}

fn check(snapshot: &FileSnapshot) -> io::Result<()> {
    let current = open_bounded(&snapshot.path, snapshot.read_limit)?;
    if current.source != snapshot.source || current.permissions != snapshot.permissions {
        return Err(io::Error::other(
            "文件已被其他程序修改。请另存副本，避免覆盖外部更改",
        ));
    }
    Ok(())
}

struct Temp(PathBuf);
impl Drop for Temp {
    fn drop(&mut self) {
        let _ = fs::remove_file(&self.0);
    }
}

fn write_temp(path: &Path, source: &str, permissions: Option<Permissions>) -> io::Result<Temp> {
    let parent = path
        .parent()
        .ok_or_else(|| io::Error::other("文件路径没有目录"))?;
    let mut options = OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(0o600);
    }
    for _ in 0..100 {
        let name = parent.join(format!(
            ".markgo-native-{}-{}.tmp",
            std::process::id(),
            TEMP_ID.fetch_add(1, Ordering::Relaxed)
        ));
        let mut file = match options.open(&name) {
            Ok(f) => f,
            Err(e) if e.kind() == io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(e),
        };
        let temp = Temp(name);
        if let Some(permissions) = permissions {
            file.set_permissions(permissions)?;
        }
        file.write_all(source.as_bytes())?;
        file.sync_all()?;
        return Ok(temp);
    }
    Err(io::Error::other("无法创建用于安全保存的临时文件"))
}

pub fn save(snapshot: &FileSnapshot, source: Arc<str>) -> io::Result<FileSnapshot> {
    if snapshot.permissions.readonly() {
        return Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            "文件只读；请另存副本",
        ));
    }
    check(snapshot)?;
    let temp = write_temp(&snapshot.path, &source, Some(snapshot.permissions.clone()))?;
    check(snapshot)?;
    fs::rename(&temp.0, &snapshot.path)?;
    Ok(FileSnapshot {
        path: snapshot.path.clone(),
        source,
        permissions: snapshot.permissions.clone(),
        read_limit: snapshot.read_limit,
    })
}

pub fn create(path: &Path, source: Arc<str>) -> io::Result<FileSnapshot> {
    // Canonicalize the parent only: the destination must not already exist.
    let name = path
        .file_name()
        .ok_or_else(|| io::Error::other("请输入文件名"))?;
    let path = fs::canonicalize(path.parent().unwrap_or(Path::new(".")))?.join(name);
    let temp = write_temp(&path, &source, None)?;
    fs::hard_link(&temp.0, &path).map_err(|error| {
        if error.kind() == io::ErrorKind::AlreadyExists {
            io::Error::new(error.kind(), "此位置已有文件，请选择新名称")
        } else {
            error
        }
    })?;
    let permissions = fs::metadata(&path)?.permissions();
    Ok(FileSnapshot {
        path,
        source,
        permissions,
        read_limit: MAX_FILE_BYTES,
    })
}

#[derive(Clone, Debug)]
pub struct FolderEntry {
    pub path: PathBuf,
    pub name: String,
    pub depth: usize,
    pub directory: bool,
}
pub fn list_folder(folder: &Path) -> io::Result<Vec<FolderEntry>> {
    fn walk(path: &Path, depth: usize, result: &mut Vec<FolderEntry>) -> io::Result<()> {
        if depth > 8 || result.len() >= 2000 {
            return Ok(());
        }
        let mut entries = fs::read_dir(path)?.collect::<Result<Vec<_>, _>>()?;
        entries.sort_by_key(|entry| entry.file_name());
        for entry in entries {
            let kind = entry.file_type()?;
            if entry.file_name().to_string_lossy().starts_with('.') {
                continue;
            }
            if kind.is_dir() {
                result.push(FolderEntry {
                    path: entry.path(),
                    name: entry.file_name().to_string_lossy().into_owned(),
                    depth,
                    directory: true,
                });
                walk(&entry.path(), depth + 1, result)?;
            } else if kind.is_file()
                && entry.path().extension().is_some_and(|s| {
                    s.eq_ignore_ascii_case("md") || s.eq_ignore_ascii_case("markdown")
                })
            {
                result.push(FolderEntry {
                    path: entry.path(),
                    name: entry.file_name().to_string_lossy().into_owned(),
                    depth,
                    directory: false,
                });
            }
            if result.len() >= 2000 {
                break;
            }
        }
        Ok(())
    }
    let mut result = vec![];
    walk(folder, 0, &mut result)?;
    Ok(result)
}
