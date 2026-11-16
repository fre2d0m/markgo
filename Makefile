.PHONY: check fmt tidy vet staticcheck test race security build dev native-check native-build native-dev native-preview native-render-check

check: fmt tidy vet staticcheck test

fmt:
	@files=$$(git ls-files --cached --others --exclude-standard '*.go'); \
		unformatted=$$(gofmt -l $$files); \
		if [ -n "$$unformatted" ]; then echo "$$unformatted"; exit 1; fi

tidy:
	go mod tidy -diff

vet:
	go vet ./...
	go tool mygo vet ./cmd/markgo
	go tool mygo vet ./cmd/markgo-preview

staticcheck:
	go tool staticcheck ./...

test:
	CGO_ENABLED=0 go test -cover ./... -count=1

race:
	go test -race -p 1 ./... -count=1

security:
	go tool govulncheck ./...

build:
	go tool mygo build

dev:
	go tool mygo dev .

native-check:
	cd native && cargo fmt --check
	cd native && cargo clippy --locked --all-targets -- -D warnings
	cd native && cargo test --locked

native-build:
	cd native && cargo build --locked --features desktop --bin markgo

native-dev:
	cd native && cargo run --locked --features desktop --bin markgo

native-preview:
	cd native && cargo run --locked --features desktop --bin markgo -- --preview

native-render-check:
	cd native && cargo run --locked --features render-check --example render_check -- ../build/gpui-evidence
