狀態：草案／待批准

# ADR-006：WebSocket＋NATS 即時架構（vs 純 WebSocket／Supabase Realtime）

- **狀態**：已決定（草案／待批准）
- **日期**：2026-11-25（Phase 00）
- **依據**：SPEC-SOURCE.md §21、§22、手冊 G

## 問題

即時訊息傳輸應採何種架構，才能處理斷網、重連、重複、遺失、亂序、離線訊息、多裝置同步、伺服器重啟、快速切網、App 背景／被終止、推播與即時重複、大量同時上線？

## 選項

| 選項 | 優點 | 缺點 |
|---|---|---|
| A. WebSocket＋NATS JetStream | Gateway 無狀態可水平擴充；Outbox 保證寫入與發布原子性；佇列持久化與重放；跨副本路由 | 須自架 NATS 叢集；消費冪等與 Sequence 須仔細設計 |
| B. 純 WebSocket（無佇列） | 最簡單 | Gateway 重啟即丟未投遞訊息；跨副本路由困難；無重放與 DLQ；不得將 WS 連線本身當可靠送達保證（§22 明文禁止此假設） |
| C. Supabase Realtime | 舊路線現成 | 已被 ETHER 規格取代（見 ADR-005）；且為外部託管，與免費開源自架方向（ADR-009）衝突 |

## 決定

選 A：WebSocket（用戶端 ↔ Realtime Gateway）＋NATS JetStream（Gateway／Worker 間的事件與佇列平面）＋Transactional Outbox（PostgreSQL 內，業務寫入與事件發布同一交易）。事件名以 SPEC-SOURCE.md §21 為準；可靠性機制為 Message ID、Idempotency Key、Sequence Number、ACK、Retry、Exponential Backoff、Message Persistence、Offline Queue、Delivery Receipt、Reconnection Recovery（§22）。

## 理由

1. §22 的全部異常情境，皆須「持久化＋重放＋去重」才能處理，純 WebSocket 無持久化層，無法滿足。
2. Outbox 保證「寫入成功即事件必達佇列」，NATS 保證「至少一次投遞」，消費端冪等保證「重放不重複」——三者組合是本架構一致性保證總表的基礎（見 `DATA_FLOW.md` §10）。
3. Gateway 無狀態化是水平擴充的前提（見 `SCALABILITY.md` §9）；NATS Consumer Group 讓 Worker 水平擴充與 Gateway 擴充解耦。

## 後果

- Gateway 不得自行寫入 `messages`，寫入路徑唯一（見 `DATA_FLOW.md` §3）。
- NATS 自架叢集的保留、備份與故障演練為維運要求（見 `SCALABILITY.md` §4）。
- 信令（SDP／ICE）走 WebSocket 即時通道，不得走 NATS（見 `DATA_FLOW.md` §6）。
