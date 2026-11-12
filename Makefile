.PHONY: check fmt tidy vet staticcheck test race security build dev

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
