狀態：草案／待批准

# ADR-011：LiveKit Community Edition 自架與 TURN 頻寬／公網部署評估

- **狀態**：已決定（草案／待批准）；TURN 評估待進行
- **日期**：2026-11-25 00:33（使用者架構調整指令）
- **依據**：SPEC-SOURCE.md 頂部「架構調整」一節、§10（通話）、手冊 I（Phase 07）

## 問題

通話媒體面應如何改採 LiveKit Community Edition 自架，同時保留全部通話功能？TURN 頻寬與公網部署需求如何評估？

## 選項

| 選項 | 說明 |
|---|---|
| A. LiveKit CE 自架 SFU＋自架 STUN／TURN | 符合架構調整指令；維運與頻寬責任自負 |
| B. LiveKit Cloud（託管） | 已被使用者指令取代，不得繼續 |
| C. 自研 SFU／媒體伺服器 | 不可行：媒體轉發的正確性與品質無法以起步成本保證；且與「採成熟方案」原則衝突 |

## 決定

選 A：通話採 LiveKit CE 自架。保留一對一／多人語音視訊、螢幕分享、房間與參與者管理、STUN／TURN、SFU、通話品質監控；信令仍走 Realtime Gateway（WebSocket），不得併入 LiveKit；通話紀錄寫入仍經 API（`calls`、`call_participants`、`call_events`）。

## TURN 頻寬與公網部署評估（特別評估項，Phase 07 前完成）

1. **公網需求**：LiveKit CE 與 TURN 須有公網可達的地址與埠口（UDP／TCP 範圍待 LiveKit CE 官方文件查證，本文件不編造埠號）；媒體 UDP 流量直達 LiveKit／TURN，不得經 Traefik 轉發；部署拓撲（單主機或獨立容器組）待部署文件定案。
2. **TURN 頻寬**：當雙方皆在對稱 NAT 後或 UDP 被擋時，媒體經 TURN 中繼，頻寬消耗隨通話人數、時長與解析度上升；TURN 頻寬列為仍可能付費項目，成本待查證，不得宣稱零成本。
3. **必須做的評估**：(1) 以官方文件確認 TURN 部署的公網 IP、埠口與頻寬需求；(2) 在有公網的測試環境實測一對一與多人通話的 TURN 中繼率與頻寬；(3) 訂出 TURN 中繼率的監控指標與告警（見 ADR-012）；(4) 若頻寬成本超過使用者免費原則可接受範圍，須停下由使用者決定，不得靜默上線後超額。
4. **降級原則**：TURN 不可用時通話建立須明確失敗並提示，不得假裝連線正常；弱網時優先保障語音。

## 理由

1. 使用者指令明定 LiveKit CE 自架且功能全保留；SFU 成熟方案是多人通話品質的前提，自研不可行。
2. TURN 是通話可達性的關鍵：沒有 TURN，中繼情境的通話根本無法建立；TURN 頻寬是自架後最主要的變動成本，必須事前評估，不得事後發現。

## 後果

- LiveKit CE 專節見 `SYSTEM_ARCHITECTURE.md` §9，擴充評估見 `SCALABILITY.md` §10，兩處不得矛盾。
- Docker Compose 啟動與通話驗證皆為「待 CI 或有 Docker 的環境驗證」（本機 VM 無 docker，見 ADR-008）。
- 本決定不代使用者決定花錢或綁卡：TURN 頻寬方案與主機頻寬的選擇，須由使用者決定。
