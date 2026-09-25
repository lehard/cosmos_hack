# Образ ant: многоступенчатая сборка только в контейнерах (AD-25, FR-109).
# Go и Node на машине не нужны. Фронтенд встраивается в бинарник.
#
# Офлайн (NFR-SEC-1): Go-зависимости — backend/vendor; npm — из
# frontend/.npm-cache (make npm-cache), иначе из сети при первой сборке.
# Базовые образы закреплены версиями и загружаются заранее.

ARG GO_IMAGE=golang:1.27.1
ARG NODE_IMAGE=node:24.21.0-slim
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian12:nonroot

# ---------------------------------------------------------------- фронтенд --
FROM ${NODE_IMAGE} AS frontend
WORKDIR /src/frontend
ENV npm_config_update_notifier=false npm_config_fund=false npm_config_audit=false
COPY frontend/package.json frontend/package-lock.json frontend/.npmrc ./
RUN --mount=type=bind,source=frontend/.npm-cache,target=/npm-seed \
    --mount=type=cache,target=/npm-cache,id=ant-npm \
    cp -a /npm-seed/. /npm-cache/ \
    && npm ci --cache /npm-cache --prefer-offline
COPY frontend/ ./
RUN npm run build

# ----------------------------------------------------------------- бэкенд --
FROM ${GO_IMAGE} AS backend
ARG VERSION=dev
WORKDIR /src/backend
ENV CGO_ENABLED=0 GOFLAGS="-mod=vendor -buildvcs=false" GOTOOLCHAIN=local GOPROXY=off
COPY third_party/ /src/third_party/
COPY backend/ ./
COPY --from=frontend /src/frontend/dist/ ./internal/infrastructure/transport/webui/dist/
RUN --mount=type=cache,target=/root/.cache/go-build,id=ant-gobuild \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/ant ./cmd/ant

# ------------------------------------------------------------ исполнение --
FROM ${RUNTIME_IMAGE}
ARG VERSION=dev
LABEL ant.project="ant" \
      org.opencontainers.image.title="ant" \
      org.opencontainers.image.description="Доверенная система контроля качества деталей" \
      org.opencontainers.image.version="${VERSION}"
COPY --from=backend /out/ant /ant
COPY deploy/config/ant.yaml /etc/ant/ant.yaml
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD ["/ant", "-healthcheck"]
ENTRYPOINT ["/ant"]
CMD ["-role=api"]
