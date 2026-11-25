狀態：草案／待批准

# ADR-013：部署改 Docker Compose＋Coolify＋OpenTofu

- **狀態**：已決定（草案／待批准）；取代 ADR-002 的部署部分
- **日期**：2026-11-25 00:33（使用者架構調整指令）
- **依據**：SPEC-SOURCE.md 頂部「架構調整」一節、手冊 C（Phase 01 基礎建設）

## 問題

部署應如何改採 Docker Compose＋Coolify＋OpenTofu＋GitHub Actions，同時滿足手冊 C 的環境劃分與 CI 要求？

## 選項

| 選項 | 說明 |
|---|---|
| A. Docker Compose（組合單位）＋Coolify（部署管理）＋OpenTofu（基礎設施宣告）＋GitHub Actions（CI／CD） | 符合架構調整指令；本機／單主機／staging／production 同一組合語言 |
| B. 維持原部署方向 | 已被使用者指令取代（見 ADR-002 已取代狀態），不得繼續 |
| C. 只用 Docker Compose，不用 Coolify／OpenTofu | 部署管理與基礎設施宣告無落實處；多環境一致性無法保證 |

## 決定

選 A：

- **Docker Compose**：本機（local）與單主機部署的組合單位；全部服務（API、Gateway、Worker、PostgreSQL、Valkey、NATS、MinIO、Meilisearch、Keycloak、OpenBao、LiveKit CE、Prometheus／Grafana／Loki／GlitchTip、Traefik、Coraza）的組合定義皆以 Compose 檔案為準；版本固定，不得用浮動標籤上 production。
- **Coolify**：容器部署管理平面（部署、重啟、回滾、環境變數注入的介面）；Coolify 本身的部署與備份為維運要求。
- **OpenTofu**：基礎設施即程式碼（主機、網路、DNS 指向、憑證的宣告）；狀態檔不得含 Secrets（Secrets 經 OpenBao）。
- **GitHub Actions**：CI／CD（建置、測試、靜態檢查、依賴掃描、容器映像建置與推送、部署觸發）；iOS 建置走 macOS runner（見 ADR-003）。

## 環境劃分（手冊 C）

local／development／staging／production 四環境，設定差異只以環境變數與 OpenBao 注入表達；staging 與 production 用同一部署流程；staging 驗收未通過不得部署 production；Production Secrets 絕對不入 Git。細節見 `SYSTEM_ARCHITECTURE.md` §6。

## 理由

1. 使用者指令明定此組合；Docker Compose 是本機開發與自架部署的共同語言，Coolify 與 OpenTofu 分別補齊「部署管理」與「基礎設施宣告」，三者缺一即部署流程不完整。
2. 手冊 C 的全部驗收（專案可建置、Docker 啟動、API 健康檢查、App／Web／Admin 啟動、PG／Valkey／NATS 連線、CI 自動檢查、無硬編碼憑證）皆以此組合為驗證載體——其中「Docker 啟動」與連線驗證，皆為「待 CI 或有 Docker 的環境驗證」（本機 VM 無 docker，見 ADR-008）。

## 後果

- 部署文件（Phase 01／Phase 11）須以本 ADR 的組合為準，不得另起一套部署語言。
- 回滾流程、備份還原演練、主機規格與成本，皆待查證（伺服器與網域為仍可能付費項目，見 ADR-009）。
- 本決定不代使用者決定花錢或綁卡：主機與網域的開通與付款，皆須使用者決定。
