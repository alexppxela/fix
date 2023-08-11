# vi: ft=Dockerfile:

ARG GO_VERSION=1.20

FROM --platform=$TARGETPLATFORM golang:$GO_VERSION AS go

RUN apt-get update && apt-get dist-upgrade -y && apt-get install -y build-essential git

ARG GOPRIVATE="github.com/artex-io/*,gitlab.com/modulus.io/*"
ARG GOPRIVATE_GITHUB_USER=""
ARG GOPRIVATE_GITHUB_TOKEN=""

WORKDIR $GOPATH/src/sylr.dev/fix

RUN if [ -n "${GOPRIVATE_GITHUB_TOKEN}" ]; then \
        if [ -n "${GOPRIVATE_GITHUB_USER}" ]; then \
            git config --global "url.https://${GOPRIVATE_GITHUB_USER}:${GOPRIVATE_GITHUB_TOKEN}@github.com/.insteadOf" "https://github.com/"; \
        else \
            git config --global "url.https://${GOPRIVATE_GITHUB_TOKEN}@github.com/.insteadOf" "https://github.com/"; \
        fi; \
    fi;

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# -----------------------------------------------------------------------------

FROM --platform=$TARGETPLATFORM go AS builder

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ARG GIT_REVISION
ARG GIT_VERSION
ARG GO_BUILD_EXTLDFLAGS
ARG GO_BUILD_LDFLAGS_OPTIMS
ARG GO_BUILD_TAGS=acceptor,validator

# Switch shell to bash
SHELL ["bash", "-c"]

RUN make build \
    GIT_REVISION=${GIT_REVISION} \
    GIT_VERSION=${GIT_VERSION} \
    GIT_UPDATE_INDEX=${GIT_UPDATE_INDEX} \
    GOPRIVATE_GITHUB_USER=${GOPRIVATE_GITHUB_USER} \
    GOPRIVATE_GITHUB_TOKEN=${GOPRIVATE_GITHUB_TOKEN} \
    GO_BUILD_TAGS="${GO_BUILD_TAGS}" \
    GO_BUILD_FLAGS="${GO_BUILD_FLAGS}" \
    GO_BUILD_EXTLDFLAGS="${GO_BUILD_EXTLDFLAGS}" \
    GO_BUILD_LDFLAGS_OPTIMS="${GO_BUILD_LDFLAGS_OPTIMS}" \
    GO_BUILD_TARGET=dist/${TARGETPLATFORM}/fix \
    GO_BUILD_TAGS=${GO_BUILD_TAGS} \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT/v/} \
    GO_BUILD_TARGET=dist/${TARGETPLATFORM}/fix

RUN useradd fix --create-home

USER fix:fix

RUN /go/src/sylr.dev/fix/dist/$TARGETPLATFORM/fix init config && \
    /go/src/sylr.dev/fix/dist/$TARGETPLATFORM/fix init database

# -----------------------------------------------------------------------------

FROM scratch

ARG TARGETPLATFORM

WORKDIR /usr/local/bin

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/services /etc/services
COPY --from=builder "/go/src/sylr.dev/fix/dist/$TARGETPLATFORM/fix" .
COPY --from=builder /home/fix /home/fix

USER fix:fix

CMD ["/usr/local/bin/fix"]
