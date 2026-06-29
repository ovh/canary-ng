BINARY := canary-ng
GOOS := linux
GOARCH := amd64
APPVERSION := $(shell cat ./VERSION)
GOVERSION := $(shell go version | awk '{print $$3}')
GITCOMMIT := $(shell git log -1 --oneline | awk '{print $$1}')
LDFLAGS = -X main.AppVersion=${APPVERSION} -X main.GoVersion=${GOVERSION} -X main.GitCommit=${GITCOMMIT}

.PHONY: clean test test-e2e

build:
	(cd src && CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o ../bin/${BINARY} cmd/${BINARY}/main.go)

release: build
	(cd bin && tar czf ${BINARY}-${APPVERSION}-${GOOS}-${GOARCH}.tar.gz ${BINARY})

test:
	(cd src && go test -cover driver/*)

test-e2e:
	docker compose up -d --build --wait
	(cd src && go test -tags e2e -count=1 ./driver/...); status=$$?; \
		docker compose down -v; \
		exit $$status

clean:
	rm -rf bin
