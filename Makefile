PROJECT_NAME      ?= fix
GO                ?= $(shell which go)
GH                ?= $(shell which gh)
GIT_UPDATE_INDEX  := $(shell git update-index --refresh > /dev/null 2>&1)
GIT_REVISION      ?= $(shell git rev-parse HEAD)
GIT_VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

export GOPRIVATE=github.com/artex-io

GOENV_GOOS               := $(shell $(GO) env GOOS)
GOENV_GOARCH             := $(shell $(GO) env GOARCH)
GOENV_GOARM              := $(shell $(GO) env GOARM)
GOENV_GOPRIVATE          := $(shell go env GOPRIVATE)
CGO_ENABLED              ?= 1
GOOS                     ?= $(GOENV_GOOS)
GOARCH                   ?= $(GOENV_GOARCH)
GOARM                    ?= $(GOENV_GOARM)
GOPRIVATE                ?= $(GOENV_GOPRIVATE)

ifneq ($(GH),)
GH_GIT_CREDS             := $(shell echo url=https://github.com/artex-io/$(PROJECT_NAME) | gh auth git-credential get)
endif # $(GH)
GOPRIVATE_GITHUB_USER    ?= $(subst username=,,$(filter username=%,$(GH_GIT_CREDS)))
GOPRIVATE_GITHUB_TOKEN   ?= $(subst password=,,$(filter password=%,$(GH_GIT_CREDS)))
GO_BUILD_SRC             := $(shell find . -name \*.go -type f) go.mod go.sum cmd/init/config/templates/*
GO_BUILD_EXTLDFLAGS      ?=
GO_BUILD_TAGS            ?=
GO_BUILD_TARGET_DEPS     :=
GO_BUILD_FLAGS           := -trimpath
GO_BUILD_LDFLAGS_OPTIMS  ?= -s -w

ifeq ($(GOOS)/$(GOARCH),$(GOENV_GOOS)/$(GOENV_GOARCH))
GO_BUILD_TARGET          ?= dist/$(PROJECT_NAME)
GO_BUILD_VERSION_TARGET  ?= dist/$(PROJECT_NAME)-$(GIT_VERSION)
else
ifeq ($(GOARCH),arm)
GO_BUILD_TARGET          ?= dist/$(PROJECT_NAME)-$(GOOS)-$(GOARCH)v$(GOARM)
GO_BUILD_VERSION_TARGET  := dist/$(PROJECT_NAME)-$(GIT_VERSION)-$(GOOS)-$(GOARCH)v$(GOARM)
else
GO_BUILD_TARGET          ?= dist/$(PROJECT_NAME)-$(GOOS)-$(GOARCH)
GO_BUILD_VERSION_TARGET  := dist/$(PROJECT_NAME)-$(GIT_VERSION)-$(GOOS)-$(GOARCH)
endif # ($(GOARCH),arm)
endif # ($(GOOS)/$(GOARCH),$(GOENV_GOOS)/$(GOENV_GOARCH))

ifneq ($(DEBUG),)
GO_BUILD_FLAGS            = -gcflags="all=-N -l"
GO_BUILD_LDFLAGS_OPTIMS   =
endif # $(DEBUG)

GO_BUILD_FLAGS_TARGET   := .go-build-flags
GO_CROSSBUILD_LINUX_AMD64_TARGET   := dist/fix-$(GIT_VERSION)-linux-amd64
#GO_CROSSBUILD_LINUX_ARM_TARGET     := dist/fix-$(GIT_VERSION)-linux-arm
GO_CROSSBUILD_LINUX_ARM64_TARGET   := dist/fix-$(GIT_VERSION)-linux-arm64
GO_CROSSBUILD_DARWIN_AMD64_TARGET  := dist/fix-$(GIT_VERSION)-darwin-amd64
GO_CROSSBUILD_DARWIN_ARM64_TARGET  := dist/fix-$(GIT_VERSION)-darwin-arm64
GO_CROSSBUILD_WINDOWS_AMD64_TARGET := dist/fix-$(GIT_VERSION)-windows-amd64.exe
GO_CROSSBUILD_WINDOWS_ARM64_TARGET := dist/fix-$(GIT_VERSION)-windows-arm64.exe

GO_CROSSBUILD_TARGETS  = $(GO_CROSSBUILD_LINUX_AMD64_TARGET)
GO_CROSSBUILD_TARGETS += $(GO_CROSSBUILD_LINUX_ARM_TARGET)
GO_CROSSBUILD_TARGETS += $(GO_CROSSBUILD_LINUX_ARM64_TARGET)
GO_CROSSBUILD_TARGETS += $(GO_CROSSBUILD_DARWIN_AMD64_TARGET)
GO_CROSSBUILD_TARGETS += $(GO_CROSSBUILD_DARWIN_ARM64_TARGET)
GO_CROSSBUILD_TARGETS += $(GO_CROSSBUILD_WINDOWS_AMD64_TARGET)
GO_CROSSBUILD_TARGETS += $(GO_CROSSBUILD_WINDOWS_ARM64_TARGET)

ZIG_CROSSBUILD_MACOS_FRAMEWORK ?= /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/System/Library/Frameworks
ZIG_CROSSBUILD_MACOS_LIB       ?= /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/lib

GO_BUILD_EXTLDFLAGS     := $(strip $(GO_BUILD_EXTLDFLAGS))
GO_BUILD_TAGS           := $(strip $(GO_BUILD_TAGS))
GO_BUILD_TARGET_DEPS    := $(strip $(GO_BUILD_TARGET_DEPS))
GO_BUILD_FLAGS          := $(strip $(GO_BUILD_FLAGS))
GO_BUILD_LDFLAGS_OPTIMS := $(strip $(GO_BUILD_LDFLAGS_OPTIMS))
GO_BUILD_LDFLAGS        := -ldflags '$(GO_BUILD_LDFLAGS_OPTIMS) -X sylr.dev/fix/cmd.Version=$(GIT_VERSION) -extldflags "$(GO_BUILD_EXTLDFLAGS)"'

GO_TOOLS_GOLANGCI_LINT ?= $(shell $(GO) env GOPATH)/bin/golangci-lint

DOCKER_BUILD_IMAGE      ?= ghcr.io/artex-io/fix
DOCKER_BUILD_VERSION    ?= $(GIT_VERSION)
DOCKER_BUILD_GO_VERSION ?= 1.20
DOCKER_BUILD_LABELS      = --label org.opencontainers.image.title=fix
DOCKER_BUILD_LABELS     += --label org.opencontainers.image.description="fix client"
DOCKER_BUILD_LABELS     += --label org.opencontainers.image.url="https://github.com/artex-io/fix"
DOCKER_BUILD_LABELS     += --label org.opencontainers.image.source="https://github.com/artex-io/fix"
DOCKER_BUILD_LABELS     += --label org.opencontainers.image.revision=$(GIT_REVISION)
DOCKER_BUILD_LABELS     += --label org.opencontainers.image.version=$(GIT_VERSION)
DOCKER_BUILD_LABELS     += --label org.opencontainers.image.created=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
DOCKER_BUILD_BUILD_ARGS ?= --build-arg=GO_VERSION=$(DOCKER_BUILD_GO_VERSION)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GIT_REVISION=$(GIT_REVISION)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GIT_VERSION=$(GIT_VERSION)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GO_BUILD_EXTLDFLAGS=-static
DOCKER_BUILD_BUILD_ARGS += --build-arg=GO_BUILD_LDFLAGS_OPTIMS="-linkmode external"
DOCKER_BUILDX_PLATFORMS ?= linux/amd64,linux/arm64
DOCKER_BUILDX_CACHE     ?= /tmp/.buildx-cache

ifeq ($(CI),true)
DOCKER_BUILD_BUILD_ARGS += --cache-to=type=registry,mode=max,ref=$(DOCKER_BUILD_IMAGE)-build-cache
DOCKER_BUILD_BUILD_ARGS += --cache-from=type=registry,ref=$(DOCKER_BUILD_IMAGE)-build-cache
DOCKER_BUILD_BUILD_ARGS += --progress=plain
DOCKER_BUILD_BUILD_ARGS += --build-arg=GOPRIVATE=$(GOPRIVATE)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GOPRIVATE_GITHUB_USER=$(GOPRIVATE_GITHUB_USER)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GOPRIVATE_GITHUB_TOKEN=$(GOPRIVATE_GITHUB_TOKEN)
else
DOCKER_BUILDX_CACHE_DIR ?= /tmp/.buildx-cache
DOCKER_BUILD_BUILD_ARGS += --cache-to=type=local,dest=$(DOCKER_BUILDX_CACHE_DIR)
DOCKER_BUILD_BUILD_ARGS += --cache-from=type=local,src=$(DOCKER_BUILDX_CACHE_DIR)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GOPRIVATE=$(GOPRIVATE)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GOPRIVATE_GITHUB_USER=$(GOPRIVATE_GITHUB_USER)
DOCKER_BUILD_BUILD_ARGS += --build-arg=GOPRIVATE_GITHUB_TOKEN=$(GOPRIVATE_GITHUB_TOKEN)
endif

# ------------------------------------------------------------------------------

.PHONY: all build crossbuild crossbuild-checksums .FORCE

build: $(GO_BUILD_VERSION_TARGET) $(GO_BUILD_TARGET)

all: crossbuild crossbuild-checksums

install:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) install -tags $(GO_BUILD_TAGS) $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS)

$(GO_BUILD_FLAGS_TARGET) : .FORCE
	@(echo "GO_VERSION=$(shell $(GO) version)"; \
	  echo "GO_GOOS=$(GOOS)"; \
	  echo "GO_GOARCH=$(GOARCH)"; \
	  echo "GO_GOARM=$(GOARM)"; \
	  echo "GO_BUILD_TAGS=$(GO_BUILD_TAGS)"; \
	  echo "GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)"; \
	  echo 'GO_BUILD_LDFLAGS=$(subst ','\'',$(GO_BUILD_LDFLAGS))'; \
	  echo "CGO_ENABLED=$(CGO_ENABLED)") \
	    | cmp -s - $@ \
	        || (echo "GO_VERSION=$(shell $(GO) version)"; \
	            echo "GO_GOOS=$(GOOS)"; \
	            echo "GO_GOARCH=$(GOARCH)"; \
	            echo "GO_GOARM=$(GOARM)"; \
	            echo "GO_BUILD_TAGS=$(GO_BUILD_TAGS)"; \
	            echo "GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)"; \
	            echo 'GO_BUILD_LDFLAGS=$(subst ','\'',$(GO_BUILD_LDFLAGS))'; \
	            echo "CGO_ENABLED=$(CGO_ENABLED)") > $@

$(GO_BUILD_TARGET): $(GO_BUILD_VERSION_TARGET)
	@(test -e $@ && unlink $@) || true
	@mkdir -p $$(dirname $@)
	@ln $< $@

$(GO_BUILD_VERSION_TARGET): $(GO_BUILD_SRC) $(GO_GENERATE_TARGET) $(GO_BUILD_FLAGS_TARGET) | $(GO_BUILD_TARGET_DEPS)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(GO) build -tags $(GO_BUILD_TAGS) $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

crossbuild: $(GO_BUILD_VERSION_TARGET) $(GO_CROSSBUILD_TARGETS)

$(GO_CROSSBUILD_LINUX_AMD64_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux" CXX="zig c++ -target x86_64-linux" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_LINUX_ARM_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm CC="zig cc -target arm-linux-gnueabihf" CXX="zig c++ -target arm-linux-gnueabihf" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_LINUX_ARM64_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 CC="zig cc -target aarch64-linux" CXX="zig c++ -target aarch64-linux" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_DARWIN_AMD64_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 CC="clang -target x86-64-apple-darwin -F$(ZIG_CROSSBUILD_MACOS_FRAMEWORK)" CXX="clang++ -target x86-64-apple-darwin -F$(ZIG_CROSSBUILD_MACOS_FRAMEWORK)" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_DARWIN_ARM64_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 CC="clang -target arm64-apple-darwin -F$(ZIG_CROSSBUILD_MACOS_FRAMEWORK)" CXX="clang++ -target arm64-apple-darwin -F$(ZIG_CROSSBUILD_MACOS_FRAMEWORK)" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_WINDOWS_AMD64_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 CC="zig cc -target x86_64-windows" CXX="zig c++ -target x86_64-windows" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_WINDOWS_ARM64_TARGET): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=arm64 CC="zig cc -target aarch64-windows" CXX="zig c++ -target aarch64-windows" $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

crossbuild-checksums: dist/checksums

dist/checksums : $(GO_CROSSBUILD_TARGETS)
	cd dist && shasum -a 256 $(PROJECT_NAME)-*-* > checksums

# -- go mod --------------------------------------------------------------------

.PHONY: go-mod-verify go-mod-tidy

go-mod-verify:
	$(GO) mod download
	git diff --quiet go.* || git diff --exit-code go.* || exit 1

go-mod-tidy:
	$(GO) mod download
	$(GO) mod tidy

# ------------------------------------------------------------------------------

test:
	$(GO) test -tags acceptor,validator ./...

lint: $(GO_TOOLS_GOLANGCI_LINT)
	$(GO_TOOLS_GOLANGCI_LINT) run

# -- tools ---------------------------------------------------------------------

.PHONY: tools

tools: $(GO_TOOLS_GOLANGCI_LINT)

$(GO_TOOLS_GOLANGCI_LINT):
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.36.0

# -- docker --------------------------------------------------------------------

.PHONY: docker-build docker-push docker-inspect

docker-build:
	docker buildx build . -f Dockerfile \
		-t $(DOCKER_BUILD_IMAGE):$(DOCKER_BUILD_VERSION) \
		--platform=$(DOCKER_BUILDX_PLATFORMS) \
		$(DOCKER_BUILD_BUILD_ARGS) \
		$(DOCKER_BUILD_LABELS) \
		--load

docker-push:
	@docker buildx build . -f Dockerfile \
		-t $(DOCKER_BUILD_IMAGE):$(DOCKER_BUILD_VERSION) \
		--platform=$(DOCKER_BUILDX_PLATFORMS) \
		$(DOCKER_BUILD_BUILD_ARGS) \
		$(DOCKER_BUILD_LABELS) \
		--push

docker-inspect:
	@docker buildx imagetools inspect $(DOCKER_BUILD_IMAGE):$(DOCKER_BUILD_VERSION)

docker-version:
	@echo $(DOCKER_BUILD_VERSION)

docker-image-exists:
	docker manifest inspect $(DOCKER_BUILD_IMAGE):$(DOCKER_BUILD_VERSION) > /dev/null

# -- helm  ---------------------------------------------------------------------

.PHONY: helm-chart-test

helm-chart-test:
	@contribs/helm-chart-test
