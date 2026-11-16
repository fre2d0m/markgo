//! Real GPU rendering of the application, in an isolated offscreen window.
use gpui_kit::{
    AppContext, Bounds, EntityInputHandler, HeadlessAppContext, WindowBounds, WindowOptions, px,
    size, test::TestWindowExt,
};
use markgo_native::desktop::{self, Workbench, assets::Assets, theme::PALETTES};
use std::sync::Arc;

fn main() {
    let output = std::env::args()
        .nth(1)
        .unwrap_or_else(|| "../build/gpui-evidence".into());
    std::fs::create_dir_all(&output).unwrap();
    for (name, width, height, palette, preview) in [
        ("paper", 1320., 840., 0, true),
        ("warm", 1320., 840., 1, true),
        ("sakura", 1320., 840., 2, true),
        ("graphite", 1320., 840., 8, true),
        ("narrow", 960., 640., 0, true),
        ("empty", 960., 640., 0, false),
    ] {
        let mut cx = HeadlessAppContext::with_platform(
            gpui_kit::platform::current_platform(true).text_system(),
            Arc::new(Assets),
            gpui_kit::platform::current_headless_renderer,
        );
        cx.update(desktop::init);
        let (handle, view) = cx
            .update(|cx| {
                gpui_kit::open_window(
                    WindowOptions {
                        window_bounds: Some(WindowBounds::Windowed(Bounds {
                            origin: Default::default(),
                            size: size(px(width), px(height)),
                        })),
                        show: false,
                        focus: false,
                        ..Default::default()
                    },
                    cx,
                    |w, c| {
                        c.new(|c| {
                            if preview {
                                Workbench::new(true, w, c)
                            } else {
                                Workbench::empty_preview(w, c)
                            }
                        })
                    },
                )
            })
            .unwrap();
        cx.update_window(handle, |_, w, c| {
            view.update(c, |v, c| v.set_palette(palette, w, c));
            w.render_frame(c);
        })
        .unwrap();
        cx.capture_screenshot(handle)
            .expect("Real Metal/WGPU renderer required")
            .save(format!("{output}/{name}.png"))
            .unwrap();
        println!(
            "Rendered {name} ({width} × {height}, {})",
            PALETTES[palette].id
        );
    }
    interactions(&output);
}

fn interactions(output: &str) {
    let mut cx = HeadlessAppContext::with_platform(
        gpui_kit::platform::current_platform(true).text_system(),
        Arc::new(Assets),
        gpui_kit::platform::current_headless_renderer,
    );
    cx.update(desktop::init);
    let (handle, view) = cx
        .update(|c| {
            gpui_kit::open_window(
                WindowOptions {
                    window_bounds: Some(WindowBounds::Windowed(Bounds {
                        origin: Default::default(),
                        size: size(px(960.), px(640.)),
                    })),
                    show: false,
                    focus: false,
                    ..Default::default()
                },
                c,
                |w, c| c.new(|c| Workbench::new(true, w, c)),
            )
        })
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.render_frame(c);
        w.click("new-tab", c);
        w.input("中文 👩‍💻 hello", c);
        assert_eq!(view.read(c).inspection(c).2, "中文 👩‍💻 hello");
        w.press("cmd-z", c);
        assert_eq!(view.read(c).inspection(c).2, "");
        w.press("cmd-shift-z", c);
        assert_eq!(view.read(c).inspection(c).2, "中文 👩‍💻 hello");
        let before = view.read(c).inspection(c);
        w.click(("file", 1usize), c);
        w.click(("file", 4usize), c);
        let after = view.read(c).inspection(c);
        assert_eq!(
            before, after,
            "tab switching retains source, caret and history"
        );
        w.press("cmd-w", c);
        assert!(w.try_find("cancel-close").is_some());
        w.click("cancel-close", c);
        assert_eq!(view.read(c).inspection(c).0, 4);
        w.press("cmd-a", c);
        w.press("backspace", c);
        w.input("/", c);
        assert!(w.try_find(("slash", 1usize)).is_some());
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/slash.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press("down", c);
        w.press("enter", c);
        w.input("真正的标题", c);
        assert_eq!(view.read(c).inspection(c).2, "# 真正的标题");
        let entity = view.read(c).input_entity();
        entity.update(c, |v, c| {
            v.replace_and_mark_text_in_range(None, "ni", Some(2..2), w, c)
        });
        w.render_frame(c);
        assert_eq!(
            view.read(c).inspection(c).2,
            "# 真正的标题",
            "preedit excluded from committed source"
        );
        entity.update(c, |v, c| v.replace_text_in_range(None, "你好", w, c));
        w.render_frame(c);
        assert_eq!(view.read(c).inspection(c).2, "# 真正的标题你好");
        entity.update(c, |v, c| {
            let selection = v.selected_text_range(false, w, c).unwrap();
            let rect = v
                .bounds_for_range(selection.range, Bounds::default(), w, c)
                .expect("candidate geometry");
            assert!(rect.size.height > px(30.));
            assert!(rect.origin.x >= px(224.));
            assert!(rect.origin.y >= px(96.));
        });
        w.press("cmd-z", c);
        assert_eq!(view.read(c).inspection(c).2, "# 真正的标题");
        w.click("appearance", c);
        for i in 0usize..12 {
            w.click(("theme", i), c);
            assert_eq!(view.read(c).inspection(c).2, "# 真正的标题");
        }
        w.click(("theme", 2usize), c);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/appearance.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.click("close-panel", c);
        w.click("search", c);
        w.input("标题", c);
        w.click("toggle-replace", c);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/search.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.click("close-search", c);
        w.press("cmd-w", c);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/close-guard.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.click("cancel-close", c);
        view.update(c, |v, c| v.render_error_fixture(w, c));
        w.render_frame(c);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/error.png"))
        .unwrap();
    println!(
        "Passed native event dispatch: input, undo/redo, tabs, close guard, slash, composition, geometry and 12 themes"
    );
}
