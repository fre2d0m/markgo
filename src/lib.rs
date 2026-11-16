//! MarkGo's source-preserving native model and optional GPUI desktop client.
//! Source/history and text-service contracts can be tested without a renderer.

pub mod document;
pub mod editing;
pub mod markdown;
pub mod preferences;
pub mod projection;
pub mod search;
pub mod session;
pub mod storage;

#[cfg(feature = "desktop")]
extern crate gpui_kit as gpui;
#[cfg(feature = "desktop")]
pub mod desktop;
