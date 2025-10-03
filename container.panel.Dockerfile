FROM --platform=amd64 node:lts-slim as web-build

ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
ENV TSC_COMPILE_CACHE=/tmp/tsbuildcache
ENV NEXT_CACHE_DIR=/tmp/tsbuildcache

RUN corepack enable && corepack prepare pnpm@10.17.0 --activate
COPY web /build

RUN rm -f /build/.env.local
RUN rm -rf /build/out

RUN \
    --mount=type=cache,id=pnpm,target=/pnpm/store \
    --mount=type=cache,id=tscache,target=/tmp/tsbuildcache \
    cd /build && \
    pnpm install --frozen-lockfile && \
    pnpm build

FROM --platform=amd64 golang:1.24.0 as build

RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin/

RUN mkdir -p /build

# build
ENV CGO_ENABLED=0 GOPATH=/go
RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache \
    --mount=target=/build/,type=bind,source=. \
    cd /build && just install

# copy web assets
COPY --from=web-build /build/out /www
