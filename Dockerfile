# syntax=docker/dockerfile:1.2
FROM golang:1.15.6-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src/hkrelay
COPY go.mod .
COPY go.sum .

RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
    --mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
        go mod download -x

COPY . .

RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
    --mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
        CGOENABLED=0 go build -o /bin/hkrelay

FROM alpine:3.12.3 AS runtime

COPY --from=build /bin/hkrelay /bin/hkrelay

CMD ["/bin/hkrelay"]
