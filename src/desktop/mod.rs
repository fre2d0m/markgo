pub mod assets;
mod controls;
mod navigation;
mod recovery;
pub mod theme;
mod views;
mod workbench;
pub mod writer;

use gpui_kit::{component::TitleBar, *};
use theme::PALETTES;
pub use workbench::Workbench;

actions!(
    markgo,
    [
        NewDocument,
        OpenDocument,
        OpenFolder,
        SaveDocument,
        SaveAsDocument,
        CloseDocument,
        NextDocument,
        PreviousDocument,
        Search,
        Appearance,
        ToggleSidebar,
        Dismiss,
        Quit,
        EditLink,
        ToggleSource
    ]
);

pub const SAMPLE: &str = "# 让想法，慢慢成为文字\n\n写作需要的，是一张安静的纸。\n\nMarkGo 把注意力留给内容，让文件、格式与工具各就其位。打开几份文档，思路可以随时切换，光标和修改也会留在原处。\n\n## 从一句话开始\n\n中文、English 和 emoji 🌱 可以写在同一段里。选中文字后设置格式，或者在空段落输入 /，继续你的思路。\n\n## 一点自己的颜色\n\n素纸、暖纸、樱花、鼠尾草，直到夜里的石墨。十二种克制的配色，为不同的写作时刻留一点余地。\n\n## 保留文字本来的样子\n\n每一份文档都是本地 Markdown 文件。修改可以撤销，保存不会默默覆盖外部变更。\n";

pub fn init(cx: &mut App) {
    gpui_kit::init(cx);
    PALETTES[0].apply(None, cx);
    writer::bind(cx);
    for modifier in ["cmd", "ctrl"] {
        cx.bind_keys([
            KeyBinding::new(
                if modifier == "cmd" { "cmd-n" } else { "ctrl-n" },
                NewDocument,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-o" } else { "ctrl-o" },
                OpenDocument,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" {
                    "cmd-shift-o"
                } else {
                    "ctrl-shift-o"
                },
                OpenFolder,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-s" } else { "ctrl-s" },
                SaveDocument,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" {
                    "cmd-shift-s"
                } else {
                    "ctrl-shift-s"
                },
                SaveAsDocument,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-w" } else { "ctrl-w" },
                CloseDocument,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-f" } else { "ctrl-f" },
                Search,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" { "cmd-," } else { "ctrl-," },
                Appearance,
                Some("MarkGo"),
            ),
            KeyBinding::new(
                if modifier == "cmd" {
                    "cmd-\\"
                } else {
                    "ctrl-\\"
                },
                ToggleSidebar,
                Some("MarkGo"),
            ),
        ]);
    }
    cx.bind_keys([
        KeyBinding::new("cmd-k", EditLink, Some("MarkGo")),
        KeyBinding::new("ctrl-k", EditLink, Some("MarkGo")),
        KeyBinding::new("cmd-shift-m", ToggleSource, Some("MarkGo")),
        KeyBinding::new("ctrl-shift-m", ToggleSource, Some("MarkGo")),
        KeyBinding::new("cmd-q", Quit, Some("MarkGo")),
        KeyBinding::new("escape", Dismiss, Some("MarkGoModal")),
        KeyBinding::new("ctrl-tab", NextDocument, Some("MarkGo")),
        KeyBinding::new("ctrl-shift-tab", PreviousDocument, Some("MarkGo")),
        KeyBinding::new("escape", Dismiss, Some("MarkGo")),
    ]);
}
pub fn options(cx: &App) -> WindowOptions {
    WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds::centered(
            None,
            size(px(1320.), px(840.)),
            cx,
        ))),
        window_min_size: Some(size(px(960.), px(640.))),
        app_id: Some("dev.markgo.app".into()),
        titlebar: Some(TitlebarOptions {
            title: Some("MarkGo".into()),
            appears_transparent: true,
            traffic_light_position: Some(point(px(16.), px(17.))),
        }),
        ..TitleBar::window_options()
    }
}
pub fn run() {
    let preview = std::env::args().any(|s| s == "--preview")
        || std::env::var("MARKGO_EPHEMERAL").is_ok_and(|s| s == "1");
    let application = application().with_assets(assets::Assets);
    // The platform may deliver file URLs before the window exists.
    let pending = std::rc::Rc::new(std::cell::RefCell::new(Vec::new()));
    type OpenHandler = std::rc::Rc<std::cell::RefCell<Option<Box<dyn Fn(Vec<String>)>>>>;
    let handler: OpenHandler = Default::default();
    let queued = pending.clone();
    let handler_copy = handler.clone();
    application.on_open_urls(move |urls| {
        if let Some(handler) = handler_copy.borrow().as_ref() {
            handler(urls);
        } else {
            queued.borrow_mut().extend(urls);
        }
    });
    let paths = std::env::args_os()
        .skip(1)
        .filter(|arg| !arg.to_string_lossy().starts_with('-'))
        .map(std::path::PathBuf::from)
        .collect::<Vec<_>>();
    application.run(move |cx| {
        init(cx);
        cx.set_quit_mode(QuitMode::LastWindowClosed);
        cx.set_menus([
            Menu::new("MarkGo").items([
                MenuItem::action("外观…", Appearance),
                MenuItem::separator(),
                MenuItem::action("退出 MarkGo", Quit),
            ]),
            Menu::new("文件").items([
                MenuItem::action("新建文档", NewDocument),
                MenuItem::action("打开文档…", OpenDocument),
                MenuItem::action("打开文件夹…", OpenFolder),
                MenuItem::separator(),
                MenuItem::action("保存", SaveDocument),
                MenuItem::action("另存为…", SaveAsDocument),
                MenuItem::separator(),
                MenuItem::action("关闭文档", CloseDocument),
            ]),
            Menu::new("编辑").items([
                MenuItem::action("撤销", writer::Undo),
                MenuItem::action("重做", writer::Redo),
                MenuItem::separator(),
                MenuItem::action("剪切", writer::Cut),
                MenuItem::action("复制", writer::Copy),
                MenuItem::action("粘贴", writer::Paste),
                MenuItem::action("全选", writer::SelectAll),
                MenuItem::separator(),
                MenuItem::action("查找与替换", Search),
            ]),
        ]);
        let mut options = options(cx);
        if preview {
            options.app_id = Some("dev.markgo.writing-preview".into());
        }
        let (handle, view) = gpui_kit::open_window(options, cx, |window, cx| {
            cx.new(|cx| Workbench::new(preview, window, cx))
        })
        .expect("无法打开 MarkGo 窗口");
        if !preview {
            let mut paths = paths;
            paths.extend(
                pending
                    .borrow_mut()
                    .drain(..)
                    .filter_map(|s| url::Url::parse(&s).ok().and_then(|u| u.to_file_path().ok())),
            );
            let _ = handle.update(cx, |_, w, c| {
                view.update(c, |s, c| s.open_external(paths, w, c))
            });
            let async_cx = cx.to_async();
            let callback = Box::new(move |urls: Vec<String>| {
                let mut async_cx = async_cx.clone();
                let paths = urls
                    .iter()
                    .filter_map(|s| url::Url::parse(s).ok().and_then(|u| u.to_file_path().ok()))
                    .collect();
                let _ = async_cx.update_window(handle, |_, w, c| {
                    view.update(c, |s, c| s.open_external(paths, w, c))
                });
            });
            *handler.borrow_mut() = Some(callback);
        }
        cx.activate(true);
    });
}
