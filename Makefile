.PHONY: check fmt core-check test build dev preview render-check native-check native-build native-dev native-preview native-render-check

check: fmt core-check
	cargo clippy --locked --features render-check --all-targets -- -D warnings

fmt:
	cargo fmt --check

core-check:
	cargo clippy --locked --no-default-features --all-targets -- -D warnings
	cargo test --locked --no-default-features

test:
	cargo test --locked --no-default-features

build:
	cargo build --locked --release --bin markgo

dev:
	cargo run --locked --bin markgo

preview:
	cargo run --locked --bin markgo -- --preview

render-check:
	cargo run --locked --features render-check --example render_check -- build/gpui-evidence

# Compatibility aliases for existing Rust contributor scripts.
native-check: check
native-build:
	cargo build --locked --bin markgo
native-dev: dev
native-preview: preview
native-render-check: render-check
