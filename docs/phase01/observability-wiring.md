# ETHER 可觀測性接線（Loki＋OpenTelemetry，Phase 01）

狀態：接入點慣例與現況對照，待 CI 或有 Docker 的環境驗證。SDK 接線待後階段，
骨架只留介面與環境變數，不得宣稱監控已生效。

依據：`docs/decisions/ADR-012-可觀測性自架.md`（分工與覆蓋）、
`docs/deployment/CLOUD_ARCHITECTURE.md` §8.2（監控告警）、
`docs/deployment/SELF_HOSTING.md` §7.3（監控驗證清單）、
`docs/testing/TEST_STRATEGY.md` §3.5／§7（壓測監控與 CI 範圍）、
`infrastructure/docker/docker-compose.yml`（現況為準）。

## 1. 日誌格式慣例（JSON、欄位）

各服務輸出結構化日誌（JSON 單行），經 Loki 彙總；日誌不得含敏感資料明文
（不得含 Token／金鑰／E2EE 明文，見 ADR-012 與 SECURITY_ARCHITECTURE.md §8 方向）。

慣例欄位（草案，程式碼軌落地時須對照實作覆核；欄位名未定稿前不得宣稱為契約）：

| 欄位 | 說明 |
|---|---|
| `time`／`timestamp` | 事件時間（RFC 3339，UTC；二選一，待定稿，不得兩種並存不說明） |
| `level` | `debug`／`info`／`warn`／`error`（取值待定稿） |
| `service` | 服務名：`api`／`realtime-gateway`／`worker`（與 compose 服務名一致） |
| `request_id` | 請求識別碼，全鏈路貫穿（CLOUD_ARCHITECTURE.md §8.2 要求含 `request_id`） |
| `trace_id` | OpenTelemetry Trace ID，與 Trace 關聯（SDK 接線後才有，見 §4） |
| `msg`／`message` | 人類可讀訊息（二選一，待定稿） |
| `error` | 錯誤摘要（僅錯誤日誌；不得含敏感資料，不得含堆疊中的憑證） |

- 保留期限、Trace 取樣率、磁碟容量待查證（ADR-012 後果），不得預寫天數與比例。
- 監控數據缺失時報告須標示「數據缺失」，不得以推測數字填補
  （CLOUD_ARCHITECTURE.md §8.2）。

## 2. OTLP Endpoint 環境變數

| 變數 | 現況 | 說明 |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | 保留於 `.env.example` Phase 01 附加段落，停用預設 | 暫定慣例 `http://otel-collector:4317`；Collector 服務尚未在 compose 中定義，服務名與埠號待定稿，不得指向不存在的服務假裝已接線 |
| `OTEL_SERVICE_NAME` | 保留於 `.env.example` Phase 01 附加段落，停用預設 | 佔位符 `ether-api`；各服務的 service name 取值不同（api／realtime-gateway／worker），上行為 api 示例，待後階段定稿 |
| `LOKI_ENDPOINT` | 保留於 `.env.example` Phase 01 附加段落，停用預設（變數名與 `services/api/internal/telemetry/telemetry.go` 的 `LoadConfig` 一致，Phase 01 佔位名，名稱待部署定稿確認） | `http://loki:3100`；`loki` 服務已在 compose 中定義（內部網路），但 Go 端日誌推送／採集方式待定稿，僅保留位址慣例 |
| `SENTRY_DSN` | 保留於 `.env.example` Phase 01 附加段落，停用預設 | GlitchTip 的 Sentry 相容 DSN，一律以環境注入，不得寫入程式碼 |

- Compose 的 `api`／`realtime-gateway`／`worker` 三服務目前**未注入**上述
  變數（compose 的 `environment` 只組裝資料庫／快取／佇列／儲存／搜尋／
  身分位址，見 compose-alignment-notes.md）。變數名保留不等於已接線；
  注入 compose 與否待協調員裁決（不得擅自改 compose）。
- OTLP 傳輸（gRPC／HTTP）、TLS、認證方式待定稿，不得預寫。

## 3. Grafana／Loki／Prometheus 在 Compose 中的現況

以 `docker-compose.yml` 為準（草案，待有 Docker 的環境驗證）：

| 服務 | 映像（草案標籤） | Healthcheck | 現況 |
|---|---|---|---|
| `prometheus` | `prom/prometheus`（未固定標籤） | `wget http://localhost:9090/-/healthy` | 配置檔 `./prometheus.yml` 已建立最小有效設定（僅自我抓取，見 SELF_HOSTING.md §9），其餘抓取目標待定稿 |
| `grafana` | `grafana/grafana`（未固定標籤） | `curl http://localhost:3000/api/health` | 依賴 `prometheus`／`loki` 健康；管理員帳密經 `${GF_SECURITY_ADMIN_*}` 參照 `.env`；儀表板定義以程式碼管理的方式待定稿 |
| `loki` | `grafana/loki`（未固定標籤） | `curl http://localhost:3100/ready` | 以容器內建 `local-config.yaml` 啟動；保留期限待定稿 |
| `glitchtip` | `glitchtip/glitchtip`（未固定標籤） | `wget http://localhost:8000/`（草案，待驗證後定稿） | 錯誤追蹤；錯誤報告不得含敏感資料 |

- 映像標籤未固定，與 CLOUD_ARCHITECTURE.md §3.1「映像版本須固定，不得用
  浮動標籤」的要求不一致，見 compose-alignment-notes.md（缺口 G-04）。
- Grafana 資料來源（Prometheus／Loki）佈建方式、告警規則與管道、儀表板
  定義，皆待定稿（告警管道見 ADR-012：須由使用者決定，不得預設）。

## 4. Phase 01 已接線 vs 待接線清單（誠實標註）

### 已接線（Phase 01 骨架，僅靜態存在）

- [x] Compose 中 `prometheus`／`grafana`／`loki`／`glitchtip` 四服務定義與
      healthcheck 草案（靜態對照完成，啟動未驗證）。
- [x] 環境變數名保留：`OTEL_EXPORTER_OTLP_ENDPOINT`、`OTEL_SERVICE_NAME`、
      `LOKI_ENDPOINT`、`SENTRY_DSN`（`.env.example` 附加段落，停用預設；
      變數名已對照 `services/api/internal/telemetry/telemetry.go` 的
      `LoadConfig`，該檔案為程式碼軌骨架，本軌未撰寫、未驗證編譯）。
- [x] 程式碼骨架中的可觀測性 stub（程式碼軌建立，本軌僅靜態對照）：
      `telemetry` 套件的 `Config`／`Logger`／`NoopLogger`（日誌全部丟棄，
      不得當作已接入監控的證據）、三服務的 `/healthz` handler（僅 liveness，
      依賴連線檢查為 TODO）。SDK 接線仍待後階段。
- [x] 日誌格式欄位慣例（本文件 §1，草案，未實作）。
- [x] `make verify-static` 靜態檢查含 compose 服務數與 Dockerfile 指令檢查，
      不含監控生效檢查（靜態檢查不等於監控驗證）。

### 待接線（後階段，不得宣稱已完成）

- [ ] OpenTelemetry SDK 接入 Go 服務（Trace／Metrics／Logs 統一接入、Trace ID
      貫穿 API／Gateway／Worker／LiveKit CE／Media／AI Adapter，見 ADR-012）。
- [ ] OTel Collector 服務定義（是否獨立容器、部署形態，待定稿；目前 compose
      中不存在）。
- [ ] 日誌採集路徑定稿（Go 端直推 Loki、經 Collector、或 Promtail 等方式，
      待定稿；目前無採集器服務）。
- [x] `prometheus.yml` 最小有效設定已建立（僅自我抓取）。
- [ ] 抓取目標定稿（含 API／WS／NATS／Worker／LiveKit／AI 指標，見 ADR-012 指標清單）。
- [ ] Grafana 資料來源佈建、儀表板定義（程式碼管理）、告警規則與管道定稿。
- [ ] GlitchTip 前端／後端錯誤上報接線（前端接入方式待定）。
- [ ] 保留策略（Trace 取樣率、指標／日誌保留期限）查證與定稿。
- [ ] 全部接線的啟動驗證（SELF_HOSTING.md §7.3 監控驗證清單逐項勾選留證）。

## 5. 缺口（只列、不補決定）

1. Collector／採集器服務形態、OTLP 傳輸與認證、service name 取值，待定稿。
2. 日誌欄位名（`time`／`timestamp`、`msg`／`message`）二選一，待定稿。
3. Prometheus 配置、Grafana 佈建、告警管道與負責人，待定稿／待使用者決定。
4. 映像標籤固定、保留策略查證，待定稿（見 compose-alignment-notes.md）。
