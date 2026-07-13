FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w" -o /out/short ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/short /short
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/short"]
