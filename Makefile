# ETHER Monorepo Makefile（Phase 01 骨架）
#
# 驗證狀態（2026-11-25，本機 VM 無 go／flutter／docker，皆為「待 CI 或有 Docker 的
# 環境驗證」，不得宣稱已在本機驗證）：
#   build        待 CI／有 Docker 的環境驗證。需 docker；程式碼與 compose 對齊
#                完成前預期失敗，失敗輸出不得靜默吞掉。
#   lint         待 CI 驗證。需 go；services/api/go.mod 尚未建立，目前會以明確
#                訊息退出。Flutter 部分只在 apps/mobile 存在時檢查。
#   test         待 CI 驗證。需 go；目前為單元測試層，整合測試需 Docker
#                （見 docs/testing/TEST_STRATEGY.md §3.2）。
#   migrate      佔位檢查，待定稿。Migration 執行者與工具待定稿
#                （見 docs/deployment/SELF_HOSTING.md §3），本目標目前不執行
#                任何 Migration，只做前置檢查並以明確訊息退出。
#   compose-up   待有 Docker 的環境驗證。需 docker；livekit.yaml／prometheus.yml
#                尚未建立、應用程式碼尚無，啟動預期不完整。
#   verify-static  本機可執行（只用 python3＋grep，PyYAML 已確認可用）。
#                只做靜態檢查：YAML 解析、Dockerfile 指令、引用一致性、
#                secrets 掃描。靜態通過不等於建置／啟動驗證。
# 缺少工具鏈的目標一律先檢查工具是否存在，不存在時印出明確訊息並以非零碼
# 退出，不得靜默失敗、不得以跳過假裝成功。

.RECIPEPREFIX = >

COMPOSE_FILE := infrastructure/docker/docker-compose.yml

.PHONY: build lint test migrate compose-up verify-static

build:
> @command -v docker >/dev/null 2>&1 || { echo "錯誤：本機找不到 docker，無法建置映像。請在有 Docker 的環境或 CI 執行 make build。"; exit 1; }
> @docker compose version >/dev/null 2>&1 || { echo "錯誤：找不到 docker compose 外掛（docker compose version 失敗），無法建置。請安裝 Docker Compose v2 後重試。"; exit 1; }
> @echo "注意：compose 對齊缺口見 docs/phase01/compose-alignment-notes.md；程式碼尚無，建置預期在對齊／程式碼完成前失敗。"
> docker compose -f $(COMPOSE_FILE) build api realtime-gateway worker

lint:
> @command -v go >/dev/null 2>&1 || { echo "錯誤：本機找不到 go，無法執行 Go lint。請安裝 Go 工具鏈或在 CI 執行 make lint。"; exit 1; }
> @test -f services/api/go.mod || { echo "錯誤：services/api/go.mod 尚未建立（尚無 Go 程式碼），無可 lint 的模組。請待程式碼軌建立模組後重試。"; exit 1; }
> @test -z "$$(cd services/api && gofmt -l cmd internal 2>/dev/null)" || { echo "錯誤：gofmt 發現格式不一致的檔案（見上列清單）。"; exit 1; }
> @(cd services/api && go vet ./...)
> @if [ -d apps/mobile ]; then command -v flutter >/dev/null 2>&1 || { echo "錯誤：apps/mobile 已存在但本機找不到 flutter，無法執行 Flutter lint。請安裝 Flutter 或在 CI 執行。"; exit 1; }; (cd apps/mobile && flutter analyze); else echo "提示：apps/mobile 尚未建立，跳過 Flutter lint（骨架缺口，非靜默跳過）。"; fi

test:
> @command -v go >/dev/null 2>&1 || { echo "錯誤：本機找不到 go，無法執行測試。請安裝 Go 工具鏈或在 CI 執行 make test。"; exit 1; }
> @test -f services/api/go.mod || { echo "錯誤：services/api/go.mod 尚未建立（尚無 Go 程式碼），無可測試的模組。請待程式碼軌建立模組後重試。"; exit 1; }
> @echo "注意：本目標只跑單元測試；整合測試需 Docker Compose 啟動真實依賴，見 docs/testing/TEST_STRATEGY.md §3.2，待 CI 驗證。"
> @(cd services/api && go test ./...)

migrate:
> @command -v docker >/dev/null 2>&1 || { echo "錯誤：本機找不到 docker，無法執行 Migration 前置檢查。請在有 Docker 的環境或 CI 執行。"; exit 1; }
> @test -d services/api/migrations || { echo "錯誤：services/api/migrations 尚未建立，且 Migration 執行者與工具待定稿（見 docs/deployment/SELF_HOSTING.md §3）。本目標不執行任何 Migration。"; exit 1; }
> @echo "錯誤：Migration 執行步驟待定稿，本目標目前只做前置檢查，不執行任何 Migration。"
> @exit 1

compose-up:
> @command -v docker >/dev/null 2>&1 || { echo "錯誤：本機找不到 docker，無法啟動 Compose。請在有 Docker 的環境執行 make compose-up。"; exit 1; }
> @docker compose version >/dev/null 2>&1 || { echo "錯誤：找不到 docker compose 外掛（docker compose version 失敗），無法啟動。請安裝 Docker Compose v2 後重試。"; exit 1; }
> @test -f .env || echo "警告：根目錄無 .env，Compose 將使用 docker-compose.yml 內的預設佔位符；請先複製 .env.example 為 .env 並填入各環境實際值（不得提交 .env）。"
> @echo "注意：請按 docs/deployment/SELF_HOSTING.md §3 分組檢查健康狀態；livekit.yaml／prometheus.yml 尚未建立、應用程式碼尚無，完整啟動待 CI 或有 Docker 的環境驗證。"
> docker compose -f $(COMPOSE_FILE) up -d

verify-static:
> @echo "== verify-static：只用 python3＋grep 的靜態檢查（不等於建置／啟動驗證） =="
> @python3 -c "import yaml; d=yaml.safe_load(open('$(COMPOSE_FILE)')); svcs=d.get('services', {}); assert len(svcs)==17, sorted(svcs); print('compose services: 17 OK:', ', '.join(sorted(svcs)))"
> @test -f services/api/Dockerfile || { echo "FAIL: services/api/Dockerfile 不存在"; exit 1; }
> @test -f services/realtime-gateway/Dockerfile || { echo "FAIL: services/realtime-gateway/Dockerfile 不存在"; exit 1; }
> @test -f services/worker/Dockerfile || { echo "FAIL: services/worker/Dockerfile 不存在"; exit 1; }
> @grep -q '^FROM golang:1.25.0-alpine3.21' services/api/Dockerfile && grep -q '^FROM golang:1.25.0-alpine3.21' services/realtime-gateway/Dockerfile && grep -q '^FROM golang:1.25.0-alpine3.21' services/worker/Dockerfile || { echo "FAIL: Dockerfile 建置階段基礎映像未固定為 golang:1.25.0-alpine3.21"; exit 1; }
> @grep -q '^FROM alpine:3.21.3' services/api/Dockerfile && grep -q '^FROM alpine:3.21.3' services/realtime-gateway/Dockerfile && grep -q '^FROM alpine:3.21.3' services/worker/Dockerfile || { echo "FAIL: Dockerfile 執行階段基礎映像未固定為 alpine:3.21.3"; exit 1; }
> @grep -q '^USER ether' services/api/Dockerfile && grep -q '^USER ether' services/realtime-gateway/Dockerfile && grep -q '^USER ether' services/worker/Dockerfile || { echo "FAIL: Dockerfile 未以非 root 使用者 ether 執行"; exit 1; }
> @! grep -qE '^USER (root|0)([: ]|$$)' services/api/Dockerfile services/realtime-gateway/Dockerfile services/worker/Dockerfile || { echo "FAIL: Dockerfile 含 root 執行"; exit 1; }
> @grep -q '^HEALTHCHECK' services/api/Dockerfile && grep -q 'http://localhost:8080/healthz' services/api/Dockerfile || { echo "FAIL: api Dockerfile HEALTHCHECK 路徑與 compose 不一致（應為 8080/healthz）"; exit 1; }
> @grep -q '^HEALTHCHECK' services/realtime-gateway/Dockerfile && grep -q 'http://localhost:8081/healthz' services/realtime-gateway/Dockerfile || { echo "FAIL: gateway Dockerfile HEALTHCHECK 路徑與 compose 不一致（應為 8081/healthz）"; exit 1; }
> @! grep -q '^HEALTHCHECK' services/worker/Dockerfile || { echo "FAIL: worker Dockerfile 不應自創 HEALTHCHECK（compose 未定義 worker healthcheck）"; exit 1; }
> @grep -qx '\.env' .gitignore || { echo "FAIL: .gitignore 未忽略 .env"; exit 1; }
> @grep -q '^\!\.env\.example' .gitignore || { echo "FAIL: .gitignore 未以否定規則保留 .env.example"; exit 1; }
> @test ! -e .github || { echo "FAIL: .github 目錄不得存在（CI 草稿只放 infrastructure/ci-drafts/）"; exit 1; }
> @! grep -qE 'sk-[A-Za-z0-9]|ghp_|gho_|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY|-----BEGIN [A-Z ]*KEY-----' .env.example || { echo "FAIL: .env.example 含疑似真實金鑰樣式的值"; exit 1; }
> @echo "-- 已知缺口（警告，不計入失敗；見 docs/phase01/compose-alignment-notes.md） --"
> @python3 -c "import yaml; d=yaml.safe_load(open('$(COMPOSE_FILE)')); b={k: (v.get('build') or {}) for k,v in d['services'].items() if 'build' in v}; print('compose build refs:', b); print('WARNING: gateway/worker 的 compose build context 與獨立 Dockerfile 路徑不一致，待協調員裁決' if b.get('realtime-gateway',{}).get('context')!='../../services/realtime-gateway' or b.get('worker',{}).get('context')!='../../services/worker' else 'OK: compose build refs 與 Dockerfile 路徑一致')"
> @test -f infrastructure/docker/livekit.yaml || echo "提示：infrastructure/docker/livekit.yaml 未建立是決議預期（livekit 以 profiles: [\"phase07\"] 保留定義、預設不啟動，見 SELF_HOSTING.md §9）"
> @test -f infrastructure/docker/prometheus.yml || echo "WARNING: infrastructure/docker/prometheus.yml 尚未建立（§4 方案要求最小有效設定已建立，見 SELF_HOSTING.md §9）"
> @test -f services/api/go.mod || echo "WARNING: services/api/go.mod 尚未建立（尚無 Go 程式碼，Dockerfile 建置路徑為骨架假設）"
> @echo "verify-static: PASS（靜態檢查通過；警告缺口見上列，建置／啟動仍待 CI 或有 Docker 的環境驗證）"
