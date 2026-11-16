use gpui_kit::{AssetSource, SharedString, component::Icon};
use std::borrow::Cow;

pub struct Assets;
pub const WINDOW_ICONS: [&str; 4] = [
    "window-minimize",
    "window-maximize",
    "window-restore",
    "window-close",
];
impl AssetSource for Assets {
    fn load(&self, path: &str) -> anyhow::Result<Option<Cow<'static, [u8]>>> {
        // TitleBar requests toolkit paths; supply the same embedded vectors as
        // the rest of MarkGo instead of relying on a second window icon family.
        let path = match path {
            "icons/window-minimize.svg" => "markgo/window-minimize.svg",
            "icons/window-maximize.svg" => "markgo/window-maximize.svg",
            "icons/window-restore.svg" => "markgo/window-restore.svg",
            "icons/window-close.svg" => "markgo/window-close.svg",
            path => path,
        };
        macro_rules! icons { ($($name:literal),*) => { match path {
            $(concat!("markgo/",$name,".svg") => return Ok(Some(Cow::Borrowed(include_bytes!(concat!("../../assets/icons/",$name,".svg")))))),*,
            _=>{}
        } } }
        icons!(
            "mark",
            "file",
            "folder",
            "close",
            "plus",
            "sidebar",
            "search",
            "save",
            "more",
            "focus",
            "exit-focus",
            "appearance",
            "bold",
            "italic",
            "code",
            "link",
            "list",
            "task-list",
            "numbered-list",
            "quote",
            "paragraph",
            "check",
            "chevron",
            "undo",
            "read",
            "edit",
            "format",
            "previous",
            "next",
            "refresh",
            "minus",
            "dirty",
            "strikethrough",
            "recovery",
            "window-minimize",
            "window-maximize",
            "window-restore",
            "window-close"
        );
        gpui_kit::assets::Assets.load(path)
    }
    fn list(&self, path: &str) -> anyhow::Result<Vec<SharedString>> {
        gpui_kit::assets::Assets.list(path)
    }
}
pub fn icon(name: &str) -> Icon {
    Icon::default().path(format!("markgo/{name}.svg"))
}
