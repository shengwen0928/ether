// Package jobs 是背景任務定義的佔位文件。
//
// 本階段沒有任何任務定義，不得寫假任務處理。NATS JetStream 消費者
// （推播扇出、搜尋索引同步、媒體縮圖／轉碼、Bot Webhook 投遞、
// 重試與死信處理，見 SYSTEM_ARCHITECTURE.md §3.2 Worker 一節與
// DATA_FLOW.md §5／§7／§8／§9）待 Phase 05 起逐項落地；消費須冪等，
// 失敗走 NATS 重試與 DLQ，不得阻塞訊息寫入路徑，不得無聲丟棄。
// 任務的 Outbox／事件來源與重試策略，待所屬階段定稿。
package jobs
