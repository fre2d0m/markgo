use gpui_kit::{AssetSource, SharedString, component::Icon};
use std::borrow::Cow;

pub struct Assets;
impl AssetSource for Assets {
    fn load(&self, path: &str) -> anyhow::Result<Option<Cow<'static, [u8]>>> {
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
            "minus"
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
