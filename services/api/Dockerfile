# ETHER API Dockerfile（Phase 01 骨架）
#
# 狀態：草稿，待 CI 或有 Docker 的環境驗證。本機 VM 無 docker／go，尚未建置過。
# 基礎映像版本於 2026-11-25 固定（見下）；CI 首次實跑時須覆核該版本是否仍受支援，
# 升級須經審查，不得改為浮動標籤。
# 建置 context：docker-compose.yml 的 api 服務為 context ../../services/api
#（相對於 infrastructure/docker/，即本目錄）、dockerfile Dockerfile，與本檔案一致。
# 程式碼骨架已由程式碼軌並行建立（go.mod go 1.25、cmd/api、internal/health
# 的 /healthz liveness handler；僅標準庫依賴、go.sum 為空檔，見建置階段註解），
# 編譯未驗證。建置路徑已對照實際 cmd 佈局，仍待 CI 建置驗證。
# Healthcheck 路徑與 docker-compose.yml 的 api healthcheck 一致：
# wget http://localhost:8080/healthz（compose 執行時以 compose 的 healthcheck 為準，
# 本檔案的 HEALTHCHECK 為獨立建置／執行時的對照，不得改路徑）。

# ---- 建置階段 ----
FROM golang:1.25.0-alpine3.21 AS builder

WORKDIR /src

# 只複製模組定義檔，先下載依賴（快取層），再複製程式碼。
# go.sum 自 M2b 起為真實檔案（lib/pq v1.12.3，PostgreSQL 驅動）：
# 本檔案檔頭原註「新增外部依賴時須產生真實 go.sum 並補回 COPY」，
# 此處即依該註補回 COPY go.sum。migrations/ 須一併複製：
# migration SQL 由 Go embed 內嵌（migrations.FS），cmd/migrate
# 與 api 皆依賴內嵌檔案，缺失時建置失敗。
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# 只複製必要檔案（其餘由 .dockerignore 排除，不得複製 .env／憑證／測試假資料）。
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- 執行階段 ----
FROM alpine:3.21.3

RUN addgroup -S ether && adduser -S ether -G ether -u 10001 \
    && apk add --no-cache ca-certificates
# wget 由 busybox 提供，供 HEALTHCHECK 使用；不另裝套件。

WORKDIR /app
COPY --from=builder --chown=ether:ether /out/api /app/api
COPY --from=builder --chown=ether:ether /out/migrate /app/migrate

USER ether:ether
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/api"]
