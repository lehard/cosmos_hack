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
# Пакет подписи в браузере (Д-72, вариант page — хранилище страницы): то же
# ядро агента токена в WASM, раздаётся интерфейсом по /signer/ (встроен в ant).
RUN --mount=type=cache,target=/root/.cache/go-build,id=ant-gobuild \
    mkdir -p ./internal/infrastructure/transport/webui/dist/signer \
    && GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w -X ant/cmd/token-agent/internal/agent.Build=${VERSION}" \
         -o ./internal/infrastructure/transport/webui/dist/signer/signer.wasm ./cmd/token-agent/wasm \
    && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./internal/infrastructure/transport/webui/dist/signer/
RUN --mount=type=cache,target=/root/.cache/go-build,id=ant-gobuild \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/ant ./cmd/ant \
    && for c in keeper verifier tamper demo-signer; do \
         go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/$c ./cmd/$c || exit 1; done
# Каталоги томов (materials.dir; хранитель, верификатор, сертификат mTLS ant,
# KEK, экспорт шины безопасности — эпик 29): именованный том при первом
# подключении наследует владельца — пользователя исполнения 65532.
RUN mkdir -p /out/var/lib/ant/materials /out/var/lib/keeper /out/var/lib/verifier /out/var/lib/ant-pki \
    /out/var/lib/ant-kek /out/var/lib/ant-export /out/var/lib/ant-tamper

# ------------------------------------------------------------ исполнение --
FROM ${RUNTIME_IMAGE}
ARG VERSION=dev
LABEL ant.project="ant" \
      org.opencontainers.image.title="ant" \
      org.opencontainers.image.description="Доверенная система контроля качества деталей" \
      org.opencontainers.image.version="${VERSION}"
COPY --from=backend /out/ant /ant
# Отдельные процессы по границам доверия (эпик 29, AD-8, AD-9, AD-28): хранитель,
# верификатор и демо-инструмент подделки — в том же образе, свои службы compose.
COPY --from=backend /out/keeper /out/verifier /out/tamper /
# demo-signer — демо-персоны сценариев (AD-26, Д-59): своя служба compose,
# перечень персон — стартовая политика нормативного слоя.
COPY --from=backend /out/demo-signer /demo-signer
COPY normative/policy/ /etc/ant/normative/policy/
COPY --from=backend --chown=65532:65532 /out/var/lib/ /var/lib/
COPY deploy/config/ant.yaml /etc/ant/ant.yaml
# Определения и ожидания тестовых сценариев (AD-26): пульт сценариев роли
# stands (stands.scenarios); потоки JSONL генератор строит сам.
COPY scenarios/definitions/ /etc/ant/scenarios/definitions/
COPY scenarios/expected/ /etc/ant/scenarios/expected/
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD ["/ant", "-healthcheck"]
ENTRYPOINT ["/ant"]
CMD ["-role=api"]
