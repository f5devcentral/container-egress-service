GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GO_VERSION = 1.16

GITHUBACTIONTRIGGERNUMBER = 12

REGISTRY = f5devcentral
NAME = ces-controller
RELEASE_TAG = 0.6.1
COMMIT = git-$(shell git rev-parse --short HEAD)
DATE = $(shell date +"%Y-%m-%d_%H:%M:%S")
GOLDFLAGS = "-w -s -X github.com/kubeovn/$(NAME)/versions.COMMIT=$(COMMIT) -X github.com/kubeovn/$(NAME)/versions.VERSION=$(RELEASE_TAG) -X github.com/kubeovn/$(NAME)/versions.BUILDDATE=$(DATE)"

ARCH = amd64

.PHONY: build-go
build-go:
	go mod tidy
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(CURDIR)/dist/$(NAME) -ldflags $(GOLDFLAGS) -v ./cmd/ces

.PHONY: build-bin
build-bin:
	docker run --rm -e GOOS=linux -e GOCACHE=/tmp -e GOARCH=$(ARCH) -e GOPROXY=https://goproxy.cn \
		-u $(shell id -u):$(shell id -g) \
		-v $(CURDIR):/go/src/github.com/kubeovn/$(NAME):ro \
		-v $(CURDIR)/dist:/go/src/github.com/kubeovn/$(NAME)/dist/ \
		golang:$(GO_VERSION) /bin/bash -c '\
		cd /go/src/github.com/kubeovn/$(NAME) && \
		make build-go'

.PHONY: release
release:  build-go
	docker buildx build --platform linux/amd64 --build-arg ARCH=amd64 -t $(REGISTRY)/$(NAME):$(RELEASE_TAG) -o type=docker -f dist/Dockerfile dist/

.PHONY: tar
tar:
	docker save $(REGISTRY)/$(NAME):$(RELEASE_TAG) -o image.tar

.PHONY: lint
lint:
	@gofmt -d $(GOFILES_NOVENDOR)
	@gofmt -l $(GOFILES_NOVENDOR) | read && echo "Code differs from gofmt's style" 1>&2 && exit 1 || true
	@GOOS=linux go vet ./...
	@GOOS=linux gosec -exclude=G304,G402 ./...
