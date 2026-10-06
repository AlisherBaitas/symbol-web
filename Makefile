.PHONY: run build test vet check fuzz audit

run:
	go run ./cmd/web

build:
	go build -o bin/symbol-web ./cmd/web

test:
	go test -race -timeout 90s ./...

vet:
	go vet ./...

check: vet
	node --check static/js/studio.js
	test -z "$$(gofmt -l cmd/web/main.go internal/ai/rules.go internal/ai/service.go internal/ai/service_test.go internal/ascii/safe_renderer.go internal/ascii/safe_renderer_test.go internal/handlers/web.go internal/handlers/web_test.go)"

fuzz:
	go test ./internal/ascii -run '^$$' -fuzz=FuzzRender -fuzztime=10s -parallel=4
	go test ./internal/handlers -run '^$$' -fuzz=FuzzAPI -fuzztime=10s -parallel=4

audit: check test build
