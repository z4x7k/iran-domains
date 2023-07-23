clean:
	go clean -r -cache -testcache -modcache
.PHONY: clean

tidy:
	go mod tidy -v -x
.PHONY: tidy

build-clean: clean build
.PHONY: build-clean

jet:
	go run github.com/go-jet/jet/v2/cmd/jet@latest -source=sqlite -dsn=./domains.db -path=./db/gen
.PHONY: jet

test:
	go test -trimpath -buildvcs=false -ldflags '-extldflags "-static" -s -w -buildid=' -race -failfast -vet=all -covermode=atomic -coverprofile=coverage.out -v ./...
.PHONY: test

ifndef app_version
app_version := dev
endif
build:
	rm -rf ./bin
	mkdir -p ./bin
	go build --tags 'urfave_cli_no_docs' -trimpath -buildvcs=false -ldflags "-extldflags '-static' -s -w -buildid='' -X 'main.AppVersion=${app_version}' -X 'main.AppCompileTime=$(shell date -Iseconds)'" -o ./bin/bot .
.PHONY: build

outdated-indirect:
	go list -u -m -f '{{if and .Update .Indirect}}{{.}}{{end}}' all
.PHONY: outdated-indirect

outdated-direct:
	go list -u -m -f '{{if and .Update (not .Indirect)}}{{.}}{{end}}' all
.PHONY: outdated-direct

outdated-all: outdated-direct outdated-indirect
.PHONY: outdated-all
