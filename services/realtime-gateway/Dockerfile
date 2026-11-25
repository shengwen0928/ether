# ETHER Realtime Gateway Dockerfile（Phase 01 骨架）
#
# 狀態：草稿，待 CI 或有 Docker 的環境驗證。本機 VM 無 docker／go，尚未建置過。
# 基礎映像版本於 2026-11-25 固定（見下）；CI 首次實跑時須覆核該版本是否仍受支援，
# 升級須經審查，不得改為浮動標籤。
#
# 對齊狀態：G-01 已裁決 A，compose 已改指本檔案，見 ADR-015。
# docker-compose.yml 的 realtime-gateway 服務現為 context
# ../../services/realtime-gateway、dockerfile Dockerfile（即本檔案），獨立
# context 已生效。原「對齊缺口」敘述已成歷史，見
# docs/phase01/compose-alignment-notes.md。
# 程式碼骨架已由程式碼軌並行建立（go.mod go 1.25、cmd/realtime-gateway、
# internal/health 的 /healthz liveness handler；僅標準庫依賴、go.sum 為空檔），
# 編譯未驗證。建置路徑已對照實際 cmd 佈局（cmd 名為 realtime-gateway，非
# gateway），仍待 CI 建置驗證。
# Healthcheck 路徑與 docker-compose.yml 的 realtime-gateway healthcheck 一致：
# wget http://localhost:8081/healthz（compose 執行時以 compose 的 healthcheck 為準，
# 本檔案的 HEALTHCHECK 為獨立建置／執行時的對照，不得改路徑）。

# ---- 建置階段 ----
FROM golang:1.25.0-alpine3.21 AS builder

WORKDIR /src

# 只複製模組定義檔，先下載依賴（快取層），再複製程式碼。
# go.sum 為空檔（零外部依賴）：三個服務的 go.mod 目前皆無外部依賴（僅標準庫），
# 不 COPY 空的 go.sum；新增外部依賴時須產生真實 go.sum 並補回 COPY。
COPY go.mod ./
RUN go mod download

# 只複製必要檔案（其餘由 .dockerignore 排除，不得複製 .env／憑證／測試假資料）。
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/realtime-gateway ./cmd/realtime-gateway

# ---- 執行階段 ----
FROM alpine:3.21.3

RUN addgroup -S ether && adduser -S ether -G ether -u 10001 \
    && apk add --no-cache ca-certificates
# wget 由 busybox 提供，供 HEALTHCHECK 使用；不另裝套件。

WORKDIR /app
COPY --from=builder --chown=ether:ether /out/realtime-gateway /app/realtime-gateway

USER ether:ether
EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://localhost:8081/healthz || exit 1

ENTRYPOINT ["/app/realtime-gateway"]
