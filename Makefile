.PHONY: build web test clean

build: web
	go build -trimpath -o remote-herdr ./cmd/remote-herdr

web:
	pnpm -C web install --frozen-lockfile
	pnpm -C web build

test:
	go vet ./...
	go test ./...
	pnpm -C web check

clean:
	rm -f remote-herdr
	find web/dist -mindepth 1 ! -name .gitkeep -delete
