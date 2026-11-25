// Package search 是 Search（搜尋）模組的訊息搜尋實作（Phase 06
// 後續里程碑，離線可驗證部分）。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.14）：跨範圍搜尋、訊息搜尋
// （關鍵字、日期區間、發送者、媒體／檔案類型）。結果須遵守存取
// 權限與封鎖規則（搜尋前後皆須過濾，不得先回傳再過濾）。無自有
// PostgreSQL 表，擁有 Meilisearch 內的搜尋索引投影；索引遺失須
// 可由 PostgreSQL 重建，不得以索引反寫業務表，不得被其他業務
// 模組依賴。向量索引歸屬待定，本模組不得預設擁有向量索引。
//
// 本里程碑範圍（僅訊息）：MemoryIndex（記憶體索引，開發與測試用）、
// MeilisearchIndex（Meilisearch 轉接器，見 meilisearch.go；未設定
// 時回 ErrNotConfigured）、Projector（事件驅動投影，消費
// message.received／message.recalled，冪等）、Consumer（NATS 訂閱
// 消費迴圈，見 consumer.go）與 Service（查詢＋權限過濾＋游標分頁）
// 及 HTTP 端點 GET /api/v1/search/messages。其他搜尋範圍（好友、
// 使用者、群組、社群、官方帳號、檔案、貼圖、動態）不在範圍。
//
// 索引同步（DATA_FLOW.md §8）：事件驅動投影，不得由業務寫入路徑
// 同步寫索引；填充迴圈（api 行程內消費迴圈依環境開關啟動，見
// cmd/api/main.go；Worker 行程接線待模組重構）與 Meilisearch
// 轉接器已落地。編輯無對應事件名，編輯後內容不經事件更新索引
// （缺口見 STATUS.md 待決定 23）。
//
// 依賴方向：只經公開介面（MembershipChecker 由 main.go 以
// chat.Service 接線）；不得直讀他模組資料表，不得 import 他模組
// 的內部套件，不得寫入任何業務表。
package search
