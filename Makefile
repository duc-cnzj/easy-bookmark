.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -s -w ./ && goimports -w ./

.PHONY: build_race
build_race:
	CGO_ENABLED=1 go build -ldflags="-w -s" -race -o bookmark .

.PHONY: build
build:
	CGO_ENABLED=1 go build -ldflags="-w -s" -o bookmark .

.PHONY: build_linux_amd64
build_linux_amd64:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bookmark-linux-amd64 .

.PHONY: build_linux_arm64
build_linux_arm64:
	CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s -extldflags '-static'" -o bookmark-linux-arm64 .

.PHONY: build_drawin_amd64
build_drawin_amd64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bookmark-darwin-amd64 .

.PHONY: build_drawin_arm64
build_drawin_arm64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o bookmark-darwin-arm64 .

.PHONY: build_windows
build_windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bookmark.exe .