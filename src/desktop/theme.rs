use gpui_kit::{
    component::{Theme, ThemeMode},
    *,
};

#[derive(Clone, Copy)]
pub struct Palette {
    pub id: &'static str,
    pub name: &'static str,
    pub paper: u32,
    pub rail: u32,
    pub hover: u32,
    pub pressed: u32,
    pub edge: u32,
    pub ink: u32,
    pub muted: u32,
    pub accent: u32,
    pub dark: bool,
}
macro_rules! palette {
    ($id:literal,$name:literal,$paper:literal,$rail:literal,$hover:literal,$pressed:literal,$edge:literal,$ink:literal,$muted:literal,$accent:literal,$dark:literal) => {
        Palette {
            id: $id,
            name: $name,
            paper: $paper,
            rail: $rail,
            hover: $hover,
            pressed: $pressed,
            edge: $edge,
            ink: $ink,
            muted: $muted,
            accent: $accent,
            dark: $dark,
        }
    };
}
pub const PALETTES: [Palette; 12] = [
    palette!(
        "paper", "素纸", 0xffffff, 0xf7f8fa, 0xeef0f3, 0xe2e5eb, 0xe7e9ee, 0x252932, 0x5d6571,
        0x414b5a, false
    ),
    palette!(
        "warm", "暖纸", 0xfbf8f2, 0xf3eee5, 0xeee8dd, 0xe4dccd, 0xe5ddcf, 0x38342e, 0x675e52,
        0x77603f, false
    ),
    palette!(
        "sakura", "樱花", 0xfff8fa, 0xf7edf1, 0xf0e1e7, 0xe8d3dc, 0xead8e0, 0x42343c, 0x715460,
        0x965471, false
    ),
    palette!(
        "sage",
        "鼠尾草",
        0xf7faf6,
        0xedf2eb,
        0xe3ecdf,
        0xd4e0d0,
        0xdce6d8,
        0x303c32,
        0x536553,
        0x526e50,
        false
    ),
    palette!(
        "mist", "晨雾", 0xf6f9fc, 0xedf2f7, 0xe1eaf3, 0xd1deeb, 0xd9e3ee, 0x2d3948, 0x536274,
        0x506e91, false
    ),
    palette!(
        "lavender",
        "薰衣草",
        0xfaf8fd,
        0xf1edf7,
        0xe9e2f2,
        0xdcd2e9,
        0xe4ddee,
        0x3c3547,
        0x665771,
        0x79628e,
        false
    ),
    palette!(
        "sand", "沙丘", 0xfaf7ef, 0xf1edde, 0xe9e2ce, 0xded3b6, 0xe4dcc7, 0x3e392b, 0x625a42,
        0x796638, false
    ),
    palette!(
        "pearl", "珍珠", 0xf6f7f7, 0xedefef, 0xe3e6e6, 0xd5dada, 0xdce1e1, 0x323b3c, 0x556163,
        0x536e70, false
    ),
    palette!(
        "graphite", "石墨", 0x191b20, 0x15171b, 0x252830, 0x303540, 0x343943, 0xe9ecf1, 0xa6adba,
        0xbcc6d7, true
    ),
    palette!(
        "ink", "夜墨", 0x18212c, 0x141b25, 0x253345, 0x304258, 0x35465b, 0xe4ecf5, 0xa4b5ca,
        0xa8c3e6, true
    ),
    palette!(
        "pine", "松林", 0x1b2521, 0x161f1b, 0x2a3830, 0x35483d, 0x394b40, 0xe4eee6, 0xa7baab,
        0xafcbb1, true
    ),
    palette!(
        "plum", "暮紫", 0x26202b, 0x201a25, 0x362c3e, 0x45384f, 0x4a3c54, 0xeee6f2, 0xbeabc8,
        0xcdb2da, true
    ),
];

impl Palette {
    pub fn apply(self, window: Option<&mut Window>, cx: &mut App) {
        Theme::change(
            if self.dark {
                ThemeMode::Dark
            } else {
                ThemeMode::Light
            },
            window,
            cx,
        );
        Theme::update(cx, |t| {
            t.font_size = px(13.);
            t.radius = px(8.);
            t.radius_lg = px(16.);
            t.window_radius = px(0.);
            t.focus_ring = false;
            let c = &mut t.colors;
            c.background = rgb(self.paper).into();
            c.foreground = rgb(self.ink).into();
            c.title_bar = rgb(self.rail).into();
            c.title_bar_border = rgb(self.edge).into();
            c.sidebar = rgb(self.rail).into();
            c.sidebar_foreground = rgb(self.ink).into();
            c.border = rgb(self.edge).into();
            c.input = rgb(self.paper).into();
            c.muted = rgb(self.rail).into();
            c.muted_foreground = rgb(self.muted).into();
            c.accent = rgb(self.hover).into();
            c.accent_foreground = rgb(self.ink).into();
            c.primary = rgb(self.accent).into();
            c.primary_foreground = rgb(if self.dark { self.rail } else { self.paper }).into();
            c.primary_hover = rgb(self.muted).into();
            c.primary_active = rgb(self.ink).into();
            c.button_primary = c.primary;
            c.button_primary_foreground = c.primary_foreground;
            c.button_primary_hover = c.primary_hover;
            c.button_primary_active = c.primary_active;
            c.button = rgb(self.paper).into();
            c.button_foreground = rgb(self.ink).into();
            c.button_hover = rgb(self.hover).into();
            c.button_active = rgb(self.pressed).into();
            c.secondary = rgb(self.hover).into();
            c.secondary_foreground = rgb(self.ink).into();
            c.secondary_hover = rgb(self.pressed).into();
            c.secondary_active = rgb(self.pressed).into();
            c.popover = rgb(self.paper).into();
            c.popover_foreground = rgb(self.ink).into();
            c.list = rgb(self.paper).into();
            c.list_hover = rgb(self.hover).into();
            c.selection = Hsla {
                a: 0.2,
                ..rgb(self.accent).into()
            };
            c.caret = rgb(self.accent).into();
            c.ring = rgb(self.accent).into();
        });
    }
}
