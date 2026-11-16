//! Real GPU rendering of the application, in an isolated offscreen window.
use gpui_kit::{
    AppContext, Bounds, EntityInputHandler, HeadlessAppContext, WindowBounds, WindowOptions, px,
    size, test::TestWindowExt,
};
use markgo::desktop::{self, Workbench, assets::Assets, theme::PALETTES};
use std::sync::Arc;

fn main() {
    let output = std::env::args()
        .nth(1)
        .unwrap_or_else(|| "build/gpui-evidence".into());
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
    rich_interactions(&output);
}

fn rich_interactions(output: &str) {
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
                |w, c| c.new(|c| Workbench::empty_preview(w, c)),
            )
        })
        .unwrap();
    let source = "# 把想法写清楚\n\n**重点**、_节奏_、~~删去~~ 和 `code`，都留在原文里。\n\n- [ ] 留一点时间，认真写一句话\n- [x] 让工具安静地待在手边\n\n> 好的工具，让表达更自然。\n\n```rust\nlet thought = \"hello, 世界\";\n```\n";
    cx.update_window(handle, |_, w, c| {
        let entity = view.read(c).input_entity();
        entity.update(c, |writer, c| {
            writer.editor = markgo::editing::Editor::new(source);
            writer.mode(false, false, w, c);
        });
        w.render_frame(c);
        let task = source.find("[ ]").unwrap();
        w.render_frame(c);
        assert!(w.try_find(("task", task)).is_some());
        w.click(("task", task), c);
        assert_eq!(
            view.read(c).inspection(c).2,
            source.replacen("[ ]", "[x]", 1)
        );
        w.press("cmd-z", c);
        assert_eq!(view.read(c).inspection(c).2, source);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/rich-writing.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.click(("code-language", 0usize), c);
    })
    .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press("cmd-a", c);
        w.input("bad language!", c);
        w.click("apply-entry", c);
        assert_eq!(
            view.read(c).inspection(c).2,
            source,
            "invalid language is non-destructive"
        );
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/code-error.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press("escape", c);
        w.click(("code-language", 0usize), c);
    })
    .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press("cmd-a", c);
        w.input("python", c);
        w.click("apply-entry", c);
        assert!(view.read(c).inspection(c).2.contains("```python"));
        w.press("cmd-z", c);
        assert_eq!(view.read(c).inspection(c).2, source);
        w.click(("copy-code", 0usize), c);
        assert_eq!(
            c.read_from_clipboard().and_then(|x| x.text()).as_deref(),
            Some("let thought = \"hello, 世界\";")
        );
        let entity = view.read(c).input_entity();
        let at = source.find("工具").unwrap();
        entity.update(c, |writer, c| {
            writer.select_source(at..at + "工具".len(), w, c)
        });
        w.click("edit-link", c);
        w.input("  javascript:alert(1)", c);
        w.click("apply-entry", c);
        assert_eq!(view.read(c).inspection(c).2, source);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/link-error.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press("escape", c);
        w.click("edit-link", c);
        w.input("https://example.com", c);
        w.click("apply-entry", c);
        assert!(
            view.read(c)
                .inspection(c)
                .2
                .contains("[工具](<https://example.com>)")
        );
        w.press("cmd-z", c);
        assert_eq!(view.read(c).inspection(c).2, source);
        w.click("all-tabs", c);
        assert!(w.try_find(("all-tab", 1usize)).is_some());
        w.click(("all-tab", 1usize), c);
        view.update(c, |s, c| s.set_palette(8, w, c));
        w.render_frame(c);
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/rich-graphite.png"))
        .unwrap();
    cx.update_window(handle,|_,w,c|{
        view.update(c,|s,c|s.set_palette(0,w,c));
        let entity=view.read(c).input_entity();entity.update(c,|writer,c|{writer.editor=markgo::editing::Editor::new("# 阅读也轻松\n\n| 章节 | 进展 |\n| --- | --- |\n| 开始 | 已写下第一句 |\n| 继续 | 留给明天 |\n\n**写作**是慢慢整理自己的想法。\n");writer.mode(false,false,w,c);});w.render_frame(c);w.click("view-menu",c);w.click(("mode",2usize),c);
        let before=view.read(c).inspection(c).2;w.input("禁止写入",c);assert_eq!(view.read(c).inspection(c).2,before);
    }).unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/reading-table.png"))
        .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.click("view-menu", c);
        w.click(("mode", 0usize), c);
        w.press("cmd-q", c);
        assert!(w.try_find("cancel-exit").is_some());
    })
    .unwrap();
    cx.capture_screenshot(handle)
        .unwrap()
        .save(format!("{output}/exit.png"))
        .unwrap();
    println!(
        "Passed rich native events: task/undo, language validation, code copy, link validation/apply/undo, tab overflow, reading immutability and exit guard"
    );
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
        let source = view.read(c).inspection(c).2;
        w.input("must not reach the covered document", c);
        w.press("cmd-b", c);
        assert_eq!(view.read(c).inspection(c).2, source);
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
