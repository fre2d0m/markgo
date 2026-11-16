//! Isolated UI events with native text shaping; GPU capture unless --events-only.
use gpui_kit::{
    AppContext, Bounds, EntityInputHandler, HeadlessAppContext, WindowBounds, WindowOptions, px,
    size, test::TestWindowExt,
};
use markgo::desktop::{self, Workbench, assets::Assets, theme::PALETTES};
use std::sync::Arc;

fn marker_interactions(output: &str) {
    let mut cx = context();
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
    cx.update_window(handle, |_, w, c| {
        w.input("#", c);
        assert_eq!(view.read(c).input_entity().read(c).editor.text(), "#");
        w.input("#", c);
        assert_eq!(view.read(c).input_entity().read(c).editor.text(), "##");
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/typed-hashes.png"));
    cx.update_window(handle, |_, w, c| {
        w.input(" ", c);
        let writer = view.read(c).input_entity();
        assert_eq!(writer.read(c).editor.text(), "");
        assert_eq!(writer.read(c).editor.heading_at(0), 2);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/empty-heading.png"));
    cx.update_window(handle, |_, w, c| {
        w.press("backspace", c);
        assert_eq!(view.read(c).inspection(c).2, "");
        w.press(&shortcut("z"), c);
        assert_eq!(view.read(c).inspection(c).2, "## ");
        w.input("标题", c);
        w.click("new-tab", c);
        w.input("另一个文档", c);
        w.click(("activate", 1usize), c);
        assert_eq!(view.read(c).inspection(c).2, "## 标题");
        w.click(("activate", 2usize), c);
        assert_eq!(view.read(c).inspection(c).2, "另一个文档");
        w.click("insert-format", c);
        w.press("escape", c);
        w.click("view-menu", c);
        w.press("escape", c);
        w.click("appearance", c);
        w.press("escape", c);
    })
    .unwrap();
    println!(
        "Passed typed hashes, heading conversion/backspace/undo, pointer tabs and toolbar dialogs"
    );
}

fn icon_assets(output: &str) {
    use gpui_kit::component::Sizable;
    use gpui_kit::{
        AssetSource, Context, IntoElement, ParentElement, Render, Styled, Window, div, rgb,
    };
    const ICONS: [&str; 38] = [
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
        "window-close",
    ];
    let rasterizer = gpui_kit::SvgRenderer::new(Arc::new(Assets));
    for name in ICONS {
        let bytes = Assets.load(&format!("markgo/{name}.svg")).unwrap().unwrap();
        let bitmap = rasterizer.render_single_frame(&bytes, 1.).unwrap();
        assert!(
            bitmap
                .as_bytes(0)
                .unwrap()
                .as_chunks::<4>()
                .0
                .iter()
                .any(|p| p[3] > 0),
            "icon rasterized blank: {name}"
        );
    }
    for name in markgo::desktop::assets::WINDOW_ICONS {
        assert!(Assets.load(&format!("icons/{name}.svg")).unwrap().is_some());
    }
    if events_only() {
        println!("Validated embedded icons and non-empty rasterized glyphs");
        return;
    }
    struct Gallery;
    impl Render for Gallery {
        fn render(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
            div()
                .size_full()
                .bg(rgb(0xffffff))
                .text_color(rgb(0x252932))
                .flex()
                .flex_wrap()
                .content_start()
                .children(ICONS.iter().map(|name| {
                    div()
                        .w(px(120.))
                        .h(px(72.))
                        .flex()
                        .flex_col()
                        .items_center()
                        .justify_center()
                        .gap(px(8.))
                        .child(markgo::desktop::assets::icon(name).with_size(px(24.)))
                        .child(div().text_size(px(10.)).child(*name))
                }))
        }
    }
    let mut cx = context();
    cx.update(desktop::init);
    let (handle, _) = cx
        .update(|c| {
            gpui_kit::open_window(
                WindowOptions {
                    window_bounds: Some(WindowBounds::Windowed(Bounds {
                        origin: Default::default(),
                        size: size(px(720.), px(504.)),
                    })),
                    show: false,
                    focus: false,
                    ..Default::default()
                },
                c,
                |_, c| c.new(|_| Gallery),
            )
        })
        .unwrap();
    cx.update_window(handle, |_, w, c| w.render_frame(c))
        .unwrap();
    let pixels = cx.capture_screenshot(handle).unwrap();
    let scale = pixels.width() as f32 / 720.;
    for (index, name) in ICONS.iter().enumerate() {
        let x = ((index % 6) as f32 * 120. + 48.) * scale;
        let y = ((index / 6) as f32 * 72. + 15.) * scale;
        let mut ink = 0;
        for dy in 0..(24. * scale) as u32 {
            for dx in 0..(24. * scale) as u32 {
                let p = pixels.get_pixel(x as u32 + dx, y as u32 + dy);
                ink += usize::from(p[0] < 180 && p[1] < 180 && p[2] < 180);
            }
        }
        assert!(ink > 4, "icon rendered blank: {name}");
    }
    pixels.save(format!("{output}/icons.png")).unwrap();
    println!("Validated embedded icons and non-empty painted glyphs");
}

fn events_only() -> bool {
    std::env::args().any(|arg| arg == "--events-only")
}

fn workbench_surfaces(output: &str) {
    let mut cx = context();
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
        view.update(c, |s, _| s.menu_fixture());
        w.render_frame(c);
    })
    .unwrap();
    for (palette, name) in [(0, "paper"), (2, "sakura"), (8, "graphite")] {
        cx.update_window(handle, |_, w, c| {
            view.update(c, |s, c| s.set_palette(palette, w, c));
            w.render_frame(c);
            w.click("all-tabs", c);
            let first = w.find(("all-tab", 1usize)).bounds();
            for id in 1usize..=3 {
                let row = w.find(("all-tab", id)).bounds();
                assert_eq!(row.origin.x, first.origin.x);
                assert_eq!(row.size.height, px(56.));
                assert!(row.size.width >= px(320.));
                assert!(row.bottom() < px(612.));
            }
        })
        .unwrap();
        save_screenshot(&mut cx, handle, format!("{output}/tabs-{name}.png"));
        cx.update_window(handle, |_, w, c| {
            w.press("escape", c);
            w.click("view-menu", c);
            let first = w.find(("mode", 0usize)).bounds();
            for id in 0usize..=2 {
                let row = w.find(("mode", id)).bounds();
                assert_eq!(row.origin.x, first.origin.x);
                assert_eq!(row.size.height, px(40.));
                assert!(row.size.width >= px(280.));
            }
            assert_eq!(w.find("menu-open").bounds().origin.x, first.origin.x);
        })
        .unwrap();
        save_screenshot(&mut cx, handle, format!("{output}/view-{name}.png"));
        cx.update_window(handle, |_, w, c| {
            w.press("escape", c);
            w.click("insert-format", c);
        })
        .unwrap();
        save_screenshot(&mut cx, handle, format!("{output}/format-{name}.png"));
        cx.update_window(handle, |_, w, c| w.press("escape", c))
            .unwrap();
    }
    println!("Passed compact menu row geometry, selection and theme surfaces");
}

fn workbench_motion(output: &str) {
    use gpui_kit::{InputEvent, MouseButton, MouseDownEvent, MouseMoveEvent, MouseUpEvent, point};
    use std::time::Duration;
    let mut cx = context();
    cx.update(desktop::init);
    cx.update(|c| c.set_reduce_motion(false));
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
    let start = cx
        .update_window(handle, |_, w, c| {
            w.render_frame(c);
            w.click("view-menu", c);
            w.find(("mode", 0usize)).bounds().origin.y
        })
        .unwrap();
    cx.advance_clock(Duration::from_millis(65));
    let middle = cx
        .update_window(handle, |_, w, c| {
            w.render_frame(c);
            w.find(("mode", 0usize)).bounds().origin.y
        })
        .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/menu-enter-mid.png"));
    cx.advance_clock(Duration::from_millis(100));
    let end = cx
        .update_window(handle, |_, w, c| {
            w.render_frame(c);
            w.find(("mode", 0usize)).bounds().origin.y
        })
        .unwrap();
    assert!(
        start < middle && middle < end,
        "popover must move through intermediate frames: {start:?}, {middle:?}, {end:?}"
    );
    assert!(end - start <= px(4.1));
    save_screenshot(&mut cx, handle, format!("{output}/menu-shown.png"));
    cx.update_window(handle, |_, w, c| {
        w.press("escape", c);
        w.input("关闭后继续写作", c);
        assert_eq!(view.read(c).inspection(c).2, "关闭后继续写作");
    })
    .unwrap();
    cx.advance_clock(Duration::from_millis(45));
    cx.update_window(handle, |_, w, c| w.render_frame(c))
        .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/menu-exit-mid.png"));
    // The departing surface must not block its toolbar trigger; reverse it.
    cx.update_window(handle, |_, w, c| w.click("view-menu", c))
        .unwrap();
    cx.advance_clock(Duration::from_millis(150));
    cx.update_window(handle, |_, w, c| {
        w.render_frame(c);
        assert!(w.try_find(("mode", 0usize)).is_some());
        w.press("escape", c);
    })
    .unwrap();
    cx.advance_clock(Duration::from_millis(100));
    cx.update_window(handle, |_, w, c| {
        w.render_frame(c);
        assert!(w.try_find(("mode", 0usize)).is_none());
    })
    .unwrap();
    let target = cx
        .update_window(handle, |_, w, c| {
            w.hover("bold", c);
            w.find("bold").bounds()
        })
        .unwrap();
    cx.advance_clock(Duration::from_millis(60));
    save_screenshot(&mut cx, handle, format!("{output}/button-hover-mid.png"));
    cx.update_window(handle, |_, w, c| {
        w.dispatch_event(
            MouseDownEvent {
                button: MouseButton::Left,
                position: target.center(),
                click_count: 1,
                ..Default::default()
            }
            .to_platform_input(),
            c,
        );
        w.render_frame(c);
    })
    .unwrap();
    cx.advance_clock(Duration::from_millis(100));
    save_screenshot(&mut cx, handle, format!("{output}/button-pressed.png"));
    cx.update_window(handle, |_, w, c| {
        assert_eq!(w.find("bold").bounds(), target);
        let outside = point(px(180.), px(600.));
        w.dispatch_event(
            MouseMoveEvent {
                position: outside,
                pressed_button: Some(MouseButton::Left),
                ..Default::default()
            }
            .to_platform_input(),
            c,
        );
        w.render_frame(c);
        w.dispatch_event(
            MouseUpEvent {
                button: MouseButton::Left,
                position: outside,
                click_count: 1,
                ..Default::default()
            }
            .to_platform_input(),
            c,
        );
        w.render_frame(c);
        assert_eq!(view.read(c).inspection(c).2, "关闭后继续写作");
    })
    .unwrap();
    cx.advance_clock(Duration::from_millis(120));
    save_screenshot(&mut cx, handle, format!("{output}/button-cancelled.png"));
    cx.update(|c| c.set_reduce_motion(true));
    cx.update_window(handle, |_, w, c| {
        w.click("view-menu", c);
        assert_eq!(w.find(("mode", 0usize)).bounds().origin.y, end);
        w.press("escape", c);
        assert!(w.try_find(("mode", 0usize)).is_none());
    })
    .unwrap();
    println!(
        "Passed popover intermediate frames, reversible closing, immediate writing focus, press/drag cancellation and reduced motion"
    );
}
fn shortcut(key: &str) -> String {
    let modifier = if cfg!(target_os = "macos") {
        "cmd"
    } else {
        "ctrl"
    };
    format!("{modifier}-{key}")
}
fn context() -> HeadlessAppContext {
    // Windows' true-headless platform deliberately uses fake font metrics.
    // Initialize DirectWrite, but every test window still belongs to TestPlatform.
    let platform = gpui_kit::platform::current_platform(!cfg!(target_os = "windows"));
    let mut cx = if events_only() {
        HeadlessAppContext::with_asset_source(platform.text_system(), Arc::new(Assets))
    } else {
        HeadlessAppContext::with_platform(
            platform.text_system(),
            Arc::new(Assets),
            gpui_kit::platform::current_headless_renderer,
        )
    };
    // Still frames and semantic events settle instantly. Dedicated motion
    // checks opt back in and advance this context's simulated clock.
    cx.update(|cx| cx.set_reduce_motion(true));
    cx
}
fn save_screenshot(cx: &mut HeadlessAppContext, handle: gpui_kit::AnyWindowHandle, path: String) {
    // Advancing the controlled clock schedules a frame, but does not draw it.
    // Capture the sampled state, including after a hover or press transition.
    cx.update_window(handle, |_, w, c| w.render_frame(c))
        .unwrap();
    if !events_only() {
        cx.capture_screenshot(handle)
            .expect("Real GPU renderer required")
            .save(path)
            .unwrap();
    }
}

fn main() {
    let output = std::env::args()
        .skip(1)
        .find(|arg| !arg.starts_with("--"))
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
        let mut cx = context();
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
        save_screenshot(&mut cx, handle, format!("{output}/{name}.png"));
        println!(
            "Validated {name} ({width} × {height}, {})",
            PALETTES[palette].id
        );
    }
    interactions(&output);
    rich_interactions(&output);
    marker_interactions(&output);
    icon_assets(&output);
    workbench_surfaces(&output);
    workbench_motion(&output);
}

fn rich_interactions(output: &str) {
    let mut cx = context();
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
        w.press(&shortcut("z"), c);
        assert_eq!(view.read(c).inspection(c).2, source);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/rich-writing.png"));
    cx.update_window(handle, |_, w, c| {
        w.click(("code-language", 0usize), c);
    })
    .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press(&shortcut("a"), c);
        w.input("bad language!", c);
        w.click("apply-entry", c);
        assert_eq!(
            view.read(c).inspection(c).2,
            source,
            "invalid language is non-destructive"
        );
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/code-error.png"));
    cx.update_window(handle, |_, w, c| {
        w.press("escape", c);
        w.click(("code-language", 0usize), c);
    })
    .unwrap();
    cx.update_window(handle, |_, w, c| {
        w.press(&shortcut("a"), c);
        w.input("python", c);
        w.click("apply-entry", c);
        assert!(view.read(c).inspection(c).2.contains("```python"));
        w.press(&shortcut("z"), c);
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
    save_screenshot(&mut cx, handle, format!("{output}/link-error.png"));
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
        w.press(&shortcut("z"), c);
        assert_eq!(view.read(c).inspection(c).2, source);
        w.click("all-tabs", c);
        assert!(w.try_find(("all-tab", 1usize)).is_some());
        w.click(("all-tab", 1usize), c);
        view.update(c, |s, c| s.set_palette(8, w, c));
        w.render_frame(c);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/rich-graphite.png"));
    cx.update_window(handle,|_,w,c|{
        view.update(c,|s,c|s.set_palette(0,w,c));
        let entity=view.read(c).input_entity();entity.update(c,|writer,c|{writer.editor=markgo::editing::Editor::new("# 阅读也轻松\n\n| 章节 | 进展 |\n| --- | --- |\n| 开始 | 已写下第一句 |\n| 继续 | 留给明天 |\n\n**写作**是慢慢整理自己的想法。\n");writer.mode(false,false,w,c);});w.render_frame(c);w.click("view-menu",c);w.click(("mode",2usize),c);
        let before=view.read(c).inspection(c).2;w.input("禁止写入",c);assert_eq!(view.read(c).inspection(c).2,before);
    }).unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/reading-table.png"));
    cx.update_window(handle, |_, w, c| {
        w.click("view-menu", c);
        w.click(("mode", 0usize), c);
        w.press("cmd-q", c);
        assert!(w.try_find("cancel-exit").is_some());
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/exit.png"));
    println!(
        "Passed rich native events: task/undo, language validation, code copy, link validation/apply/undo, tab overflow, reading immutability and exit guard"
    );
}

fn interactions(output: &str) {
    let mut cx = context();
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
        w.press(&shortcut("z"), c);
        assert_eq!(view.read(c).inspection(c).2, "");
        w.press(&shortcut("shift-z"), c);
        assert_eq!(view.read(c).inspection(c).2, "中文 👩‍💻 hello");
        let before = view.read(c).inspection(c);
        w.click(("file", 1usize), c);
        w.click(("file", 4usize), c);
        let after = view.read(c).inspection(c);
        assert_eq!(
            before, after,
            "tab switching retains source, caret and history"
        );
        w.press(&shortcut("w"), c);
        assert!(w.try_find("cancel-close").is_some());
        w.click("cancel-close", c);
        assert_eq!(view.read(c).inspection(c).0, 4);
        w.press(&shortcut("a"), c);
        w.press("backspace", c);
        w.input("/", c);
        assert!(w.try_find(("slash", 1usize)).is_some());
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/slash.png"));
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
        w.press(&shortcut("z"), c);
        assert_eq!(view.read(c).inspection(c).2, "# 真正的标题");
        w.click("appearance", c);
        for i in 0usize..12 {
            w.click(("theme", i), c);
            assert_eq!(view.read(c).inspection(c).2, "# 真正的标题");
        }
        w.click(("theme", 2usize), c);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/appearance.png"));
    cx.update_window(handle, |_, w, c| {
        w.click("close-panel", c);
        w.click("search", c);
        w.input("标题", c);
        w.click("toggle-replace", c);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/search.png"));
    cx.update_window(handle, |_, w, c| {
        w.click("close-search", c);
        w.press(&shortcut("w"), c);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/close-guard.png"));
    cx.update_window(handle, |_, w, c| {
        w.click("cancel-close", c);
        view.update(c, |v, c| v.render_error_fixture(w, c));
        w.render_frame(c);
        let source = view.read(c).inspection(c).2;
        w.input("must not reach the covered document", c);
        w.press(&shortcut("b"), c);
        assert_eq!(view.read(c).inspection(c).2, source);
    })
    .unwrap();
    save_screenshot(&mut cx, handle, format!("{output}/error.png"));
    println!(
        "Passed native event dispatch: input, undo/redo, tabs, close guard, slash, composition, geometry and 12 themes"
    );
}
