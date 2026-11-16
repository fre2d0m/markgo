use super::{assets::icon, theme::Palette};
use gpui_kit::{
    base::motion::{self, Transition},
    component::{
        Disableable, Sizable,
        button::{Button, ButtonCustomVariant, ButtonVariants},
    },
    prelude::*,
    *,
};
use std::time::Duration;

#[derive(IntoElement)]
pub(super) struct IconButton {
    id: ElementId,
    name: String,
    button: Button,
    palette: Palette,
    disabled: bool,
}

impl IconButton {
    pub fn new(id: impl Into<ElementId>, name: &str, label: &str, palette: Palette) -> Self {
        let id = id.into();
        Self {
            button: Button::new(id.clone())
                .accessibility_label(label.to_owned())
                .tooltip(label.to_owned())
                .size(px(40.))
                .p(px(0.))
                .rounded(px(8.)),
            id,
            name: name.into(),
            palette,
            disabled: false,
        }
    }
    pub fn disabled(mut self, disabled: bool) -> Self {
        self.disabled = disabled;
        self.button = self.button.disabled(disabled);
        self
    }
    pub fn on_click(
        mut self,
        listener: impl Fn(&ClickEvent, &mut Window, &mut App) + 'static,
    ) -> Self {
        self.button = self.button.on_click(listener);
        self
    }
}

impl RenderOnce for IconButton {
    fn render(self, window: &mut Window, cx: &mut App) -> impl IntoElement {
        let channel =
            |name: &str| ElementId::NamedChild(self.id.clone().into(), name.to_owned().into());
        let state = window.use_keyed_state(channel("pointer"), cx, |_, _| (false, false));
        let (hovered, pressed) = *state.read(cx);
        let p = self.palette;
        let clear = Hsla {
            a: 0.,
            ..Hsla::from(rgb(p.hover))
        };
        let target = if self.disabled {
            clear
        } else if pressed {
            rgb(p.pressed).into()
        } else if hovered {
            rgb(p.hover).into()
        } else {
            clear
        };
        let color = motion::transition(
            channel("color"),
            target,
            Transition::new(Duration::from_millis(100)),
            window,
            cx,
        );
        let scale = motion::transition(
            channel("press"),
            if pressed && !self.disabled {
                0.96_f32
            } else {
                1.
            },
            Transition::new(Duration::from_millis(90)),
            window,
            cx,
        );
        let feedback_id = channel("feedback");
        let button = self
            .button
            .custom(
                ButtonCustomVariant::new(cx)
                    .color(color)
                    .foreground(rgb(p.ink).into())
                    .hover(color)
                    .active(color),
            )
            .child(
                icon(&self.name)
                    .with_size(px(18.))
                    .transform(Transformation::scale(size(scale, scale))),
            );
        // Toolkit tooltips own the inner button's hover listener. Observe the
        // fixed-size wrapper so feedback and tooltips can coexist safely.
        div()
            .id(feedback_id)
            .size(px(40.))
            .flex_shrink_0()
            .on_hover({
                let state = state.clone();
                move |hovered, w, c| {
                    state.update(c, |s, _| {
                        s.0 = *hovered;
                        if !hovered {
                            s.1 = false;
                        }
                    });
                    w.refresh();
                }
            })
            .on_mouse_down(MouseButton::Left, {
                let state = state.clone();
                move |_, w, c| {
                    state.update(c, |s, _| s.1 = true);
                    w.refresh();
                }
            })
            .on_mouse_up(MouseButton::Left, {
                let state = state.clone();
                move |_, w, c| {
                    state.update(c, |s, _| s.1 = false);
                    w.refresh();
                }
            })
            .on_mouse_up_out(MouseButton::Left, move |_, w, c| {
                state.update(c, |s, _| s.1 = false);
                w.refresh();
            })
            .child(button)
    }
}
