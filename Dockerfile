# Build assets
FROM --platform=$BUILDPLATFORM node:25.8.1-alpine AS node

RUN npm install -g --force corepack && corepack enable

WORKDIR /build

# Install dependencies from lock file
COPY pnpm-*.yaml ./
RUN pnpm fetch --ignore-scripts --no-optional

# Copy package.json and install dependencies
COPY package.json ./
RUN pnpm install --offline --ignore-scripts --no-optional

# Copy assets and translations to build
COPY vite.config.ts tsconfig.json .prettierrc.cjs .npmrc ./
COPY assets ./assets
COPY locales ./locales
COPY public ./public

# Build assets
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git && mkdir /dozzle

WORKDIR /dozzle

ENV GOPROXY=direct
ENV GOSUMDB=off

# Copy go mod files
COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy all other files
COPY internal ./internal
COPY types ./types
COPY main.go ./
COPY protos ./protos
COPY shared_key.pem shared_cert.pem ./

# Copy assets built with node
COPY --from=node /build/dist ./dist

# Args
ARG TAG=v10.1.2
ARG TARGETOS TARGETARCH

# Build binary
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -mod=mod -ldflags "-s -w -X github.com/amir20/dozzle/internal/support/cli.Version=$TAG" -o dozzle

RUN mkdir /data

FROM --platform=$BUILDPLATFORM aquasec/trivy:0.67.2 AS trivy

ENV TRIVY_CACHE_DIR=/tmp/trivy-cache
ENV TRIVY_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-db,public.ecr.aws/aquasecurity/trivy-db
ENV TRIVY_JAVA_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-java-db,public.ecr.aws/aquasecurity/trivy-java-db

RUN trivy image --download-db-only --no-progress

FROM scratch

ENV PATH=/usr/local/bin
ENV TRIVY_CACHE_DIR=/tmp/trivy-cache
ENV TRIVY_DB_REPOSITORY="ghcr.io/aquasecurity/trivy-db,public.ecr.aws/aquasecurity/trivy-db"
ENV TRIVY_JAVA_DB_REPOSITORY="ghcr.io/aquasecurity/trivy-java-db,public.ecr.aws/aquasecurity/trivy-java-db"

COPY --from=builder /data /data
COPY --from=builder /tmp /tmp
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle
COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy
COPY --from=trivy /tmp/trivy-cache /tmp/trivy-cache

EXPOSE 8080

ENTRYPOINT ["/dozzle"]
